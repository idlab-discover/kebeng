package middleware

import (
	"net/http"
	"strings"
    "fmt"

	"github.com/gin-gonic/gin"
	"github.com/idlab-discover/kebeng/services/gateway/internal/auth"
	"github.com/idlab-discover/kebeng/services/gateway/internal/errors"
)

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
        el := errors.New()
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
            el.Add(errors.BadRequest, "missing Authorization header")
            c.JSON(http.StatusUnauthorized, gin.H{"error_list": el})
			c.Abort()
			return
		}

		// Parse macaroon from header
		_, err := parseMacaroon(authHeader,el)
		if err != nil {
			c.JSON(el.GetHTTPStatus(), el)
			c.Abort()
			return
		}

        // for now hardcoded email value to test untill discharge macaroon is available
        c.Set("email", "test@gmail.com")
		//c.Set("email", email)
		c.Next()
	}
}

// TODO: check returns account id or email idk yet depends what is in discharge macaroon
func parseMacaroon(authHeader string, el *errors.ErrorList) (string, error) {
    
	// Expected format: "Macaroon root=<root-macaroon> discharge=<discharge-macaroon>"
	parts := strings.Split(authHeader, " ")
	if len(parts) < 3 {
        el.Add(errors.BadRequest, "Invalid Authorization header")
        return "", fmt.Errorf("Invalid Authorization header: %s", authHeader)
	}

	var dischargeMacaroon string
    var rootMacaroon string
	for _, part := range parts {
        if strings.HasPrefix(part, "root=") {
            rootMacaroon = strings.TrimPrefix(part, "root=")
        }
		if strings.HasPrefix(part, "discharge=") {
			dischargeMacaroon = strings.TrimPrefix(part, "discharge=")
		}
	}
    
    if rootMacaroon == "" {
        el.Add(errors.InvalidField, "Missing root macaroon")
        return "", fmt.Errorf("Missing root macaroon")
    }
	if dischargeMacaroon == "" {
        el.Add(errors.InvalidField, "Missing discharge macaroon")
		return "", fmt.Errorf("Missing discharge macaroon")
	}

    // TODO: actually check the macaroons as of now don't know format, just template functions
    // Verify the root macaroon
    err := auth.VerifyRootMacaroon(rootMacaroon, el)
    if err != nil {
        el.Add(errors.Unauthorized, "Invalid root macaroon")
    }

	// Verify the macaroon and extract user info
	email, err := auth.VerifyDischargeMacaroon(dischargeMacaroon,el)
	if err != nil {
        el.Add(errors.Unauthorized, "Invalid discharge macaroon")
		return "", err
	}

	return email, nil
}
