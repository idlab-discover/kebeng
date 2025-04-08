package auth

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"encoding/json"

	cerror "github.com/idlab-discover/kebeng/common/cerror"
	"github.com/idlab-discover/kebeng/services/gateway/internal/config"
	mc "github.com/idlab-discover/kebeng/services/gateway/internal/macaroon"
	"github.com/idlab-discover/kebeng/services/gateway/internal/model"
	macaroon "gopkg.in/macaroon.v2"
)

func GenerateMacaroon(ctx context.Context, req *model.GenerateMacaroonRequest, snapIDs map[string]bool, macaroonConfig *config.MacaroonConfig) *model.MacaroonResponse {
	el := cerror.NewErrorList()

	// NOTE: don't need to check if macaroonconfig is okay since we do that at startup

	// start creating the macaroon
	m, err := macaroon.New(
		[]byte(macaroonConfig.RootKey),
		[]byte(macaroonConfig.RootId),
		macaroonConfig.RootLocation,
		macaroon.V1,
	)
	if err != nil {
		el.Add(cerror.InternalServerError, fmt.Sprintf("failed to create macaroon: %v", err))
		return &model.MacaroonResponse{Errors: *el}
	}

	// add valid since is in a macaroon when request from the canonical snapstore
	validSince := time.Now().Format(time.RFC3339Nano)
	err = m.AddFirstPartyCaveat([]byte(fmt.Sprintf("%s|valid_since|%s", macaroonConfig.RootLocation, validSince)))
	if err != nil {
		el.Add(cerror.InternalServerError, fmt.Sprintf("failed to add valid_since caveat: %v", err))
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
		el.Add(cerror.InternalServerError, fmt.Sprintf("failed to add third party caveat: %v", err))
	}

	// check generated macaroon requesting it from API and was in this format
	// (This aggregates the requested permissions into a single JSON array.)
	if len(req.Permissions) > 0 {
		permsJson, err := json.Marshal(req.Permissions)
		if err != nil {
			el.Add(cerror.InternalServerError, fmt.Sprintf("failed to marshal permissions: %v", err))
		} else {
			err = m.AddFirstPartyCaveat([]byte(fmt.Sprintf("%s|acl|%s", macaroonConfig.RootLocation, permsJson)))
			if err != nil {
				el.Add(cerror.InternalServerError, fmt.Sprintf("failed to add acl caveat: %v", err))
			}
		}
	}

	// add channels as caveats
	if len(req.Channels) > 0 {
		channelJson, err := json.Marshal(req.Channels)
		if err != nil {
			el.Add(cerror.InternalServerError, fmt.Sprintf("failed to marshal channel: %v", err))
		} else {
			caveat := fmt.Sprintf("%s|channel|%s", macaroonConfig.RootLocation, channelJson)
			err = m.AddFirstPartyCaveat([]byte(caveat))
			if err != nil {
				el.Add(cerror.InternalServerError, fmt.Sprintf("failed to add channel: %v, err:%v", req.Channels, err))
			}
		}
	}

	// add packages as caveats
	// we get the snapIDs instead of 2 different formats => allows consistency when decoding macaroon
	if len(snapIDs) > 0 {
		snapIDKeys := make([]string, len(snapIDs))
		i := 0
		for k := range snapIDs {
			snapIDKeys[i] = k
			i++
		}

		packagesJson, err := json.Marshal(snapIDKeys)
		if err != nil {
			el.Add(cerror.InternalServerError, fmt.Sprintf("failed to marshal packages: %v", err))
		} else {
			caveat := fmt.Sprintf("%s|snap_id|%s", macaroonConfig.RootLocation, packagesJson)
			err = m.AddFirstPartyCaveat([]byte(caveat))
			if err != nil {
				el.Add(cerror.InternalServerError, fmt.Sprintf("failed to add snap_id: %s, err: %v", packagesJson, err))
			}
		}
	}

	// Determine expiry: use the explicit expires value if provided,
	// otherwise if any permission among the default ones is requested,
	// set expiry to one year from now.
	var expiryTimestamp string
	if req.Expires != "" {
		// try parsing input to check if it is a valid time
		_, err := time.Parse(time.RFC3339, req.Expires)
		if err != nil {
			el.Add(cerror.InvalidField, fmt.Sprintf("invalid expires format: %v", err))
			return &model.MacaroonResponse{Errors: *el}
		}
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
			el.Add(cerror.InternalServerError, fmt.Sprintf("failed to add expires caveat: %v", err))
		}
	}

	// previous actions have to be successful before proceeding
	if len(*el) > 0 {
		return &model.MacaroonResponse{Errors: *el}
	}

	serializedMacaroon, err := mc.MacaroonSerialize(m)
	if err != nil {
		el.Add(cerror.InternalServerError, fmt.Sprintf("failed to serialize macaroon: %v", err))
		return &model.MacaroonResponse{Errors: *el}
	}

	return &model.MacaroonResponse{
		Macaroon: serializedMacaroon,
		Errors:   *el,
	}
}

