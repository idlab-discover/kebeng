package handler

import (
	"github.com/gin-gonic/gin"
	accClient "github.com/idlab-discover/kebeng/services/account/client"
	assertionClient "github.com/idlab-discover/kebeng/services/assertion/client"
	account "github.com/idlab-discover/kebeng/services/gateway/handler/account"
	auth "github.com/idlab-discover/kebeng/services/gateway/handler/auth"
	snap "github.com/idlab-discover/kebeng/services/gateway/handler/snap"
	"github.com/idlab-discover/kebeng/services/gateway/internal/config"
	"github.com/idlab-discover/kebeng/services/gateway/internal/middleware"
	"github.com/idlab-discover/kebeng/services/gateway/internal/util"
	storeClient "github.com/idlab-discover/kebeng/services/store/client"
)

// this file handles all the http requests and maps it to the correct client

// handler that handles all the endpoints
type Handler struct {
	snapHandler    snap.Handler
	accountHandler account.Handler
	authHandler    auth.Handler
}

func NewHandler(accountClient accClient.AccountClientInterface, storeClient storeClient.StoreClientInterface, assertionClient assertionClient.AssertionClientInterface, config *config.Config) *Handler {
	baseHandler := util.NewBaseHandler(accountClient, storeClient, assertionClient, config)

	return &Handler{
		snapHandler:    snap.Handler{BaseHandler: baseHandler},
		accountHandler: account.Handler{BaseHandler: baseHandler},
		authHandler:    auth.Handler{BaseHandler: baseHandler},
	}
}

func (h *Handler) SetupEndpoints(r *gin.Engine) {
	// NOTE: doesn't really check macaroons yet just sets email to test value
	authGroup := r.Group("/dev/api")
	authGroup.Use(middleware.AuthMiddleware())

	// ********** ACCOUNT **********

	r.POST("/createAccount", h.accountHandler.CreateAccount)
	authGroup.GET("/account/", h.accountHandler.GetAccount)
	authGroup.PATCH("/account", h.accountHandler.PatchAccount)

	// ********** SNAP **********

	r.POST("api/v1/snaps/auth/nonces", h.snapHandler.RequestStoreDeviceNonce)      // TODO
	r.POST("api/v1/snaps/auth/sessions", h.snapHandler.RequestStoreDeviceSessions) // TODO
	r.POST("/v2/snaps/refresh", h.snapHandler.RefreshSnap)                         // TODO
	authGroup.POST("/dev/api/register-name/", h.snapHandler.RegisterSnapName)
	r.GET("/v2/snaps/find", h.snapHandler.FindSnaps) // TODO
	r.POST("/dev/api/register-name-dispute/", h.snapHandler.RegisterSnapNameDispute)
	r.POST("/dev/api/snaps/:snap_id/builds", h.snapHandler.ProcessSnapBuildAssertion)

	// ********** AUTH **********

	r.POST("/dev/api/acl/", h.authHandler.GenerateMacaroon)
	r.POST("/dev/api/acl/verify/", h.authHandler.VerifyMacaroon) //TODO: implement correctly to many unknows of what has to be included and what not, need good source
}
