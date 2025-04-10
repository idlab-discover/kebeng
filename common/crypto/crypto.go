package crypto

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"

	"github.com/idlab-discover/kebeng/common/cerror"
	"github.com/snapcore/snapd/asserts"
	"github.com/snapcore/snapd/asserts/assertstest"
)

// Key represents a key that can be used for signing assertions.
type Key struct {
	Name     string `json:"name"`
	Sha3_384 string `json:"sha3-384"`
}

func GetPublicKeyPEM(key *rsa.PrivateKey) (string, *cerror.CustomError) {
	return ExportRsaPublicKeyAsPemStr(&key.PublicKey)
}

func CreateKeyPair(bitsSizeOfKey int) (asserts.PrivateKey, *rsa.PrivateKey) {
	pk, rsaPK := assertstest.GenerateKey(bitsSizeOfKey)
	return pk, rsaPK
}

func ExportRsaPrivateKeyAsPemStr(privkey *rsa.PrivateKey) string {
	privkeyBytes := x509.MarshalPKCS1PrivateKey(privkey)
	privkeyPem := pem.EncodeToMemory(
		&pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: privkeyBytes,
		},
	)
	return string(privkeyPem)
}

func ExportRsaPublicKeyAsPemStr(pubkey *rsa.PublicKey) (string, *cerror.CustomError) {
	pubkeyBytes, err := x509.MarshalPKIXPublicKey(pubkey)
	if err != nil {
		return "", cerror.NewCustomError(cerror.InternalServerError, "failed to marshal public key")
	}
	pubkeyPem := pem.EncodeToMemory(
		&pem.Block{
			Type:  "RSA PUBLIC KEY",
			Bytes: pubkeyBytes,
		},
	)

	return string(pubkeyPem), nil
}

var (
	ErrKeyMustBePEMEncoded = cerror.NewCustomError(cerror.InternalServerError, "key must be PEM encoded")
	ErrNotRSAPrivateKey    = cerror.NewCustomError(cerror.InternalServerError, "key is not a valid RSA private key")
	ErrNotRSAPublicKey     = cerror.NewCustomError(cerror.InternalServerError, "key is not a valid RSA public key")
)

func GetPrivateKeyFromPEMFile(keyPath string) (asserts.PrivateKey, *cerror.CustomError) {
	bytes, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, cerror.NewCustomError(cerror.InternalServerError, fmt.Sprintf("failed to read key file: %s", err.Error()))
	}
	if len(bytes) == 0 {
		return nil, cerror.NewCustomError(cerror.InternalServerError, fmt.Sprintf("key file is empty at: %s", keyPath))
	}
	rootPrivateKey, cerr := ParseRSAPrivateKeyFromPEM(bytes)
	if cerr != nil {
		return nil, cerr
	}

	return asserts.RSAPrivateKey(rootPrivateKey), nil
}

// from: "github.com/dgrijalva/jwt-go"
// Parse PEM encoded PKCS1 or PKCS8 private key
func ParseRSAPrivateKeyFromPEM(key []byte) (*rsa.PrivateKey, *cerror.CustomError) {
	var err error

	// Parse PEM block
	var block *pem.Block
	if block, _ = pem.Decode(key); block == nil {
		return nil, ErrKeyMustBePEMEncoded
	}

	var parsedKey interface{}
	if parsedKey, err = x509.ParsePKCS1PrivateKey(block.Bytes); err != nil {
		if parsedKey, err = x509.ParsePKCS8PrivateKey(block.Bytes); err != nil {
			return nil, cerror.NewCustomError(cerror.InternalServerError, "failed to parse private key")
		}
	}

	var pkey *rsa.PrivateKey
	var ok bool
	if pkey, ok = parsedKey.(*rsa.PrivateKey); !ok {
		return nil, ErrNotRSAPrivateKey
	}

	return pkey, nil
}
