package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	accClient "github.com/idlab-discover/kebeng/services/account/client"
	"github.com/idlab-discover/kebeng/services/gateway/internal/auth"
	"github.com/idlab-discover/kebeng/services/gateway/internal/config"
	"github.com/idlab-discover/kebeng/services/gateway/internal/errors"
	"github.com/idlab-discover/kebeng/services/gateway/internal/message"
	"github.com/idlab-discover/kebeng/services/gateway/internal/middleware"
	storeClient "github.com/idlab-discover/kebeng/services/store/client"
	storepb "github.com/idlab-discover/kebeng/services/store/proto"
)

// this file handles all the http requests and maps it to the correct client

type Handler struct {
	config        *config.Config
	AccountClient *accClient.AccountClient
	StoreClient   *storeClient.StoreClient
}

func NewHandler(accountClient *accClient.AccountClient, storeClient *storeClient.StoreClient, config *config.Config) *Handler {
	return &Handler{
		config:        config,
		AccountClient: accountClient,
		StoreClient:   storeClient,
	}
}

// TODO: add middelware function that checks authentication with discharge macaroon (can't get discharge macaroon as of now, so don't know the format/what it contains)
func (h *Handler) SetupEndpoints(r *gin.Engine) {
    // NOTE: doesn't really check macaroons yet just sets email to test value
    authGroup := r.Group("/dev/api")
    authGroup.Use(middleware.AuthMiddleware())

	r.POST("/createAccount", h.createAccount)
	r.POST("/dev/api/register-name/", h.RegisterSnapName)
	r.POST("/dev/api/acl/", h.generateMacaroon)

    //TODO: implement correctly to many unknows of what has to be included and what not, need good source
    r.POST("/dev/api/acl/verify/", h.verifyMacaroon)     

    authGroup.GET("/account/", h.getAccount)
}

func (h *Handler) createAccount(c *gin.Context) {
	var req message.CreateAccountRequest
    el := errors.New()
	if err := c.ShouldBindJSON(&req); err != nil {
        el.Add(errors.BadRequest, errors.FormatBindError(err))
		c.JSON(el.GetHTTPStatus(), gin.H{"error_list": el})
		return
	}
	account := h.AccountClient.CreateAccount(req.DisplayName, req.Username, req.Email)
	if len(account.Errors) > 0 {
		c.JSON(el.GetHTTPStatus(), gin.H{"error_list": el})
		return
	}

	c.JSON(http.StatusOK, message.CreateAccountResponse{Id: account.Id})
}

func (h *Handler) RegisterSnapName(c *gin.Context) {
	el := errors.New()
	var req message.RegisterSnapNameReq
	if err := c.ShouldBindJSON(&req); err != nil { el.Add(errors.BadRequest, errors.FormatBindError(err))
		c.JSON(http.StatusBadRequest, gin.H{"error_list": el})
		return
	}

	dryRun := c.Query("dry_run") == "1"
	resp := h.StoreClient.RegisterSnapName(req.SnapName, req.IsPrivate, req.Store, dryRun)
	if len(resp.Errors) > 0 {
		c.JSON(http.StatusInternalServerError, gin.H{"error_list": formatErrors(resp.Errors)})
		return
	}

	statusCode := http.StatusCreated
	if dryRun {
		statusCode = http.StatusOK
	}

	c.JSON(statusCode, message.RegisterSnapNameRes{SnapId: resp.Id, SnapName: req.SnapName})
}

func (h *Handler) generateMacaroon(c *gin.Context) {
	var req *message.GenerateMacaroonRequest
	el := errors.New()
	if err := c.ShouldBindJSON(&req); err != nil {
		el.Add(errors.BadRequest, errors.FormatBindError(err))
		c.JSON(el.GetHTTPStatus(),gin.H{"error_list": el,})
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
            Id:       p.SnapId,
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

func (h *Handler) getAccount(c *gin.Context) {
    el := errors.New()
    email, isThere := c.Get("email")
    if !isThere {
        el.Add(errors.BadRequest, "missing email")
        c.JSON(el.GetHTTPStatus(), gin.H{"error_list": el})
        return
    }

    account := h.AccountClient.GetAccountByEmail(email.(string))
    if len(account.Errors) > 0 {
        el.ExtendAccountError(account.Errors)
        c.JSON(http.StatusInternalServerError, gin.H{"error_list": el})
        return
    }
    accId, err := uuid.Parse(account.Id)
    if err != nil {
        // should never happen really
        el.Add(errors.InternalServerError, err.Error())
        c.JSON(el.GetHTTPStatus(), gin.H{"error_list": el})
        return
    }

    
    keys := h.AccountClient.GetAccountKeysByAccountID(account.Id)
    if len(keys.Errors) > 0 {
        el.ExtendAccountError(keys.Errors)
        c.JSON(http.StatusInternalServerError, gin.H{"error_list": el})
        return
    }
    // convert to message.AccountKey
    messageKeys := make([]message.AccountKey, len(keys.Keys))
    for i, k := range keys.Keys {
        messageKeys[i] = message.AccountKey{
            Name: k.Name,
            PublicKeySHA384: k.Sha3384,
            Since: k.Since.AsTime(),
            Until: k.Until.AsTime(),
        }
    }

    // Get snaps
    entries := h.StoreClient.GetEntriesByAccountID(account.Id)
    if len(entries.Errors) > 0 {
        el.ExtendStoreError(entries.Errors)
        c.JSON(http.StatusInternalServerError, gin.H{"error_list": el})
        return
    }
    // Get all revisions for every snapEntry
    snapEntries := make([]*storepb.GetRevisionsByEntryIdRequest, len(entries.Entries))
    for i, e := range entries.Entries {
        snapEntries[i] = &storepb.GetRevisionsByEntryIdRequest{Id: e}
    }

    // do 1 request where we get all the snapRevisions for every snapEntry
    revisions := h.StoreClient.GetRevisionsByEntryIds(&storepb.GetRevisionsByEntryIdRequests{Entries: snapEntries})
    if len(revisions.Errors) > 0 {
        el.ExtendStoreError(revisions.Errors)
        c.JSON(http.StatusInternalServerError, gin.H{"error_list": el})
        return
    }

    // do 1 request where we get all the Publishers for every snapEntry
    publisherRequests := make([]*storepb.GetPublisherRequest, len(snapEntries))
    for i, e := range snapEntries {
        publisherRequests[i] = &storepb.GetPublisherRequest{Entry: e.Id}
    }
    publishers := h.StoreClient.GetPublishers(&storepb.GetPublishersRequest{publishers: publisherRequests})
    if len(publishers.Errors) > 0 {
        el.ExtendStoreError(publishers.Errors)
        c.JSON(http.StatusInternalServerError, gin.H{"error_list": el})
        return
    }



    
    // not yet implemented don't support brandstores yet
    // stores := h.AccountClient.GetStores(account.Id)
    
    // leaving out the deprecated fields for now
    resp := message.AccountResponse{
        ID: accId,
        DisplayName: account.DisplayName,
        Email: account.Email,
        Username: account.Username,

        AccountKeys: messageKeys,
        Snaps: nil,
        Stores: nil,
    }
    c.JSON(http.StatusOK,resp)
}

func formatErrors(errors []*storepb.Error) []map[string]string {
	errs := make([]map[string]string, len(errors))
	for i, e := range errors {
		errs[i] = map[string]string{"code": e.GetCode(), "message": e.GetMessage()}
	}
	return errs
}
