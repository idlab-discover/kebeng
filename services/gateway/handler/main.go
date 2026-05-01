package handler

import (
	account "github.com/idlab-discover/kebeng/services/gateway/handler/account"
	"github.com/idlab-discover/kebeng/services/gateway/handler/assertion"
	auth "github.com/idlab-discover/kebeng/services/gateway/handler/auth"
	snap "github.com/idlab-discover/kebeng/services/gateway/handler/snap"
	"github.com/idlab-discover/kebeng/services/gateway/internal/config"
	"github.com/idlab-discover/kebeng/services/gateway/internal/middleware"
	"github.com/idlab-discover/kebeng/services/gateway/internal/util"

	"github.com/gin-gonic/gin"
	monitoring "github.com/idlab-discover/kebeng/common/monitoring"
	accClient "github.com/idlab-discover/kebeng/services/account/client"
	assertionClient "github.com/idlab-discover/kebeng/services/assertion/client"
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
	r.GET("/v2/assertions/account-key/:public_key_sha3_384", h.assertionHandler.GetAccountKeyAssertion)

	// ********** AUTH **********

	r.POST("/dev/api/acl/", h.authHandler.GenerateMacaroon)
	r.POST("/dev/api/acl/verify/", h.authHandler.VerifyMacaroon) //TODO: implement correctly too many unknows of what has to be included and what not, need good source

	// ********** COHORTS ********
	r.POST("/v2/cohorts", h.snapHandler.CreateCohorts)

	// ********** DELTAS **********
	r.GET("/download-delta/:delta-format/:snap-name/:delta-name", h.snapHandler.DownloadDelta)
}

func (h *Handler) SetupEndpointsWithMonitoring(r *gin.Engine) {
	// NOTE: doesn't really check macaroons yet just sets email to test value
	authGroup := r.Group("/dev/api")
	authGroup.Use(middleware.AuthMiddleware())

	// ********** ACCOUNT **********

	r.POST("/AddAccount", monitoring.Wrapper("AddAccount", h.accountHandler.AddAccount))
	authGroup.GET("/account/", monitoring.Wrapper("GetAccount", h.accountHandler.GetAccount))
	authGroup.PATCH("/account", monitoring.Wrapper("PatchAccount", h.accountHandler.PatchAccount))

	// ********** SNAP **********

	r.POST("api/v1/snaps/auth/nonces", monitoring.Wrapper("RequestStoreDeviceNonce", h.snapHandler.RequestStoreDeviceNonce))
	r.POST("api/v1/snaps/auth/sessions", monitoring.Wrapper("RequestStoreDeviceSessions", h.snapHandler.RequestStoreDeviceSessions))
	r.POST("/v2/snaps/refresh", monitoring.Wrapper("RefreshSnap", h.snapHandler.RefreshSnap))
	authGroup.POST("/register-name/", monitoring.Wrapper("RegisterSnapName", h.snapHandler.RegisterSnapName))
	r.GET("/v2/snaps/find", monitoring.Wrapper("FindSnaps", h.snapHandler.FindSnaps))
	r.POST("/dev/api/register-name-dispute/", monitoring.Wrapper("RegisterSnapNameDispute", h.snapHandler.RegisterSnapNameDispute))
	r.GET("/download/:revision_id", monitoring.Wrapper("DownloadSnap", h.snapHandler.DownloadSnap))
	authGroup.POST("/snap-push/", monitoring.Wrapper("SnapPush", h.snapHandler.SnapPush))
	r.POST("/unscanned-upload/", monitoring.Wrapper("UnscannedUpload", h.snapHandler.UnscannedUpload))
	authGroup.GET("/snaps/:upload_id/status", monitoring.Wrapper("GetUploadStatus", h.snapHandler.GetUploadStatus))

	// ********** ASSERTION **********
	r.GET("v2/assertions/snap-revision/:rev_sha3_384", monitoring.Wrapper("GetSnapRevisionAssertion", h.assertionHandler.GetSnapRevisionAssertion))
	r.GET("v2/assertions/snap-declaration/:series/:snap_id", monitoring.Wrapper("GetSnapDeclarationAssertion", h.assertionHandler.GetSnapDeclarationAssertion))
	r.GET("/v2/assertions/account/:account_id", monitoring.Wrapper("GetAccountAssertion", h.assertionHandler.GetAccountAssertion))
	r.GET("/v2/assertions/account-key/:public_key_sha3_384", monitoring.Wrapper("GetAccountKeyAssertion", h.assertionHandler.GetAccountKeyAssertion))

	// ********** AUTH **********

	r.POST("/dev/api/acl/", monitoring.Wrapper("GenerateMacaroon", h.authHandler.GenerateMacaroon))
	r.POST("/dev/api/acl/verify/", monitoring.Wrapper("VerifyMacaroon", h.authHandler.VerifyMacaroon))

	// ********** COHORTS ********
	r.POST("/v2/cohorts", monitoring.Wrapper("CreateCohorts", h.snapHandler.CreateCohorts))

	// ********** DELTAS **********
	r.GET("/download-delta/:delta-format/:snap-name/:delta-name", monitoring.Wrapper("DownloadDelta", h.snapHandler.DownloadDelta))

}
