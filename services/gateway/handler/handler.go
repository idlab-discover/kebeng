package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	accClient "github.com/idlab-discover/kebeng/services/account/client"
	assertionClient "github.com/idlab-discover/kebeng/services/assertion/client"
	"github.com/idlab-discover/kebeng/services/gateway/internal/auth"
	"github.com/idlab-discover/kebeng/services/gateway/internal/config"
	"github.com/idlab-discover/kebeng/services/gateway/internal/errors"
	"github.com/idlab-discover/kebeng/services/gateway/internal/message"
	storeClient "github.com/idlab-discover/kebeng/services/store/client"
	storepb "github.com/idlab-discover/kebeng/services/store/proto"
)

// this file handles all the http requests and maps it to the correct client

type Handler struct {
	config          *config.Config
	AccountClient   *accClient.AccountClient
	StoreClient     *storeClient.StoreClient
	AssertionClient *assertionClient.AssertionClient
}

func NewHandler(accountClient *accClient.AccountClient, storeClient *storeClient.StoreClient, config *config.Config) *Handler {
	return &Handler{
		config:        config,
		AccountClient: accountClient,
		StoreClient:   storeClient,
	}
}

func (h *Handler) SetupEndpoints(r *gin.Engine) {
	r.POST("/createAccount", h.createAccount)
	r.POST("/dev/api/register-name/", h.RegisterSnapName)
	r.POST("/dev/api/acl/", h.generateMacaroon)
	r.POST("/dev/api/register-name-dispute/", h.RegisterSnapNameDispute)
	r.POST("/dev/api/snaps/:snap_id/builds", h.ProcessSnapBuildAssertion)
	r.POST("/dev/api/acl/verify/", h.verifyMacaroon) //TODO: implement correctly to many unknows of what has to be included and what not, need good source
}

func (h *Handler) createAccount(c *gin.Context) {
	var req message.CreateAccountRequest
	el := errors.New()
	if err := c.ShouldBindJSON(&req); err != nil {
		el.Add(errors.BadRequest, errors.FormatBindError(err))
		c.JSON(el.GetHTTPStatus(), gin.H{"error_list": el})
		return
	}
	account, err := h.AccountClient.CreateAccount(req.DisplayName, req.Username, req.Email)
	if err != nil {
		el.Add(errors.InternalServerError, errors.FormatBindError(err))
		c.JSON(el.GetHTTPStatus(), gin.H{"error_list": el})
		return
	}

	c.JSON(http.StatusOK, message.CreateAccountResponse{Id: account.Id})
}

func (h *Handler) RegisterSnapName(c *gin.Context) {
	el := errors.New()
	var req message.RegisterSnapNameReq
	if err := c.ShouldBindJSON(&req); err != nil {
		el.Add(errors.BadRequest, errors.FormatBindError(err))
		c.JSON(el.GetHTTPStatus(), gin.H{"error_list": el})
		return
	}

	dryRun := c.Query("dry_run") == "1"
	resp := h.StoreClient.RegisterSnapName(req.SnapName, req.IsPrivate, req.Store, dryRun)
	if len(resp.Errors) > 0 {
		el.ExtendStoreError(resp.Errors)
		c.JSON(el.GetHTTPStatus(), gin.H{"error_list": el})
		return
	}

	statusCode := http.StatusCreated
	if dryRun {
		statusCode = http.StatusOK
	}

	c.JSON(statusCode, message.RegisterSnapNameRes{SnapId: resp.Id, SnapName: req.SnapName})
}

// TODO: Implement this function
// Reason for not being implemented yet:
// There is currently no way to keep track of disputes (in the database),
// disputes should be handled by a natural person, not by the system,
// which is not possible at the moment.
func (h *Handler) RegisterSnapNameDispute(c *gin.Context) {
	el := errors.New()
	el.Add(errors.NotImplemented, "Not implemented")
	c.JSON(el.GetHTTPStatus(), gin.H{"error_list": el})
}

