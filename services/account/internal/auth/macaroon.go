package auth

import (
	"encoding/base64"
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
/*
func GetACLMacaroon(context context.Context, macaroonConfig *config.MacaroonConfig, acl string) (*macaroon.Macaroon, error) {
    m, err := macaroon.New([]byte(macaroonConfig.RootKey), macaroonConfig.RootId[:], macaroonConfig.RootLocation, macaroon.V1)
	if err != nil {
        return nil, fmt.Errorf("failed to create macaroon: %v", err)
	}
    
    err = m.AddThirdPartyCaveat([]byte(macaroonConfig.DischargeKey), macaroonConfig.ThirdPartyCaveatId[:], macaroonConfig.ThirdPartyLocation)
    if err != nil {
        return nil, fmt.Errorf("failed to add third party caveat: %v", err)
    }

    err = m.AddFirstPartyCaveat([]byte(acl))
    if err != nil {
        return nil, fmt.Errorf("failed to add first party caveat: %v", err)
    }
    return m, nil
}
*/
