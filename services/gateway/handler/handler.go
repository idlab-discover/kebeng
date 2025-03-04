package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	accClient "github.com/idlab-discover/kebeng/services/account/client"
	"github.com/idlab-discover/kebeng/services/gateway/internal/config"
	"github.com/idlab-discover/kebeng/services/gateway/internal/errors"
	"github.com/idlab-discover/kebeng/services/gateway/internal/message"
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

func (h *Handler) SetupEndpoints(r *gin.Engine) {
	r.POST("/createAccount", h.createAccount)
	// r.GET("/getMacaroon", h.generateMacaroon)
	r.POST("/dev/api/register-name/", h.RegisterSnapName)
}

func (h *Handler) createAccount(c *gin.Context) {
	var req message.CreateAccountReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"could not bind request to json": err.Error()})
		return
	}
	account, err := h.AccountClient.CreateAccount(req.DisplayName, req.Username, req.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error creating account": err.Error()})
		return
	}

	c.JSON(http.StatusOK, message.CreateAccountRes{Id: account.Id})
}

func (h *Handler) RegisterSnapName(c *gin.Context) {
	var req message.RegisterSnapNameReq
	if err := c.ShouldBindJSON(&req); err != nil {
		// TODO: steal code from Joran to print this error better
		c.JSON(http.StatusBadRequest, gin.H{"error_list": []map[string]string{{"code": errors.BadRequest, "message": err.Error()}}})
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

func formatErrors(errors []*storepb.Error) []map[string]string {
	errs := make([]map[string]string, len(errors))
	for i, e := range errors {
		errs[i] = map[string]string{"code": e.GetCode(), "message": e.GetMessage()}
	}
	return errs
}

/*
// TODO: know what the fuck this receives
func (h *Handler) generateMacaroon(c *gin.Context) {
    var req message.GenerateMacaroonReq
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"could not bind request to json": err.Error()})
        return
    }

    account, err := h.AccountClient.GetAccountByID(req.AccountId)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error getting account": err.Error()})
        return
    }

    acl := account.Username
    macaroon, err := auth.GenerateMacaroon(c, acl, h.config.MacaroonConfig)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error generating macaroon": err.Error()})
        return
    }
    c.JSON(http.StatusOK, macaroon)
}
*/
