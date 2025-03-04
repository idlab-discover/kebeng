package auth

import (
	"context"
	"fmt"

	"encoding/base64"
    "encoding/binary"

	"github.com/idlab-discover/kebeng/services/gateway/internal/config"
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

func  GenerateMacaroon(ctx context.Context, req *message.GenerateMacaroonReq, macaroonConfig *config.MacaroonConfig) (*message.MacaroonRes, error) {
    m, err := macaroon.New(
        []byte(macaroonConfig.RootKey), 
        uint64ToBytes(macaroonConfig.RootId),
        macaroonConfig.RootLocation, 
        macaroon.V1,
    )
	if err != nil {
        return nil, fmt.Errorf("failed to create macaroon: %v", err)
	}
    err = m.AddThirdPartyCaveat([]byte(macaroonConfig.DischargeKey), uint64ToBytes(macaroonConfig.ThirdPartyCaveatId), macaroonConfig.ThirdPartyLocation)
    if err != nil {
        return nil, fmt.Errorf("failed to add third party caveat: %v", err)
    }
    
    // add permissions as caveats
    for _, perm := range req.Permissions {
        caveat := fmt.Sprintf("permission=%s", perm)
        err = m.AddFirstPartyCaveat([]byte(caveat))
        if err != nil {
            return nil, fmt.Errorf("failed to add first party caveat: %v", err)
        }
    }

    // add channels as caveats
    for _, channel := range req.Channels {
        caveat := fmt.Sprintf("channel=%s", channel)
        err = m.AddFirstPartyCaveat([]byte(caveat))
        if err != nil {
            return nil, fmt.Errorf("failed to add first party caveat: %v", err)
        }
    }

    // add packages as caveats
    // note: we need to check that either one of the formats is valid
    for _, pkg := range req.Packages {
        if pkg.Name != "" && pkg.Series != "" {
            caveat := fmt.Sprintf("package=%s&series=%s", pkg.Name, pkg.Series)
            err = m.AddFirstPartyCaveat([]byte(caveat))
            if err != nil {
                return nil, fmt.Errorf("failed to add first party caveat: %v", err)
            }
        } else if pkg.SnapId != "" {
            caveat := fmt.Sprintf("snap_id=%s", pkg.SnapId)
            err = m.AddFirstPartyCaveat([]byte(caveat))
            if err != nil {
                return nil, fmt.Errorf("failed to add first party caveat: %v", err)
            }
        } else {
            return nil, fmt.Errorf("invalid package restriction format")
        }
    }
        

    
    serializedMacaroon, err := MacaroonSerialize(m)
    if err != nil {
        return nil, fmt.Errorf("failed to serialize macaroon: %v", err)
    }
    
    return &message.MacaroonRes{Macaroon: serializedMacaroon}, nil
}

func uint64ToBytes(u uint) []byte {
    b := make([]byte, 8) // uint64 occupies 8 bytes
    binary.BigEndian.PutUint64(b, uint64(u))
    return b
}

