package macaroon

import (
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/macaroon.v2"
)

func TestMacaroonSerialize(t *testing.T) {
	// Create a sample macaroon.
	m, err := macaroon.New([]byte("secret"), []byte("identifier"), "location", macaroon.V1)
	assert.NoError(t, err, "expected no error creating macaroon")

	// Serialize the macaroon.
	serialized, err := MacaroonSerialize(m)
	assert.NoError(t, err, "expected no error serializing macaroon")
	assert.NotEmpty(t, serialized, "serialized macaroon should not be empty")

	// Decode the base64 value to verify that it equals what MarshalBinary returns.
	decoded, err := base64.RawURLEncoding.DecodeString(serialized)
	assert.NoError(t, err, "expected no error decoding base64 string")

	marshalled, err := m.MarshalBinary()
	assert.NoError(t, err, "expected no error marshaling macaroon to binary")
	assert.Equal(t, marshalled, decoded, "decoded value should equal original marshalled binary")
}

func TestMacaroonDeserialize(t *testing.T) {
	// Create a sample macaroon and serialize it.
	m, err := macaroon.New([]byte("secret"), []byte("identifier"), "location", macaroon.V1)
	assert.NoError(t, err, "expected no error creating macaroon")
	serialized, err := MacaroonSerialize(m)
	assert.NoError(t, err, "expected no error serializing macaroon")

	// Deserialize the serialized macaroon.
	deserialized, err := MacaroonDeserialize(serialized)
	assert.NoError(t, err, "expected no error in deserialization")

	// Verify that the number of caveats matches.
	assert.Equal(t, len(m.Caveats()), len(deserialized.Caveats()), "number of caveats should match")

	// Test the error case: attempt to deserialize an invalid string.
	invalidStr := "this is not a valid base64 string!"
	_, err = MacaroonDeserialize(invalidStr)
	assert.Error(t, err, "expected an error for invalid serialized macaroon")
}

func TestMacaroonSerializeDeserialize(t *testing.T) {
	// Create a sample macaroon.
	m, err := macaroon.New([]byte("secret"), []byte("identifier"), "location", macaroon.V1)
	assert.NoError(t, err, "expected no error creating macaroon")

	// Serialize the macaroon.
	serialized, err := MacaroonSerialize(m)
	assert.NoError(t, err, "expected no error serializing macaroon")

	// Deserialize the macaroon.
	deserialized, err := MacaroonDeserialize(serialized)
	assert.NoError(t, err, "expected no error deserializing macaroon")

	// Verify that the original and deserialized macaroons are equal.
	assert.Equal(t, m, deserialized, "original and deserialized macaroons should be equal")
}
