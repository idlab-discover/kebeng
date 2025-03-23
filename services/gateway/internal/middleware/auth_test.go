package middleware

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"gopkg.in/macaroon.v2"

	cerror "github.com/idlab-discover/kebeng/common/cerror"
)

// NOTE: these tests will need to be adjusted whenever Ubuntu SSO authentication is implemented or other way of authorization is used

// validMacaroonString generates a valid serialized macaroon string.
func validMacaroonString(t *testing.T) string {
	t.Helper()
	// Create a new macaroon with a secret, identifier, and location.
	m, err := macaroon.New([]byte("secret-key"), []byte("identifier"), "localhost", macaroon.V1)
	if err != nil {
		t.Fatalf("failed to create macaroon: %v", err)
	}
	// Marshal the macaroon to binary.
	bin, err := m.MarshalBinary()
	if err != nil {
		t.Fatalf("failed to marshal macaroon: %v", err)
	}
	// Encode with base64.RawURLEncoding (without padding) as expected by MacaroonDeserialize.
	return base64.RawURLEncoding.EncodeToString(bin)
}

// --- Tests for parseMacaroon ---

func TestParseMacaroon_InvalidHeader(t *testing.T) {
	// Header with less than 3 parts.
	el := cerror.NewErrorList()
	header := "InvalidHeader"
	root, discharge, err := parseMacaroon(header, el)
	assert.Error(t, err, "Expected error when header has insufficient parts")
	assert.Nil(t, root)
	assert.Nil(t, discharge)
	// Optionally, check that the error list contains the expected message.
	assert.Equal(t, http.StatusUnauthorized, el.GetHTTPStatus())
}

func TestParseMacaroon_MissingRoot(t *testing.T) {
	el := cerror.NewErrorList()
	dischargeMacStr := validMacaroonString(t)
	header := "Macaroon root=" + " discharge=" + dischargeMacStr
	root, discharge, err := parseMacaroon(header, el)
	assert.Error(t, err, "Expected error when root macaroon is missing")
	assert.Nil(t, root)
	assert.Nil(t, discharge)
	assert.Equal(t, http.StatusUnauthorized, el.GetHTTPStatus())
}

func TestParseMacaroon_MissingDischarge(t *testing.T) {
	el := cerror.NewErrorList()

	rootMacStr := validMacaroonString(t)
	header := "Macaroon root=" + rootMacStr + " discharge="
	root, discharge, err := parseMacaroon(header, el)
	assert.Error(t, err, "Expected error when discharge macaroon is missing")
	assert.Nil(t, root)
	assert.Nil(t, discharge)
	assert.Equal(t, http.StatusUnauthorized, el.GetHTTPStatus())
}

func TestParseMacaroon_ValidHeader(t *testing.T) {
	el := cerror.NewErrorList()
	rootMacStr := validMacaroonString(t)
	dischargeMacStr := validMacaroonString(t)
	header := "Macaroon root=" + rootMacStr + " discharge=" + dischargeMacStr

	root, discharge, err := parseMacaroon(header, el)
	assert.NoError(t, err, "Expected no error with valid header")
	assert.NotNil(t, root)
	assert.NotNil(t, discharge)
}

// --- Tests for AuthMiddleware ---
// We test the  by setting up a Gin engine and invoking it with various headers.

func TestAuthMiddleware_MissingAuthorizationHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(AuthMiddleware())
	r.GET("/test", func(c *gin.Context) {
		// This handler should never be reached.
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	// Do not set "Authorization" header.
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Expect unauthorized status.
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	body := w.Body.String()
	assert.Contains(t, body, "missing Authorization header")
}

func TestAuthMiddleware_InvalidAuthorizationHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(AuthMiddleware())
	r.GET("/test", func(c *gin.Context) {
		// Should not be reached.
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	// Header that does not have enough parts.
	req.Header.Set("Authorization", "InvalidHeader")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	body := w.Body.String()
	assert.Contains(t, body, "Invalid Authorization header")
}

func TestAuthMiddleware_MissingRootMacaroon(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(AuthMiddleware())
	r.GET("/test", func(c *gin.Context) {
		// Should not be reached.
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Macaroon root= discharge=valid_discharge")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	body := w.Body.String()
	assert.Contains(t, body, "Missing root macaroon")
}

func TestAuthMiddleware_MissingDischargeMacaroon(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(AuthMiddleware())
	r.GET("/test", func(c *gin.Context) {
		// Should not be reached.
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Macaroon root=valid_root discharge=")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	body := w.Body.String()
	assert.Contains(t, body, "Missing discharge macaroon")
}

func TestAuthMiddleware_ValidHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	// The middleware will set a hardcoded email ("test@gmail.com") for testing.
	r.Use(AuthMiddleware())
	r.GET("/test", func(c *gin.Context) {
		// Return the values set in the context.
		email, _ := c.Get("email")
		rootMacaroon, _ := c.Get("rootMacaroon")
		dischargeMacaroon, _ := c.Get("dischargeMacaroon")
		c.JSON(http.StatusOK, gin.H{
			"email":             email,
			"rootMacaroon":      fmt.Sprintf("%v", rootMacaroon),
			"dischargeMacaroon": fmt.Sprintf("%v", dischargeMacaroon),
		})
	})

	req, _ := http.NewRequest("GET", "/test", nil)

	rootMacStr := validMacaroonString(t)
	dischargeMacStr := validMacaroonString(t)
	header := "Macaroon root=" + rootMacStr + " discharge=" + dischargeMacStr
	req.Header.Set("Authorization", header)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	// The  hardcodes email as "test@gmail.com" for testing purposes.
	assert.Equal(t, "test@gmail.com", resp["email"])
	// Since the macaroon objects are serialized via fmt.Sprintf, we simply check they are not empty.
	assert.NotEqual(t, "", resp["rootMacaroon"])
	assert.NotEqual(t, "", resp["dischargeMacaroon"])
}
