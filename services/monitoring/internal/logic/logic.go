package logic

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"time"

	"monitoring/internal/config"
	"monitoring/internal/model"

	"github.com/sirupsen/logrus"
)

type Logic struct {
	Config *config.Config
	Client *http.Client
}

func NewLogic(cfg *config.Config, client *http.Client) *Logic {
	return &Logic{Config: cfg, Client: client}
}

func (l *Logic) RegisterName(snapName string) error {
	url := fmt.Sprintf("%s/dev/api/register-name/", l.Config.StoreUrl)
	payload := map[string]string{"snap_name": snapName}
	b, _ := json.Marshal(payload)
	if _, err := l.doRequest("POST", url, "application/json", bytes.NewReader(b)); err != nil {
		logrus.Error("RegisterName:", err)
		return err
	}
	return nil
}

func (l *Logic) RefreshSnap(req model.RefreshSnapRequest) (*model.RefreshSnapResponses, error) {
	url := fmt.Sprintf("%s/dev/api/refresh-snap", l.Config.StoreUrl)
	b, _ := json.Marshal(req)
	respBytes, err := l.doRequest("POST", url, "application/json", bytes.NewReader(b))
	if err != nil {
		logrus.Error("RefreshSnap:", err)
		return nil, err
	}
	var out model.RefreshSnapResponses
	if err := json.Unmarshal(respBytes, &out); err != nil {
		return nil, fmt.Errorf("unmarshal refresh-snap response: %w", err)
	}
	return &out, nil
}

func (l *Logic) RefreshDownload(snapName, channel string) (*model.RefreshSnapResponses, error) {
	instanceKey := fmt.Sprintf("download-%d", time.Now().UnixNano())
	action := &model.Action{
		Action:      "download",
		InstanceKey: instanceKey,
		Name:        snapName,
		Channel:     channel,
	}
	req := model.RefreshSnapRequest{
		Context: []model.Context{},
		Actions: []*model.Action{action},
		Fields:  []string{"download"},
	}
	return l.RefreshSnap(req)
}

func (l *Logic) RefreshInstall(snapName, channel string) (*model.RefreshSnapResponses, error) {
	instanceKey := fmt.Sprintf("install-%d", time.Now().UnixNano())
	action := &model.Action{
		Action:      "install",
		InstanceKey: instanceKey,
		Name:        snapName,
		Channel:     channel,
	}
	req := model.RefreshSnapRequest{
		Context: []model.Context{},
		Actions: []*model.Action{action},
		Fields:  []string{"install"},
	}
	return l.RefreshSnap(req)
}

func (l *Logic) SnapDownload(revisionID string) error {
	url := fmt.Sprintf("%s/download/%s", l.Config.StoreUrl, revisionID)
	if _, err := l.doRequest("GET", url, "", nil); err != nil {
		logrus.Error("SnapDownload:", err)
		return err
	}
	return nil
}

func (l *Logic) SnapPush(req model.SnapPushRequest) error {
	url := fmt.Sprintf("%s/dev/api/snap-push/", l.Config.StoreUrl)
	b, _ := json.Marshal(req)
	if _, err := l.doRequest("POST", url, "application/json", bytes.NewReader(b)); err != nil {
		logrus.Error("SnapPush:", err)
		return err
	}
	return nil
}

func (l *Logic) UnscannedUpload(reader io.Reader) error {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	p, _ := w.CreateFormFile("binary", fmt.Sprintf("%s", time.Now().Format(time.RFC3339)))
	io.Copy(p, reader)
	w.Close()

	url := fmt.Sprintf("%s/unscanned-upload/", l.Config.StoreUrl)
	if _, err := l.doRequest("POST", url, w.FormDataContentType(), &buf); err != nil {
		logrus.Error("UnscannedUpload:", err)
		return err
	}
	return nil
}

func (l *Logic) GetUploadStatus(uploadID string) (*model.UploadStatusResponse, error) {
	url := fmt.Sprintf("%s/dev/api/snaps/%s/status", l.Config.StoreUrl, uploadID)
	respBytes, err := l.doRequest("GET", url, "", nil)
	if err != nil {
		logrus.Error("GetUploadStatus:", err)
		return nil, err
	}
	// TODO: check if this works and if it can be unmarshalled errors is proto.Error instead of normal cerror idk why
	var out model.UploadStatusResponse
	if err := json.Unmarshal(respBytes, &out); err != nil {
		return nil, fmt.Errorf("unmarshal upload-status response: %w", err)
	}
	return &out, nil
}

func (l *Logic) GetSnapRevisionAssertion(revSHA, maxFormat string) (string, error) {
	url := fmt.Sprintf("%s/v2/assertions/snap-revision/%s?max-format=%s",
		l.Config.StoreUrl, revSHA, maxFormat)
	respBytes, err := l.doRequest("GET", url, "", nil)
	if err != nil {
		logrus.Error("GetSnapRevisionAssertion:", err)
		return "", err
	}
	return string(respBytes), nil
}

func (l *Logic) GetSnapDeclarationAssertion(series, snapID string) (string, error) {
	url := fmt.Sprintf("%s/v2/assertions/snap-declaration/%s/%s",
		l.Config.StoreUrl, series, snapID)
	respBytes, err := l.doRequest("GET", url, "", nil)
	if err != nil {
		logrus.Error("GetSnapDeclarationAssertion:", err)
		return "", err
	}
	return string(respBytes), nil
}

func (l *Logic) GetAccountAssertion(accountID, maxFormat string) (string, error) {
	url := fmt.Sprintf("%s/v2/assertions/account/%s?max-format=%s",
		l.Config.StoreUrl, accountID, maxFormat)
	respBytes, err := l.doRequest("GET", url, "", nil)
	if err != nil {
		logrus.Error("GetAccountAssertion:", err)
		return "", err
	}
	return string(respBytes), nil
}

func (l *Logic) GetAccountKeyAssertion(pubKeySha, maxFormat string) (string, error) {
	url := fmt.Sprintf("%s/v2/assertions/account-key/%s?max-format=%s",
		l.Config.StoreUrl, pubKeySha, maxFormat)
	respBytes, err := l.doRequest("GET", url, "", nil)
	if err != nil {
		logrus.Error("GetAccountKeyAssertion:", err)
		return "", err
	}
	return string(respBytes), nil
}

func (l *Logic) RegisterNameAndUnscannedUpload(snapName string, reader io.Reader) error {
	if err := l.RegisterName(snapName); err != nil {
		return fmt.Errorf("registering name: %w", err)
	}
	if err := l.UnscannedUpload(reader); err != nil {
		return fmt.Errorf("unscanned upload: %w", err)
	}
	return nil
}

// ================ Helper Functions ================

// doRequest sends an HTTP request, checks for errors, and returns the response body.
func (l *Logic) doRequest(method, url, contentType string, body io.Reader) ([]byte, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, fmt.Errorf("creating %s request to %s: %w", method, url, err)
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	req.Header.Set("Authorization", fmt.Sprintf(
		"Macaroon root=%s, discharge=%s",
		l.Config.Macaroon, l.Config.Macaroon,
	))

	resp, err := l.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP %s %s error: %w", method, url, err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response from %s: %w", url, err)
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("bad status %d from %s: %s", resp.StatusCode, url, respBytes)
	}
	return respBytes, nil
}
