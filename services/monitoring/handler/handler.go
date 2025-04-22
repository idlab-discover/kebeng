package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"monitoring/internal/config"
	"net/http"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	Config *config.Config
}

func NewHandler(cfg *config.Config) *Handler {
	return &Handler{
		Config: cfg,
	}
}

func (h *Handler) SetupEndpoints(r *gin.Engine) {
	r.GET("/register-name", h.RegisterName)
}

func (h *Handler) RegisterName(c *gin.Context) {
	data := map[string]string{
		"snap_name": "sinus123",
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		fmt.Println("Error marshalling JSON:", err)
		return
	}

	resp, err := http.Post(fmt.Sprintf("%s/dev/api/register-name/", h.Config.StoreUrl), "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Println("HTTP error:", err)
		return
	}
	defer resp.Body.Close()
}
