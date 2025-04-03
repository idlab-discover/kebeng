package middleware

import (
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
	cerror "github.com/idlab-discover/kebeng/common/cerror"
	"github.com/idlab-discover/kebeng/services/gateway/handler/auth"
	customMacaroon "github.com/idlab-discover/kebeng/services/gateway/internal/macaroon"
	"gopkg.in/macaroon.v2"
)

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		el := cerror.NewErrorList()
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			el.Add(cerror.Unauthorized, "missing Authorization header")
			c.JSON(el.GetHTTPStatus(), gin.H{"error-list": el})
			c.Abort()
			return
		}

		// Parse macaroon from header
		rootMacaroon, dischargeMacaroon, err := parseMacaroon(authHeader, el)
		if err != nil {
			c.JSON(el.GetHTTPStatus(), gin.H{"error-list": el})
			c.Abort()
			return
		}

		// TODO: currently template of how it would be done but can't yet parse for real since we don't have dischargeMacaroon
		// Verify the macaroon and extract user info
		/*
			email, err := auth.VerifyDischargeMacaroon(dischargeMacaroon, el)
			if err != nil {
				el.Add(cerror.Unauthorized, "Invalid discharge macaroon")
				c.JSON(el.GetHTTPStatus(), el)
				c.Abort()
				return
			}
			c.Set("email", email)
		*/
		// for now hardcoded email value to test untill discharge macaroon is available
		c.Set("email", "test@gmail.com")
		// set macaroon inside context so that endpoints can access it
		c.Set("rootMacaroon", rootMacaroon)
		c.Set("dischargeMacaroon", dischargeMacaroon)
		c.Next()
	}
}

// TODO: check returns account id or email idk yet depends what is in discharge macaroon
func parseMacaroon(authHeader string, el *cerror.ErrorList) (*macaroon.Macaroon, *macaroon.Macaroon, error) {

	// Expected format: "Macaroon root=<root-macaroon> discharge=<discharge-macaroon>"
	parts := strings.Split(authHeader, " ")
	if len(parts) < 3 {
		el.Add(cerror.Unauthorized, "Invalid Authorization header")
		return nil, nil, fmt.Errorf("Invalid Authorization header: %s", authHeader)
	}

	var dischargeMacaroon string
	var rootMacaroon string
	for _, part := range parts {
		if strings.HasPrefix(part, "root=") {
			rootMacaroon = strings.TrimPrefix(part, "root=")
			rootMacaroon = strings.TrimSuffix(rootMacaroon, ",")
		}
		if strings.HasPrefix(part, "discharge=") {
			dischargeMacaroon = strings.TrimPrefix(part, "discharge=")
			rootMacaroon = strings.TrimSuffix(rootMacaroon, ",")
		}
	}

	if rootMacaroon == "" {
		el.Add(cerror.Unauthorized, "Missing root macaroon")
		return nil, nil, fmt.Errorf("Missing root macaroon")
	}
	if dischargeMacaroon == "" {
		el.Add(cerror.Unauthorized, "Missing discharge macaroon")
		return nil, nil, fmt.Errorf("Missing discharge macaroon")
	}

	// TODO: actually check the macaroons as of now don't know format, just template functions
	// Verify the root macaroon
	err := auth.VerifyRootMacaroon(rootMacaroon, el)
	if err != nil {
		el.Add(cerror.Unauthorized, "Invalid root macaroon")
	}

	// deserialize the macaroon
	deserializedRootMacaroon, err := customMacaroon.MacaroonDeserialize(rootMacaroon)
	if err != nil {
		el.Add(cerror.Unauthorized, "Invalid root macaroon")
	}

	// Verify the discharge macaroon
	deserializedDischargeMacaroon, err := customMacaroon.MacaroonDeserialize(dischargeMacaroon)
	if err != nil {
		el.Add(cerror.Unauthorized, "Invalid discharge macaroon")
	}

	if len(*el) > 0 {
		return nil, nil, fmt.Errorf("Could not parse macaroon")
	}
	// return nil, nil, nil
	return deserializedRootMacaroon, deserializedDischargeMacaroon, nil
}