func ValidateGenerateMacaroonRequest(req *model.GenerateMacaroonRequest, el *cerror.ErrorList) {

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
			*el = append(*el, cerror.NewCustomError(cerror.InvalidField, fmt.Sprintf("permission value '%s' is not allowed", perm)))
		}
	}

	for _, ch := range req.Channels {
		if _, ok := validChannels[ch]; !ok {
			*el = append(*el, cerror.NewCustomError(cerror.InvalidField, fmt.Sprintf("channel value '%s' is not allowed", ch)))
		}
	}

	// Validate Packages: each package must have either name/series or snap_id.
	// we don't allow both since this might cause internal conflicts if each of them refer to a different package
	for i, pkg := range req.Packages {
		// Case 1: Nothing provided.
		if pkg.Name == "" && pkg.Series == "" && pkg.SnapId == "" {
			*el = append(*el, cerror.NewCustomError(cerror.InvalidField, fmt.Sprintf("package at index %d: must provide either name/series or snap_id", i)))
			continue
		}
		// Case 2: snap_id is provided.
		if pkg.SnapId != "" {
			// If snap_id is provided, name and series should be empty.
			if pkg.Name != "" || pkg.Series != "" {
				*el = append(*el, cerror.NewCustomError(cerror.InvalidField, fmt.Sprintf("package at index %d: snap_id should be provided alone", i)))
			}
			// This package is valid, so we can continue.
			continue
		}
		// Case 3: snap_id is not provided, so name and series must both be provided.
		if pkg.Name == "" || pkg.Series == "" {
			*el = append(*el, cerror.NewCustomError(cerror.InvalidField, fmt.Sprintf("package at index %d: must provide both name and series", i)))
		}
	}

	// validate expires: if provided, must be in valid ISO8601 (RFC3339) format.
	if req.Expires != "" {
		if _, err := time.Parse(time.RFC3339, req.Expires); err != nil {
			*el = append(*el, cerror.NewCustomError(cerror.InvalidField, fmt.Sprintf("expires: %v", err)))
		}
	}
}

// TODO: implement further
// can only do this if we know format of discharge macaroon
func VerifyAndGetEmail(cfg *config.Config, el *cerror.ErrorList, authData string) *string {
	rootS, dischargeS := GetRootMacaroonsFromString(authData)

	root, err := mc.MacaroonDeserialize(rootS)
	if err != nil {
		el.Add(cerror.InternalServerError, fmt.Sprintf("failed to deserialize root macaroon: %v", err))
	}

	discharge, err := mc.MacaroonDeserialize(dischargeS)
	if err != nil {
		el.Add(cerror.InternalServerError, fmt.Sprintf("failed to deserialize discharge macaroon: %v", err))
	}
	if root == nil || discharge == nil {
		el.Add(cerror.Unauthorized, "missing macaroons")
		return nil
	}

	// TODO: verify both root macaroon and discharge macaroon
	return nil
}

func GetRootMacaroonsFromString(macaroonAuth string) (string, string) {
	tokensString := strings.TrimSpace(strings.TrimPrefix(macaroonAuth, "Macaroon"))

	// snapcraft uses a comma and snapd a space
	var tokens []string
	if strings.Contains(tokensString, ",") {
		tokens = strings.Split(tokensString, ",")
	} else {
		tokens = strings.Fields(tokensString)
	}

	var root, discharge string
	for _, token := range tokens {
		token = strings.TrimSpace(token)
		if strings.HasPrefix(token, "root=") {
			root = strings.TrimPrefix(token, "root=")
		} else if strings.HasPrefix(token, "discharge=") {
			discharge = strings.TrimPrefix(token, "discharge=")
		}
	}
	return root, discharge
}

// TODO: implement
// returns email or id that is extracted out of discharge macaroon
func VerifyDischargeMacaroon(dischargeMacaroon string, el *cerror.ErrorList) (string, error) {
	return "", nil
}

// TODO: implement
func VerifyRootMacaroon(rootMacaroon string, el *cerror.ErrorList) error {
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
			return slices.Contains(permissions, permission)
		}
	}
	return false
}
