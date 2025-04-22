package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"monitoring/internal/config"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
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
		logrus.Errorf("Error marshalling JSON: %s", err)
		return
	}

	url := fmt.Sprintf("%s/dev/api/register-name/", h.Config.StoreUrl)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		logrus.Errorf("Error creating request: %s", err)
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Macaroon root=%s, discharge=%s", h.Config.Macaroon, h.Config.Macaroon))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		logrus.Errorf("HTTP error: %s", err)
		return
	}
	defer resp.Body.Close()
}
