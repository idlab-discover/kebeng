package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	accClient "github.com/idlab-discover/kebeng/services/account/client"
	"github.com/idlab-discover/kebeng/services/gateway/internal/errors"
	"github.com/idlab-discover/kebeng/services/gateway/internal/message"
	storeClient "github.com/idlab-discover/kebeng/services/store/client"
	storepb "github.com/idlab-discover/kebeng/services/store/proto"
)

// this file handles all the http requests and maps it to the correct client

type Handler struct {
	AccountClient *accClient.AccountClient
	StoreClient   *storeClient.StoreClient
}

func NewHandler(accountClient *accClient.AccountClient, storeClient *storeClient.StoreClient) *Handler {
	return &Handler{
		AccountClient: accountClient,
		StoreClient:   storeClient,
	}
}

func (h *Handler) SetupEndpoints(r *gin.Engine) {
	r.POST("/createAccount", h.createAccount)
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
		c.JSON(http.StatusBadRequest, gin.H{"error_list": []map[string]string{{"code": errors.BadRequest, "message": err.Error()}}})
		return
	}
	resp := h.StoreClient.RegisterSnapName(req.SnapName, req.IsPrivate, req.Store)
	if len(resp.Errors) > 0 {
		c.JSON(http.StatusInternalServerError, gin.H{"error_list": formatErrors(resp.Errors)})
		return
	}

	c.JSON(http.StatusOK, gin.H{"id": resp.Id, "display_name": resp.SnapName})
}

func formatErrors(errors []*storepb.Error) []map[string]string {
	errs := make([]map[string]string, len(errors))
	for i, e := range errors {
		errs[i] = map[string]string{"code": e.GetCode(), "message": e.GetMessage()}
	}
	return errs
}
