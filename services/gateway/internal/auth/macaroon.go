package auth

import (
	"context"
	"fmt"
	"strings"
	"time"

	"encoding/base64"
	"encoding/binary"
	"encoding/json"

	"github.com/idlab-discover/kebeng/services/gateway/internal/config"
	"github.com/idlab-discover/kebeng/services/gateway/internal/errors"
	"github.com/idlab-discover/kebeng/services/gateway/internal/message"
	macaroon "gopkg.in/macaroon.v2"
)

// MacaroonSerialize returns a store-compatible serialized representation of the given macaroon
func MacaroonSerialize(m *macaroon.Macaroon) (string, error) {
	marshalled, err := m.MarshalBinary()
	if err != nil {
		return "", err
	}
	encoded := base64.RawURLEncoding.EncodeToString(marshalled)
	return encoded, nil
}

// MacaroonDeserialize returns a deserialized macaroon from a given store-compatible serialization
func MacaroonDeserialize(serializedMacaroon string) (*macaroon.Macaroon, error) {
	var m macaroon.Macaroon
	decoded, err := base64.RawURLEncoding.DecodeString(serializedMacaroon)
	if err != nil {
		return nil, err
	}
	err = m.UnmarshalBinary(decoded)
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func GenerateMacaroon(ctx context.Context, req *message.GenerateMacaroonRequest, snapIDs map[string]bool, macaroonConfig *config.MacaroonConfig) *message.MacaroonResponse {
	el := errors.New()

	// start creating the macaroon
	m, err := macaroon.New(
		[]byte(macaroonConfig.RootKey),
		[]byte(macaroonConfig.RootId),
		macaroonConfig.RootLocation,
		macaroon.V1,
	)
	if err != nil {
		el.Add(errors.InternalServerError, fmt.Sprintf("failed to create macaroon: %v", err))
		return &message.MacaroonResponse{Errors: *el}
	}

	// add valid since is in a macaroon when request from the canonical snapstore
	validSince := time.Now().Format(time.RFC3339Nano)
	err = m.AddFirstPartyCaveat([]byte(fmt.Sprintf("%s|valid_since|%s", macaroonConfig.RootLocation, validSince)))
	if err != nil {
		el.Add(errors.InternalServerError, fmt.Sprintf("failed to add valid_since caveat: %v", err))
	}

	// TODO: change version maybe? don't know what the version refers to
	// TODO: maybe encode secret in real example it looks encoded
	thirdPartyCaveatID := fmt.Sprintf(`{"version": 1, "secret" : "%s"}`, macaroonConfig.DischargeKey)
	err = m.AddThirdPartyCaveat(
		[]byte(macaroonConfig.DischargeKey),
		[]byte(thirdPartyCaveatID),
		macaroonConfig.ThirdPartyLocation,
	)
	if err != nil {
		el.Add(errors.InternalServerError, fmt.Sprintf("failed to add third party caveat: %v", err))
	}

	// check generated macaroon requesting it from API and was in this format
	// (This aggregates the requested permissions into a single JSON array.)
	if len(req.Permissions) > 0 {
		permsJson, err := json.Marshal(req.Permissions)
		if err != nil {
			el.Add(errors.InternalServerError, fmt.Sprintf("failed to marshal permissions: %v", err))
		} else {
			err = m.AddFirstPartyCaveat([]byte(fmt.Sprintf("%s|acl|%s", macaroonConfig.RootLocation, permsJson)))
			if err != nil {
				el.Add(errors.InternalServerError, fmt.Sprintf("failed to add acl caveat: %v", err))
			}
		}
	}

	// add channels as caveats
	for _, channel := range req.Channels {
		caveat := fmt.Sprintf("%s|channel|%s", macaroonConfig.RootLocation, channel)
		err = m.AddFirstPartyCaveat([]byte(caveat))
		if err != nil {
			el.Add(errors.InternalServerError, fmt.Sprintf("failed to add channel: %v, err:%v", channel, err))
		}
	}

	// add packages as caveats
	// we get the snapIDs instead of 2 different formats => allows consistency when decoding macaroon
	for snapID := range snapIDs {
		caveat := fmt.Sprintf("%s|snap_id|%s", macaroonConfig.RootLocation, snapID)
		err = m.AddFirstPartyCaveat([]byte(caveat))
		if err != nil {
			el.Add(errors.InternalServerError, fmt.Sprintf("failed to add snap_id: %s, err: %v", snapID, err))
		}
	}

	// Determine expiry: use the explicit expires value if provided,
	// otherwise if any permission among the default ones is requested,
	// set expiry to one year from now.
	var expiryTimestamp string
	if req.Expires != "" {
		expiryTimestamp = req.Expires
	} else {
		defaultExpiryPermissions := map[string]bool{
			"edit_account":       true,
			"modify_account_key": true,
			"package_access":     true,
			"store_admin":        true,
			"store_review":       true,
		}
		for _, perm := range req.Permissions {
			if defaultExpiryPermissions[perm] {
				expiryTimestamp = time.Now().AddDate(1, 0, 0).Format(time.RFC3339)
				break
			}
		}
	}
	if expiryTimestamp != "" {
		err = m.AddFirstPartyCaveat([]byte(fmt.Sprintf("%s|expires|%s", macaroonConfig.RootLocation, expiryTimestamp)))
		if err != nil {
			el.Add(errors.InternalServerError, fmt.Sprintf("failed to add expires caveat: %v", err))
		}
	}

	// previous actions have to be successful before proceeding
	if len(*el) > 0 {
		return &message.MacaroonResponse{Errors: *el}
	}

	serializedMacaroon, err := MacaroonSerialize(m)
	if err != nil {
		el.Add(errors.InternalServerError, fmt.Sprintf("failed to serialize macaroon: %v", err))
		return &message.MacaroonResponse{Errors: *el}
	}

	return &message.MacaroonResponse{
		Macaroon: serializedMacaroon,
		Errors:   *el,
	}
}

func ValidateGenerateMacaroonRequest(req *message.GenerateMacaroonRequest, el *errors.ErrorList) {

	// Allowed permissions.
	validPermissions := map[string]struct{}{
		"edit_account":           {},
		"modify_account_key":     {},
		"package_access":         {},
		"package_register":       {},
		"package_push":           {},
		"package_release":        {},
		"package_update":         {},
		"package_metrics":        {},
		"package_manage":         {},
		"package_upload":         {},
		"package_upload_request": {},
		"package_purchase":       {},
	}

	validChannels := map[string]struct{}{
		"edge":      {},
		"beta":      {},
		"candidate": {},
		"stable":    {},
	}

	for _, perm := range req.Permissions {
		if _, ok := validPermissions[perm]; !ok {
			*el = append(*el, map[string]string{
				"code":    errors.InvalidField,
				"message": fmt.Sprintf("permission '%s' is not valid", perm),
			})
		}
	}

	for _, ch := range req.Channels {
		if _, ok := validChannels[ch]; !ok {
			*el = append(*el, map[string]string{
				"code":    errors.InvalidField,
				"message": "channel value cannot be empty",
			})
		}
	}

	// Validate Packages: each package must have either name/series or snap_id.
	// we don't allow both since this might cause internal conflicts if each of them refer to a different package
	for i, pkg := range req.Packages {
		// Case 1: Nothing provided.
		if pkg.Name == "" && pkg.Series == "" && pkg.SnapId == "" {
			*el = append(*el, map[string]string{
				"code":    errors.InvalidField,
				"message": fmt.Sprintf("package at index %d: must specify either name/series or snap_id in format: {'name': 'the-name', 'series': '16'} or {'snap_id': 'some-snap-id-1234'}", i),
			})
			continue
		}
		// Case 2: snap_id is provided.
		if pkg.SnapId != "" {
			// If snap_id is provided, name and series should be empty.
			if pkg.Name != "" || pkg.Series != "" {
				*el = append(*el, map[string]string{
					"code":    errors.InvalidField,
					"message": fmt.Sprintf("package at index %d: cannot provide both snap_id and name/series", i),
				})
			}
			// This package is valid, so we can continue.
			continue
		}
		// Case 3: snap_id is not provided, so name and series must both be provided.
		if pkg.Name == "" || pkg.Series == "" {
			*el = append(*el, map[string]string{
				"code":    errors.InvalidField,
				"message": fmt.Sprintf("package at index %d: both name and series must be provided if snap_id is not specified", i),
			})
		}
	}

	// validate expires: if provided, must be in valid ISO8601 (RFC3339) format.
	if req.Expires != "" {
		if _, err := time.Parse(time.RFC3339, req.Expires); err != nil {
			*el = append(*el, map[string]string{
				"code":    errors.InvalidField,
				"message": fmt.Sprintf("expires field is not a valid ISO8601 timestamp: %v", err),
			})
		}
	}
	return
}

func VerifyAndGetEmail(cfg *config.Config, el *errors.ErrorList, authData string) *string {
	rootS, dischargeS := GetRootMacaroonsFromString(authData)

	root, err := MacaroonDeserialize(rootS)
	if err != nil {
		el.Add(errors.InternalServerError, fmt.Sprintf("failed to deserialize root macaroon: %v", err))
	}

	discharge, err := MacaroonDeserialize(dischargeS)
	if err != nil {
		el.Add(errors.InternalServerError, fmt.Sprintf("failed to deserialize discharge macaroon: %v", err))
	}
	if root == nil || discharge == nil {
		el.Add(errors.Unauthorized, "missing macaroons")
		return nil
	}

	// Verify the root macaroon using the configured root key.
	if err := root.Verify([]byte(cfg.MacaroonConfig.RootKey), func(caveat string) error {
		// check first party caveats for valid_since and expires the rest is checked when extracted and used to know if valid
		// Check if this is a valid_since caveat.
		if strings.HasPrefix(caveat, "valid_since=") {
			ts := strings.TrimPrefix(caveat, "valid_since=")
			t, err := time.Parse(time.RFC3339Nano, ts)
			if err != nil {
				return fmt.Errorf("invalid valid_since format: %v", err)
			}
			// The macaroon becomes valid after t. If now is before t, it's not yet valid.
			if time.Now().Before(t) {
				return fmt.Errorf("macaroon not yet valid: valid since %s", t.Format(time.RFC3339))
			}
			return nil
		}
		// Check if this is an expires caveat.
		if strings.HasPrefix(caveat, "expires=") {
			ts := strings.TrimPrefix(caveat, "expires=")
			t, err := time.Parse(time.RFC3339, ts)
			if err != nil {
				return fmt.Errorf("invalid expires format: %v", err)
			}
			// If the current time is after t, then the macaroon is expired.
			if time.Now().After(t) {
				return fmt.Errorf("macaroon expired on %s", t.Format(time.RFC3339))
			}
			return nil
		}

		return nil
	}, []*macaroon.Macaroon{discharge}); err != nil {
		el.Add(errors.Unauthorized, fmt.Sprintf("failed to verify macaroon: %v", err))
		return nil
	}

	// TODO: think this is bs
	// Check the root caveats for an authorization indicator.
	authorized := false
	for _, cav := range root.Caveats() {
		if string(cav.Id) != "is-authorized-or-whatever" {
			authorized = true
			break
		}
	}
	if !authorized {
		el.Add(errors.Unauthorized, "unauthorized: missing authorization indicator")
		return nil
	}

	// TODO: not sure if the discharge macaroon has an email field in here
	// Should try to test with real snapd to see what they pass and if they pass a macaroon with email or something
	// Extract email from the discharge macaroon caveats.
	var email string
	for _, cav := range discharge.Caveats() {
		caveatStr := string(cav.Id)
		if strings.Contains(caveatStr, "email") {
			parts := strings.SplitN(caveatStr, "=", 2)
			if len(parts) != 2 {
				el.Add(errors.BadRequest, "email caveat malformed")
				return nil
			}
			email = parts[1]
			break
		}
	}
	if email == "" {
		el.Add(errors.Unauthorized, "unauthorized: missing email in discharge macaroon")
		return nil
	}
	return &email
}

func GetRootMacaroonsFromString(macaroonAuth string) (string, string) {
	tokensString := strings.TrimPrefix(macaroonAuth, "Macaroon")
	tokens := strings.Split(tokensString, ",")
	var root string
	var discharge string
	for _, t := range tokens {
		if strings.Contains(t, " root=") {
			root = strings.TrimPrefix(t, " root=")
		} else {
			discharge = strings.TrimPrefix(t, " discharge=")
		}
	}

	return root, discharge
}

// TODO: implement
// returns email or id that is extracted out of discharge macaroon
func VerifyDischargeMacaroon(dischargeMacaroon string, el *errors.ErrorList) (string, error) {
	return "", nil
}

// TODO: implement
func VerifyRootMacaroon(rootMacaroon string, el *errors.ErrorList) error {
	return nil
}

func HasPermission(macaroon *macaroon.Macaroon, permission string) bool {
	for _, cav := range macaroon.Caveats() {
		caveatStr := string(cav.Id)
		if strings.Contains(caveatStr, "|acl|") {
			parts := strings.SplitN(caveatStr, "|acl|", 2)
			if len(parts) != 2 {
				return false
			}
			var permissions []string
			if err := json.Unmarshal([]byte(parts[1]), &permissions); err != nil {
				return false
			}
			for _, p := range permissions {
				if p == permission {
					return true
				}
			}
		}
	}
	return false
}

func uint64ToBytes(u uint) []byte {
	b := make([]byte, 8) // uint64 occupies 8 bytes
	binary.BigEndian.PutUint64(b, uint64(u))
	return b
}
