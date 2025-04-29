package util

import (
	"gateway/internal/config"

	accClient "github.com/idlab-discover/kebeng/services/account/client"
	assertionClient "github.com/idlab-discover/kebeng/services/assertion/client"
	storeClient "github.com/idlab-discover/kebeng/services/store/client"
)

// the Base handler that every other handler such as SnapHandler should inherit from
type BaseHandler struct {
	Config          *config.Config
	AccountClient   accClient.AccountClientInterface
	StoreClient     storeClient.StoreClientInterface
	AssertionClient assertionClient.AssertionClientInterface
}

func NewBaseHandler(accountClient accClient.AccountClientInterface, storeClient storeClient.StoreClientInterface, assertionClient assertionClient.AssertionClientInterface, config *config.Config) *BaseHandler {
	return &BaseHandler{
		Config:          config,
		AccountClient:   accountClient,
		StoreClient:     storeClient,
		AssertionClient: assertionClient,
	}
}

// Helper functions for safe pointer dereferencing.
func GetString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func GetFloat64(f *float64) float64 {
	if f == nil {
		return 0.0
	}
	return *f
}

func GetBool(b *bool) bool {
	if b == nil {
		return false
	}
	return *b
}
