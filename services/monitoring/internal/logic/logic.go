package logic

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"monitoring/internal/config"

	"github.com/sirupsen/logrus"
)

type Logic struct {
	Config *config.Config
	Client *http.Client
}

func NewLogic(cfg *config.Config, client *http.Client) *Logic {
	return &Logic{
		Config: cfg,
		Client: client,
	}
}

func (l *Logic) RegisterName(snapName string) error {
	url := fmt.Sprintf("%s/dev/api/register-name/", l.Config.StoreUrl)
	data := map[string]string{
		"snap_name": snapName,
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		logrus.Errorf("Error marshalling JSON: %s", err)
		return fmt.Errorf("error marshalling JSON: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		logrus.Errorf("Error creating request: %s", err)
		return fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Macaroon root=%s, discharge=%s", l.Config.Macaroon, l.Config.Macaroon))

	resp, err := l.Client.Do(req)
	if err != nil {
		logrus.Errorf("HTTP error: %s", err)
		return fmt.Errorf("HTTP error: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("bad status %d: %s", resp.StatusCode, body)
	}
	return nil
}