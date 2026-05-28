package logic

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"time"

	"github.com/idlab-discover/kebeng/services/monitoring/internal/config"
	"github.com/idlab-discover/kebeng/services/monitoring/internal/model"

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
	url := fmt.Sprintf("%s/v2/snaps/refresh", l.Config.StoreUrl)
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

// SnapDownload no longer uses doRequest (which does ReadAll).
// Instead we stream the GET response body directly to io.Discard.
func (l *Logic) SnapDownload(downloadURL string) error {
	req, err := http.NewRequest("GET", downloadURL, nil)
	if err != nil {
		return fmt.Errorf("creating GET %s: %w", downloadURL, err)
	}
	req.Header.Set("Authorization", fmt.Sprintf(
		"Macaroon root=%s, discharge=%s",
		l.Config.Macaroon, l.Config.Macaroon,
	))

	resp, err := l.Client.Do(req)
	if err != nil {
		logrus.Error("SnapDownload:", err)
		return fmt.Errorf("HTTP GET %s error: %w", downloadURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		// read a small snippet of the error body, not the whole file
		buf := make([]byte, 512)
		n, _ := resp.Body.Read(buf)
		return fmt.Errorf("bad status %d from %s: %s", resp.StatusCode, downloadURL, buf[:n])
	}

	// stream into /dev/null (or any io.Writer)
	if _, err := io.Copy(io.Discard, resp.Body); err != nil {
		logrus.Error("SnapDownload copy:", err)
		return fmt.Errorf("reading body from %s: %w", downloadURL, err)
	}
	return nil
}

func (l *Logic) SnapPush(req model.SnapPushRequest) (*model.SnapPushResponse, error) {
	url := fmt.Sprintf("%s/dev/api/snap-push/", l.Config.StoreUrl)
	b, _ := json.Marshal(req)
	body, err := l.doRequest("POST", url, "application/json", bytes.NewReader(b))
	if err != nil {
		logrus.Error("SnapPush:", err)
		return nil, err
	}
	var out model.SnapPushResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("unmarshal snap-push response: %w", err)
	}
	return &out, nil
}

// UnscannedUpload now uses an io.Pipe + multipart.Writer so the binary
// is streamed directly from `reader` into the request body.
func (l *Logic) UnscannedUpload(reader io.Reader, entryName string) (*model.UnscannedUploadResponse, error) {
	pr, pw := io.Pipe()
	mw := multipart.NewWriter(pw)

	// spawn the goroutine that writes the multipart body into the pipe
	go func() {
		defer pw.Close()
		defer mw.Close()

		part, err := mw.CreateFormFile("binary", entryName)
		if err != nil {
			pw.CloseWithError(fmt.Errorf("create form file: %w", err))
			return
		}
		if _, err := io.Copy(part, reader); err != nil {
			pw.CloseWithError(fmt.Errorf("copy binary into multipart: %w", err))
			return
		}
	}()

	url := fmt.Sprintf("%s/unscanned-upload/", l.Config.StoreUrl)
	req, err := http.NewRequest("POST", url, pr)
	if err != nil {
		return nil, fmt.Errorf("creating POST %s: %w", url, err)
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.Header.Set("Authorization", fmt.Sprintf(
		"Macaroon root=%s, discharge=%s",
		l.Config.Macaroon, l.Config.Macaroon,
	))

	resp, err := l.Client.Do(req)
	if err != nil {
		logrus.Error("UnscannedUpload:", err)
		return nil, fmt.Errorf("HTTP POST %s error: %w", url, err)
	}
	defer resp.Body.Close()

	// read the (small) JSON response
	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response from %s: %w", url, err)
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("bad status %d from %s: %s", resp.StatusCode, url, respBytes)
	}

	var out model.UnscannedUploadResponse
	if err := json.Unmarshal(respBytes, &out); err != nil {
		return nil, fmt.Errorf("unmarshal unscanned-upload response: %w", err)
	}
	return &out, nil
}

