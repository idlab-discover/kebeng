package handler

import (

    accClient "github.com/idlab-discover/kebeng/services/account/client"
    storeClient "github.com/idlab-discover/kebeng/services/store/client"
    "github.com/gin-gonic/gin"
    "github.com/idlab-discover/kebeng/services/gateway/internal/message"
    "net/http"
)

// this file handles all the http requests and maps it to the correct client

type Handler struct {
    AccountClient *accClient.AccountClient
    StoreClient *storeClient.StoreClient
}

func NewHandler(accountClient *accClient.AccountClient, storeClient *storeClient.StoreClient) *Handler {
    return &Handler{
        AccountClient: accountClient,
        StoreClient: storeClient,
    }
}



func (h *Handler) SetupEndpoints(r *gin.Engine) {
    r.GET("/createAccount",h.createAccount)
}
 
func (h *Handler) createAccount(c *gin.Context) {
    var req message.CreateAccountReq
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"could not bind req to json": err.Error()})
        return
    }
    account, err := h.AccountClient.CreateAccount(req.DisplayName,req.Username, req.Email)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error creating account": err.Error()})
        return
    }

    c.JSON(http.StatusOK, message.CreateAccountRes{Id: account.Id})
}
