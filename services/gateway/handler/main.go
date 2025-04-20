package handler

import (
	"github.com/gin-gonic/gin"
	accClient "github.com/idlab-discover/kebeng/services/account/client"
	assertionClient "github.com/idlab-discover/kebeng/services/assertion/client"
	account "github.com/idlab-discover/kebeng/services/gateway/handler/account"
	"github.com/idlab-discover/kebeng/services/gateway/handler/assertion"
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
	snapHandler      snap.Handler
	accountHandler   account.Handler
	authHandler      auth.Handler
	assertionHandler assertion.Handler
}

func NewHandler(accountClient accClient.AccountClientInterface, storeClient storeClient.StoreClientInterface, assertionClient assertionClient.AssertionClientInterface, config *config.Config) *Handler {
	baseHandler := util.NewBaseHandler(accountClient, storeClient, assertionClient, config)

	return &Handler{
		snapHandler:      snap.Handler{BaseHandler: baseHandler},
		accountHandler:   account.Handler{BaseHandler: baseHandler},
		authHandler:      auth.Handler{BaseHandler: baseHandler},
		assertionHandler: assertion.Handler{BaseHandler: baseHandler},
	}
}

func (h *Handler) SetupEndpoints(r *gin.Engine) {
	// NOTE: doesn't really check macaroons yet just sets email to test value
	authGroup := r.Group("/dev/api")
	authGroup.Use(middleware.AuthMiddleware())

	// ********** ACCOUNT **********

	r.POST("/AddAccount", h.accountHandler.AddAccount)
	authGroup.GET("/account/", h.accountHandler.GetAccount)
	authGroup.PATCH("/account", h.accountHandler.PatchAccount)

	// ********** SNAP **********

	r.POST("api/v1/snaps/auth/nonces", h.snapHandler.RequestStoreDeviceNonce)      // TODO
	r.POST("api/v1/snaps/auth/sessions", h.snapHandler.RequestStoreDeviceSessions) // TODO
	r.POST("/v2/snaps/refresh", h.snapHandler.RefreshSnap)                         // TODO
	authGroup.POST("/register-name/", h.snapHandler.RegisterSnapName)
	r.GET("/v2/snaps/find", h.snapHandler.FindSnaps) // TODO
	r.POST("/dev/api/register-name-dispute/", h.snapHandler.RegisterSnapNameDispute)
	r.POST("/dev/api/snaps/:snap_id/builds", h.snapHandler.ProcessSnapBuildAssertion)
	r.GET("/download/:revision_id", h.snapHandler.DownloadSnap)
	authGroup.POST("/snap-push/", h.snapHandler.SnapPush)
	r.POST("/unscanned-upload/", h.snapHandler.UnscannedUpload)
	authGroup.GET("/snaps/:upload_id/status", h.snapHandler.GetUploadStatus)

	// ********** ASSERTION **********
	// NOTE: maybe put all the assertions under v2/assertions/:type/:id
	// instead of all seperate endpoints, the above way is the one snapcraft uses but doesn't really matter i think for simplicity this for now
	r.GET("v2/assertions/snap-revision/:rev_sha3_384", h.assertionHandler.GetSnapRevisionAssertion)
	r.GET("v2/assertions/snap-declaration/:series/:snap_id", h.assertionHandler.GetSnapDeclarationAssertion)
	r.GET("/v2/assertions/account/:account_id", h.assertionHandler.GetAccountAssertion)

	// ********** AUTH **********

	r.POST("/dev/api/acl/", h.authHandler.GenerateMacaroon)
	r.POST("/dev/api/acl/verify/", h.authHandler.VerifyMacaroon) //TODO: implement correctly too many unknows of what has to be included and what not, need good source
}