func (l *Logic) UnscannedDeltaUpload(reader io.Reader, entryName string) (*model.UnscannedUploadResponse, error) {
	pr, pw := io.Pipe()
	mw := multipart.NewWriter(pw)

	go func() {
		defer pw.Close()
		defer mw.Close()

		part, err := mw.CreateFormFile("binary", entryName)
		if err != nil {
			pw.CloseWithError(fmt.Errorf("create form file: %w", err))
			return
		}
		if _, err := io.Copy(part, reader); err != nil {
			pw.CloseWithError(fmt.Errorf("copy binary into multipart: %w", err))
			return
		}
	}()

	url := fmt.Sprintf("%s/unscanned-delta-upload/", l.Config.StoreUrl)
	req, err := http.NewRequest("POST", url, pr)
	if err != nil {
		return nil, fmt.Errorf("creating POST %s: %w", url, err)
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.Header.Set("Authorization", fmt.Sprintf(
		"Macaroon root=%s, discharge=%s",
		l.Config.Macaroon, l.Config.Macaroon,
	))

	resp, err := l.Client.Do(req)
	if err != nil {
		logrus.Error("UnscannedDeltaUpload:", err)
		return nil, fmt.Errorf("HTTP POST %s error: %w", url, err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response from %s: %w", url, err)
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("bad status %d from %s: %s", resp.StatusCode, url, respBytes)
	}

	var out model.UnscannedUploadResponse
	if err := json.Unmarshal(respBytes, &out); err != nil {
		return nil, fmt.Errorf("unmarshal unscanned-delta-upload response: %w", err)
	}
	return &out, nil
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
		logrus.Error("getAccountAssertion: ", err)
		return "", err
	}
	return string(respBytes), nil
}

func (l *Logic) GetAccountKeyAssertion(pubKeySha, maxFormat string) (string, error) {
	url := fmt.Sprintf("%s/v2/assertions/account-key/%s?max-format=%s",
		l.Config.StoreUrl, pubKeySha, maxFormat)
	respBytes, err := l.doRequest("GET", url, "", nil)
	if err != nil {
		logrus.Error("getAccountKeyAssertion: ", err)
		return "", err
	}
	return string(respBytes), nil
}

func (l *Logic) RegisterNameAndUnscannedUpload(snapName string, reader io.Reader) error {
	if err := l.RegisterName(snapName); err != nil {
		return fmt.Errorf("registering name: %w", err)
	}
	if _, err := l.UnscannedUpload(reader, snapName); err != nil {
		return fmt.Errorf("unscanned upload: %w", err)
	}
	return nil
}

func (l *Logic) CreateCohorts(snapNames []string) (*model.CreateCohortsResult, error) {
	url := fmt.Sprintf("%s/v2/cohorts", l.Config.StoreUrl)
	payload := map[string][]string{"snaps": snapNames}
	b, _ := json.Marshal(payload)

	respBytes, err := l.doRequest("POST", url, "application/json", bytes.NewReader(b))
	if err != nil {
		logrus.Error("CreateCohorts: ", err)
		return nil, err
	}

	var out model.CreateCohortsResult
	if err := json.Unmarshal(respBytes, &out); err != nil {
		return nil, fmt.Errorf("unmarshal create-cohorts response: %v", err)
	}
	return &out, nil
}

func (l *Logic) FindSnaps(req model.FindSnapsRequest) (*model.FindSnapsResponse, error) {
	params := url.Values{}
	if req.Query != "" {
		params.Set("q", req.Query)
	}

	for _, field := range req.Fields {
		params.Add("fields", field)
	}

	for _, arch := range req.Architectures {
		params.Add("architecture", arch)
	}

	for _, ch := range req.Channels {
		params.Add("channel", ch)
	}

	for _, conf := range req.Confinements {
		params.Add("confinement", conf)
	}

	if req.Private {
		params.Add("private", "true")
	}

	fullURL := fmt.Sprintf("%s/v2/snaps/find", l.Config.StoreUrl)
	if len(params) > 0 {
		fullURL = fmt.Sprintf("%s?%s", fullURL, params.Encode())
	}

	respBytes, err := l.doRequest("GET", fullURL, "", nil)
	if err != nil {
		logrus.Error("FindSnaps:", err)
		return nil, err
	}

	var out model.FindSnapsResponse
	if err := json.Unmarshal(respBytes, &out); err != nil {
		return nil, fmt.Errorf("unmarshal find-snaps response: %w", err)
	}
	return &out, nil
}

func (l *Logic) DeltaPush(req model.DeltaUploadRequest, snapName string, unscannedFileName string, sha string) (*model.DeltaPushResponse, error) {
	url := fmt.Sprintf("%s/dev/api/snap-delta-push/", l.Config.StoreUrl)
	payload := map[string]any{
		"name":                   snapName,
		"updown_id":              unscannedFileName,
		"base_revision_sequence": req.BaseRevisionSequence,
		"delta_sha3_384":         sha,
		"delta_format":           req.DeltaFormat,
		"tracks_and_channels":    req.TracksAndChannels,
		"timeout_seconds":        req.TimeoutSeconds,
	}
	b, _ := json.Marshal(payload)
	respBytes, err := l.doRequest("POST", url, "application/json", bytes.NewReader(b))
	if err != nil {
		logrus.Error("DeltaPush:", err)
		return nil, err
	}
	var out model.DeltaPushResponse
	if err := json.Unmarshal(respBytes, &out); err != nil {
		return nil, fmt.Errorf("unmarshal delta-push response: %w", err)
	}
	return &out, nil
}

func (l *Logic) DeltaDownload(snapName, deltaFormat, deltaName string) error {

	url := fmt.Sprintf("%s/download-delta/%s/%s/%s", l.Config.StoreUrl, deltaFormat, snapName, deltaName)

	req, err := http.NewRequest("GET", url, nil)

	if err != nil {
		return fmt.Errorf("creating GET %s: %w", url, err)
	}

	req.Header.Set("Authorization", fmt.Sprintf(
		"Macaroon root=%s, discharge=%s",
		l.Config.Macaroon, l.Config.Macaroon,
	))

	resp, err := l.Client.Do(req)
	if err != nil {
		logrus.Error("DeltaDownload:", err)
		return fmt.Errorf("HTTP GET %s error: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		buf := make([]byte, 512)
		n, _ := resp.Body.Read(buf)
		return fmt.Errorf("bad status %d from %s: %s", resp.StatusCode, url, buf[:n])
	}

	if _, err := io.Copy(io.Discard, resp.Body); err != nil {
		logrus.Error("DeltaDownload copy:", err)
		return fmt.Errorf("reading body from %s: %w", url, err)
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
