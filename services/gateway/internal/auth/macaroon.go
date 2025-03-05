package auth

import (
	"context"
	"fmt"
    "time"

	"encoding/base64"
    "encoding/binary"

	"github.com/idlab-discover/kebeng/services/gateway/internal/config"
	"github.com/idlab-discover/kebeng/services/gateway/internal/message"
    "github.com/idlab-discover/kebeng/services/gateway/internal/errors"
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

func  GenerateMacaroon(ctx context.Context, req *message.GenerateMacaroonRequest, macaroonConfig *config.MacaroonConfig) (*message.MacaroonResponse) {
    el := errors.New()

    
    // start creating the macaroon
    m, err := macaroon.New(
        []byte(macaroonConfig.RootKey), 
        uint64ToBytes(macaroonConfig.RootId),
        macaroonConfig.RootLocation, 
        macaroon.V1,
    )
	if err != nil {
        el.Add(errors.InternalServerError, fmt.Sprintf("failed to create macaroon: %v", err))
        return &message.MacaroonResponse{Errors: *el}
	}

    err = m.AddThirdPartyCaveat([]byte(macaroonConfig.DischargeKey), uint64ToBytes(macaroonConfig.ThirdPartyCaveatId), macaroonConfig.ThirdPartyLocation)
    if err != nil {
        el.Add(errors.InternalServerError, fmt.Sprintf("failed to add third party caveat: %v", err))
    }
    
    // add permissions as caveats
    for _, perm := range req.Permissions {
        caveat := fmt.Sprintf("permission=%s", perm)
        err = m.AddFirstPartyCaveat([]byte(caveat))
        if err != nil {
            el.Add(errors.InternalServerError, fmt.Sprintf("failed to add permission: %v, err:%v",perm, err))
        }
    }

    // add channels as caveats
    for _, channel := range req.Channels {
        caveat := fmt.Sprintf("channel=%s", channel)
        err = m.AddFirstPartyCaveat([]byte(caveat))
        if err != nil {
            el.Add(errors.InternalServerError, fmt.Sprintf("failed to add channel: %v, err:%v",channel, err))
        }
    }

    // add packages as caveats
    // formats should already be checked here so we can assume they are correct
    for _, pkg := range req.Packages {
        if pkg.SnapId != "" {
            caveat := fmt.Sprintf("snap_id=%s", pkg.SnapId)
            err = m.AddFirstPartyCaveat([]byte(caveat))
            if err != nil {
                el.Add(errors.InternalServerError, fmt.Sprintf("failed to add snap_id: %s, err: %v", pkg.SnapId, err))
            }
        } else {
            // Since validation already checked, we assume pkg.Name and pkg.Series are both non-empty.
            caveat := fmt.Sprintf("name=%s&series=%s", pkg.Name, pkg.Series)
            err = m.AddFirstPartyCaveat([]byte(caveat))
            if err != nil {
                el.Add(errors.InternalServerError, fmt.Sprintf("failed to add package restriction for %s, err: %v", pkg.Name, err))
            }
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
        Errors: *el,
    }
}

func ValidateGenerateMacaroonRequest(req *message.GenerateMacaroonRequest, el *errors.ErrorList) {

	// Allowed permissions.
	validPermissions := map[string]struct{}{
		"edit_account":              {},
		"modify_account_key":        {},
		"package_access":            {},
		"package_register":          {},
		"package_push":              {},
		"package_release":           {},
		"package_update":            {},
		"package_metrics":           {},
		"package_manage":            {},
		"package_upload":            {},
		"package_upload_request":    {},
	}

    validChannels := map[string]struct{}{
        "edge": {},
        "beta": {},
        "candidate": {},
        "stable": {},
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
                "code":     errors.InvalidField,
                "message": fmt.Sprintf("package at index %d: must specify either name/series or snap_id", i),
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

func uint64ToBytes(u uint) []byte {
    b := make([]byte, 8) // uint64 occupies 8 bytes
    binary.BigEndian.PutUint64(b, uint64(u))
    return b
}