func (h *Handler) generateMacaroon(c *gin.Context) {
	el := errors.New()
	var req *message.GenerateMacaroonRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		el.Add(errors.BadRequest, errors.FormatBindError(err))
		c.JSON(el.GetHTTPStatus(), gin.H{"error_list": el})
		return
	}

	// validate input
	auth.ValidateGenerateMacaroonRequest(req, el)
	if len(*el) > 0 {
		c.JSON(http.StatusBadRequest,
			gin.H{
				"error_list": el,
			})
		return
	}

	// check whether snapEntry exists else 404 not found
	entries := make([]*storepb.GetEntryRequest, len(req.Packages))
	for i, p := range req.Packages {
		entries[i] = &storepb.GetEntryRequest{
			Name: p.Name,
			Id:   p.SnapId,
		}
	}

	entriesResponse := h.StoreClient.GetEntries(&storepb.GetEntriesRequest{Entries: entries})
	if len(entriesResponse.Errors) > 0 {
		el.ExtendStoreError(entriesResponse.Errors)
		c.JSON(el.GetHTTPStatus(), gin.H{"error_list": el})
		return
	}

	// mimic set to prevent duplicate ids in macaroon
	snapIDs := make(map[string]bool, len(entriesResponse.Entries))
	for _, e := range entriesResponse.Entries {
		snapIDs[e.Id] = true
	}

	macaroon := auth.GenerateMacaroon(c, req, snapIDs, h.config.MacaroonConfig)
	if len(macaroon.Errors) > 0 {
		el.Extend(macaroon.Errors)
		c.JSON(el.GetHTTPStatus(), gin.H{"error_list": el})
		return
	}
	c.JSON(http.StatusOK, gin.H{"macaroon": macaroon.Macaroon})
}

func (h *Handler) ProcessSnapBuildAssertion(c *gin.Context) {
	el := errors.New()
	var req *message.SnapBuildAssertionReq
	if err := c.ShouldBindJSON(&req); err != nil {
		el.Add(errors.BadRequest, errors.FormatBindError(err))
		c.JSON(el.GetHTTPStatus(), gin.H{"error_list": el})
		return
	}
	// SnapID is a path parameter
	snapID := c.Param("snap_id")
	if snapID == "" {
		el.Add(errors.BadRequest, "snap_id is required")
		c.JSON(el.GetHTTPStatus(), gin.H{"error_list": el})
		return
	}

	// Process the snap build assertion with the snapID and req
	resp := h.AssertionClient.ProcessSnapBuildAssertion(req.Assertion)
	if len(resp.Errors) > 0 {
		el.ExtendAssertionError(resp.Errors)
		c.JSON(el.GetHTTPStatus(), gin.H{"error_list": el})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Snap build assertion processed successfully"})
}

// TODO: implement correctly to many unknows of what has to be included and what not, need good source
func (h *Handler) verifyMacaroon(c *gin.Context) {
	// not implemented for now
	c.JSON(http.StatusNotImplemented, gin.H{"error_list": errors.NewError(errors.NotImplemented, "not implemented too many unknowns of implementation")})
	return
	/*
			var req *message.VerifyRequest
			el := errors.New()
			if err := c.ShouldBindJSON(&req); err != nil {
				el.Add(errors.BadRequest, errors.FormatBindError(err))
				c.JSON(el.GetHTTPStatus(),gin.H{"error_list": el,})
				return
			}

		    // TODO: Don't know if you can get email from this actually or not (from discharge macaroon normally?)
			userEmail := auth.VerifyAndGetEmail(h.config, el, req.AuthData.Authorization)
		    // need userEmail to continue
			if userEmail == nil {
		        c.JSON(el.GetHTTPStatus(), gin.H{"error_list": el})
		        return
			}

		    // TODO: change account client to use errorList
			user, _ := h.AccountClient.GetAccountByEmail(*userEmail)
		    if user == nil {
		        el.Add(errors.ResourceNotFound, fmt.Sprintf("user with email %s not found", *userEmail))
		        c.JSON(el.GetHTTPStatus(), gin.H{"error_list": el})
		        return
		    }

		    // TODO: check wtf these values have to be
		    // probably get them from the macaroon
		    _ = message.VerifyResponse{
		        Allowed:               true, // idk what dertemines this
		        DeviceRefreshRequired: false, // obsolote
		        RefreshRequired:       false, // check with expiry time of macaroon
		        Account: &message.VerifyAccount{
		            Email:       user.Email,
		            DisplayName: user.DisplayName,
		            OpenId:      "oid1234", // what id is this?
		            Verified:    true, // idk what this is
		        },
		        Device:      nil, // obsolete
		        LastAuth:    "2016-05-26T12:53:23Z", // should be from macaroon
		        Permissions: &[]string{"package_access", "package_manage", "package_push", "package_register", "package_release", "package_update"}, // should be from macaroon
		        SnapIds:     nil, // should be from macaroon
		        Channels:    nil, // should be from macaroon
		    }
		    return
	*/
}
