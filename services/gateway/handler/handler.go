package handler

import (
    "net/http"
    accClient "github.com/idlab-discover/kebeng/services/account/client"
    storeClient "github.com/idlab-discover/kebeng/services/store/client"
    "github.com/gin-gonic/gin"
    "github.com/idlab-discover/kebeng/services/gateway/internal/message"
    "github.com/idlab-discover/kebeng/services/gateway/internal/config"
    "github.com/idlab-discover/kebeng/services/gateway/internal/auth"
)

// this file handles all the http requests and maps it to the correct client

type Handler struct {
    config *config.Config
    AccountClient *accClient.AccountClient
    StoreClient *storeClient.StoreClient
}

func NewHandler(accountClient *accClient.AccountClient, storeClient *storeClient.StoreClient, config *config.Config) *Handler {
    return &Handler{
        config: config,
        AccountClient: accountClient,
        StoreClient: storeClient,
    }
}



func (h *Handler) SetupEndpoints(r *gin.Engine) {
    r.POST("/createAccount",h.createAccount)
    r.GET("/getMacaroon", h.generateMacaroon)
}
 
func (h *Handler) createAccount(c *gin.Context) {
    var req message.CreateAccountReq
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"could not bind request to json": err.Error()})
        return
    }
    account, err := h.AccountClient.CreateAccount(req.DisplayName,req.Username, req.Email)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error creating account": err.Error()})
        return
    }

    c.JSON(http.StatusOK, message.CreateAccountRes{Id: account.Id})
}

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
