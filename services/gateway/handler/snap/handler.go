package snap

import (
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"slices"

	"github.com/idlab-discover/kebeng/services/gateway/internal/model"
	"github.com/idlab-discover/kebeng/services/gateway/internal/util"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	cerror "github.com/idlab-discover/kebeng/common/cerror"
	storepb "github.com/idlab-discover/kebeng/services/store/proto"
	"github.com/sirupsen/logrus"
)

type Handler struct {
	*util.BaseHandler
}

// TODO: Implement this function properly
// Right now it's just a placeholder to make sure Snapcraft can be installed in the lxc container
func (h *Handler) RequestStoreDeviceNonce(c *gin.Context) {
	c.JSON(http.StatusOK, model.RequestStoreDeviceNonceResponse{Nonce: "this-nonce"})
}

// TODO: Implement this function properly
// Right now it's just a placeholder to make sure Snapcraft can be installed in the lxc container
func (h *Handler) RequestStoreDeviceSessions(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"macaroon": "shakamacaroon"})
}

// TODO: Implement this function properly
// Add all the cases for the different refresh actions
func (h *Handler) RefreshSnap(c *gin.Context) {
	el := cerror.NewErrorList()
	var req model.RefreshSnapRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		el.Add(cerror.BadRequest, cerror.FormatBindError(err))
		c.JSON(el.GetHTTPStatus(), gin.H{"error_list": el})
		return
	}

	var resp model.RefreshSnapResponses
	for _, action := range req.Actions {
		switch action.Action {
		case "download":
			res, cerr := h.refreshSnapDownload(action, el)
			if cerr != nil {
				c.JSON(el.GetHTTPStatus(), gin.H{"error_list": el})
				return
			}
			result := "download"
			res.Result = result
			resp.Responses = append(resp.Responses, res)
		case "install": // same as download just different result value, for now
			res, cerr := h.refreshSnapDownload(action, el)
			if cerr != nil {
				c.JSON(el.GetHTTPStatus(), gin.H{"error_list": el})
				return
			}
			result := "install"
			res.Result = result
			resp.Responses = append(resp.Responses, res)
		default:
			el.Add(cerror.NotImplemented, "Action not implemented")
			c.JSON(el.GetHTTPStatus(), gin.H{"error_list": el})
			return
		}
	}

	c.JSON(http.StatusOK, resp)
}

func (h *Handler) refreshSnapDownload(action *model.Action, el *cerror.ErrorList) (*model.RefreshSnapResult, *cerror.CustomError) {
	var res model.RefreshSnapResult
	snapEntry, latestRevision := h.getLatestRevisionByEntryName(el, action.Name, action.Channel)
	if el.HasError() {
		return nil, cerror.NewCustomError(cerror.ResourceNotFound, fmt.Sprintf("latest revision not found by name: %s", action.Name))
	}

	// if publisher not found we should error this is not safe if we don't know who published it
	publisher := h.AccountClient.GetAccountByID(snapEntry.PublisherId)
	if len(publisher.Errors) > 0 {
		el.ExtendProtoError(publisher.Errors)
		res.Result = "error"
		return &res, cerror.NewCustomError(cerror.InternalServerError, fmt.Sprintf("account error: %v", publisher.Errors))
	}

	downloadUrl := fmt.Sprintf("%s/download/%s", h.Config.StoreUrl, latestRevision.Id)

	raw, err := base64.RawURLEncoding.DecodeString(latestRevision.Sha3_384Encoded)
	if err != nil {
		el.Add(cerror.InternalServerError, fmt.Sprintf("error decoding sha3_384: %s", err.Error()))
	}

	hexSum := hex.EncodeToString(raw)

	res.InstanceKey = action.InstanceKey
	res.SnapId = snapEntry.Id
	res.Name = snapEntry.SnapName
	res.Snap = &model.RefreshSnap{
		Architectures: latestRevision.Architectures,
		SnapId:        snapEntry.Id,
		Name:          snapEntry.SnapName,
		Publisher: &model.Publisher{
			Username: publisher.Username,
			ID:       publisher.Id,
		},
		Download: &model.Download{
			URL:      &downloadUrl,
			Sha3_384: &hexSum,
			Size:     &latestRevision.Size,
		},
		Version:     latestRevision.Version,
		Revision:    latestRevision.SequenceNumber,
		Confinement: snapEntry.Confinement,
		Type:        snapEntry.Type,
		Base:        snapEntry.Base,
	}
	return &res, nil
}

// This endpoint is used to download when doing snap install since they ask for a seperate endpoint that just immediately downloads
func (h *Handler) DownloadSnap(c *gin.Context) {
	el := cerror.NewErrorList()
	revisionId := c.Param("revision_id")
	if revisionId == "" {
		el.Add(cerror.BadRequest, "revision_id is required")
		c.JSON(el.GetHTTPStatus(), gin.H{"error_list": el})
		return
	}
	h.downloadSnap(c, revisionId, el)
}

func (h *Handler) FindSnaps(c *gin.Context) {
	el := cerror.NewErrorList()
	var req model.FindSnapsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		el.Add(cerror.BadRequest, cerror.FormatBindError(err))
		c.JSON(el.GetHTTPStatus(), gin.H{"error_list": el})
		return
	}
}

func (h *Handler) RegisterSnapName(c *gin.Context) {
	el := cerror.NewErrorList()
	var req model.RegisterSnapNameRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		el.Add(cerror.BadRequest, cerror.FormatBindError(err))
		c.JSON(el.GetHTTPStatus(), gin.H{"error-list": el})
		return
	}

	// Get email of the user from the macaroon
	c.Get("email")
	email, ok := c.Get("email")
	if !ok {
		el.Add(cerror.Unauthorized, "email not found in macaroon")
		c.JSON(el.GetHTTPStatus(), gin.H{"error-list": el})
		return
	}

	// Get the account by email -> we need the account ID to register the snap name
	account := h.AccountClient.GetAccountByEmail(email.(string))
	if len(account.Errors) > 0 {
		el.ExtendProtoError(account.Errors)
		c.JSON(el.GetHTTPStatus(), gin.H{"error-list": el})
		return
	}

	// Parse the account ID
	accountUUID, err := uuid.Parse(account.Id)
	if err != nil {
		el.Add(cerror.BadRequest, "invalid account ID format")
		c.JSON(el.GetHTTPStatus(), gin.H{"error-list": el})
		return
	}

	dryRun := c.Query("dry_run") == "1"

	// if store name is empty, use the default store name
	if req.Store == "" {
		req.Store = h.Config.StoreName
	}

	resp := h.StoreClient.RegisterSnapName(req.SnapName, "app", "", "", req.IsPrivate, "active", 0.0, req.Store, "", dryRun, accountUUID)
	if len(resp.Errors) > 0 {
		el.ExtendProtoError(resp.Errors)
		c.JSON(el.GetHTTPStatus(), gin.H{"error-list": el})
		return
	}

	statusCode := http.StatusCreated
	if dryRun {
		if resp.SnapName == "" {
			statusCode = http.StatusNoContent
		} else {
			statusCode = http.StatusOK
		}
	}

	c.JSON(statusCode, model.RegisterSnapNameResponse{SnapId: resp.Id, SnapName: req.SnapName})
}

// TODO: Implement this function
// Reason for not being implemented yet:
// There is currently no way to keep track of disputes (in the database),
// disputes should be handled by a natural person, not by the system,
// which is not possible at the moment.
func (h *Handler) RegisterSnapNameDispute(c *gin.Context) {
	el := cerror.NewErrorList()
	el.Add(cerror.NotImplemented, "Not implemented")
	c.JSON(el.GetHTTPStatus(), gin.H{"error_list": el})
}

// SnapPush checks if there exists a snap entry for the uploaded snap package.
// It calls the RegisterSnapName function with dryRun = true.
func (h *Handler) SnapPush(c *gin.Context) {
	el := cerror.NewErrorList()
	var req *model.SnapPushRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		el.Add(cerror.BadRequest, cerror.FormatBindError(err))
		c.JSON(el.GetHTTPStatus(), gin.H{"error-list": el})
		return
	}

	// Get email of the user from the macaroon
	email, ok := c.Get("email")
	if !ok {
		el.Add(cerror.Unauthorized, "email not found in macaroon")
		c.JSON(el.GetHTTPStatus(), gin.H{"error-list": el})
		return
	}

	// Get the account by email -> we need the account ID to register the snap name
	account := h.AccountClient.GetAccountByEmail(email.(string))
	if len(account.Errors) > 0 {
		el.ExtendProtoError(account.Errors)
		c.JSON(el.GetHTTPStatus(), gin.H{"error-list": el})
		return
	}

	// Parse the account ID
	accountUUID, err := uuid.Parse(account.Id)
	if err != nil {
		el.Add(cerror.BadRequest, "invalid account ID format")
		c.JSON(el.GetHTTPStatus(), gin.H{"error-list": el})
		return
	}

	// Dry run to check if the snap name is registered
	entry := h.StoreClient.RegisterSnapName(req.Name, "", "", "", false, "", 0.0, "", "", true, accountUUID)
	if len(entry.Errors) > 0 {
		el.ExtendProtoError(entry.Errors)
		c.JSON(el.GetHTTPStatus(), gin.H{"error-list": el})
		return
	}

	if entry.SnapName == "" {
		el.AddCustomError(cerror.NewCustomError(cerror.ResourceNotFound, "Snap name not found for name="+req.Name))
		c.JSON(el.GetHTTPStatus(), gin.H{"error-list": el})
		return
	}

	// TODO: Dry run to check if the user is allowed to push the snap -> check permissions in root macaroon
	if req.DryRun {
		if entry.SnapName != "" {
			c.JSON(http.StatusOK, gin.H{
				"success":   true,
				"snap_name": entry.SnapName,
				"snap_id":   entry.Id,
				"status":    "dry-run",
			})
			return
		} else {
			el.Add(cerror.BadRequest, "snap name not found for name="+req.Name)
			c.JSON(el.GetHTTPStatus(), gin.H{"error-list": el})
			return
		}
	}

	// If the snap name is registered, we can proceed with the upload
	parsedEntryUUID, err := uuid.Parse(entry.Id)
	if err != nil {
		el.Add(cerror.BadRequest, "invalid entry ID format")
		c.JSON(el.GetHTTPStatus(), gin.H{"error-list": el})
		return
	}

	// Get the file name of the unscanned upload from the request body
	logrus.Debugf("Temp file name: %s", req.UnscannedFileName)

	// Create a new snap upload with status "pending" and revision 0
	upload := h.StoreClient.AddUpload(parsedEntryUUID, accountUUID, entry.SnapName, "pending", req.UnscannedFileName, 0)
	if len(upload.Errors) > 0 {
		el.ExtendProtoError(upload.Errors)
		c.JSON(el.GetHTTPStatus(), gin.H{"error-list": el})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":            true,
		"snap_name":          entry.SnapName,
		"upload_id":          upload.Id,
		"status_details_url": fmt.Sprintf("%s/dev/api/snaps/%s/status", h.Config.StoreUrl, upload.Id),
	})

	// After the status details URL is returned, we can proceed with creating a new revision
	// We need the sha3_384_encoded hash of the snap package, and other metadata
	metadata := h.StoreClient.GetObjectCustomMetadata("unscanned", req.UnscannedFileName)
	if metadata.Errors.HasError() {
		el.Extend(*metadata.Errors)
		return
	}
	logrus.Debugf("metadata: %v", metadata)

	// Update the snapEntry with the metadata
	updatedEntry := h.StoreClient.UpdateSnapEntryWithMetadata(parsedEntryUUID, metadata)
	if len(updatedEntry.Errors) > 0 {
		el.ExtendProtoError(updatedEntry.Errors)
		return
	}
	logrus.Debugf("Updated entry: %v", updatedEntry)

	// Create a new revision for the snap upload
	revision := h.StoreClient.AddRevision(entry.SnapName, metadata.Sha3_384Encoded, uint64(req.BinaryFileSize), metadata.Architectures, req.Channels, req.UnscannedFileName)
	if len(revision.Errors) > 0 {
		el.ExtendProtoError(revision.Errors)
	}

	// Create a new SnapRevisionAssertion for the snap upload
	revAssertion := h.AssertionClient.AddSnapRevisionAssertion(metadata.Sha3_384Encoded, accountUUID.String(), entry.Id, revision.Revision, uint64(req.BinaryFileSize))
	if len(revAssertion.Errors) > 0 {
		el.ExtendProtoError(revAssertion.Errors)
	}

	// Create a new SnapDeclarationAssertion for the snap upload
	declAssertion := h.AssertionClient.AddSnapDeclarationAssertion(entry.Id, entry.SnapName, accountUUID.String(), req.Series, metadata.RefreshControl, nil, metadata.Plugs, metadata.Slots) // TODO: support aliases
	if len(declAssertion.Errors) > 0 {
		el.ExtendProtoError(declAssertion.Errors)
	}

	// Ignore 'resource not found' errors for the revision -> this is expected if the revision already exists, or tracks and channels didn't exist
	el.RemoveErrorWithCode(cerror.ResourceNotFound)

	// Update the upload status to "processed"
	updatedUpload := h.StoreClient.UpdateUploadStatus(upload.Id, "processed", revision.Revision, el)
	if len(updatedUpload.Errors) > 0 {
		el.ExtendProtoError(updatedUpload.Errors)
		return
	}
}

func (h *Handler) UnscannedUpload(c *gin.Context) {
	el := cerror.NewErrorList()

	// Grab the multipart reader for the request
	mr, err := c.Request.MultipartReader()
	if err != nil {
		el.Add(cerror.BadRequest, "invalid multipart/form-data binary file not found")
		c.JSON(el.GetHTTPStatus(), gin.H{"error-list": el})
		return
	}

	var (
		filePart io.Reader
		filename string
	)
	// Iterate parts until we find the "binary" field
	for {
		part, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			el.Add(cerror.InternalServerError, "error reading multipart")
			c.JSON(el.GetHTTPStatus(), gin.H{"error-list": el})
			return
		}
		if part.FormName() == "binary" {
			filename = part.FileName()
			filePart = part
			break
		}
		part.Close() // skip unrelated parts
	}

	if filePart == nil {
		el.Add(cerror.BadRequest, "binary part not found")
		c.JSON(el.GetHTTPStatus(), gin.H{"error-list": el})
		return
	}

	// Stream the file part directly to your StoreClient
	resp := h.StoreClient.UnscannedUpload(c, filePart)
	if len(resp.Errors) > 0 {
		el.ExtendProtoError(resp.Errors)
		c.JSON(el.GetHTTPStatus(), gin.H{"error-list": el})
		return
	}
	if resp.GetTempFileName() == "" {
		el.Add(cerror.InternalServerError, "upload failed: no ID returned")
		c.JSON(el.GetHTTPStatus(), gin.H{"error-list": el})
		return
	}

	// Respond with the upload info
	c.JSON(http.StatusOK, gin.H{
		"successful": true,
		"upload_id":  resp.GetTempFileName(),
		"filename":   filename,
		"size":       resp.GetSize(),
	})
}

func (h *Handler) GetUploadStatus(c *gin.Context) {
	el := cerror.NewErrorList()
	uploadId := c.Param("upload_id")
	if uploadId == "" {
		el.Add(cerror.BadRequest, "upload_id is required")
		c.JSON(el.GetHTTPStatus(), gin.H{"error-list": el})
		return
	}
	// Get the upload status from the store
	uploadStatus := h.StoreClient.GetUploadStatus(uploadId)
	if len(uploadStatus.Errors) > 0 {
		el.ExtendProtoError(uploadStatus.Errors)
		c.JSON(el.GetHTTPStatus(), gin.H{"error-list": el})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":      "200",
		"processed": uploadStatus.Processed,
		"revision":  uploadStatus.Revision,
		"errors":    uploadStatus.Errors,
	})
}

// ########################## HELPER FUNCTIONS ##########################

// download helper function that just downloads a snap nothing else
func (h *Handler) downloadSnap(c *gin.Context, revisionId string, el *cerror.ErrorList) {
	if revisionId == "" {
		el.Add(cerror.BadRequest, "revision_id is required")
		c.JSON(el.GetHTTPStatus(), gin.H{"error_list": el})
		return
	}

	stream, err := h.StoreClient.SnapDownloadStream(revisionId)
	if err != nil {
		el.Add(cerror.InternalServerError, err.Error())
		c.JSON(el.GetHTTPStatus(), gin.H{"error_list": el})
		return
	}
	defer func() {
		err := stream.CloseSend()
		if err != nil {
			logrus.Errorf("failed to close stream: %v", err)
		}
	}()

	c.Writer.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, revisionId+".snap"))
	c.Writer.Header().Set("Content-Type", "application/octet-stream")
	c.Writer.WriteHeader(http.StatusOK)

	for {
		resp, err := stream.Recv()
		switch {
		case err == io.EOF:
			return
		case err != nil:
			logrus.Errorf("download stream error: %v", err)
			return
		}

		if resp == nil {
			logrus.Errorf("download stream returned nil resp")
			return
		}
		if len(resp.Errors) > 0 {
			logrus.Errorf("failed while reading downloadstream: %+v", resp.Errors)
			return
		}

		if data := resp.GetData(); data != nil {
			if _, writeErr := c.Writer.Write(data.Chunk); writeErr != nil {
				logrus.Errorf("write to HTTP client failed: %v", writeErr)
				return
			}
			if f, ok := c.Writer.(http.Flusher); ok {
				f.Flush()
			}
		}
	}
}

func (h *Handler) getLatestRevisionByEntryName(el *cerror.ErrorList, entryName string, channelAndTrack ...string) (*storepb.GetEntryResponse, *storepb.GetRevisionResponse) {
	// should only get 1 entry since name is unique
	snapEntries := h.StoreClient.GetEntries(&storepb.GetEntriesRequest{
		Entries: []*storepb.GetEntryRequest{
			{
				Name:                &entryName,
				PreloadAssociations: []string{"REVISIONS"},
			},
		},
	})
	if len(snapEntries.Errors) > 0 {
		el.ExtendProtoError(snapEntries.Errors)
		return nil, nil
	}
	if len(snapEntries.Entries) == 0 {
		el.Add(cerror.ResourceNotFound, fmt.Sprintf("snap entry not found with name %s", entryName))
		return nil, nil
	}
	snapEntry := snapEntries.Entries[0]

	track := "latest"
	channel := "stable"

	// Determine track and channel from variadic parameters.
	switch len(channelAndTrack) {
	case 0:
		// Use defaults.
	case 1:
		if isChannel(channelAndTrack[0]) {
			channel = channelAndTrack[0]
		} else {
			track = channelAndTrack[0]
		}
	default: // len >= 2; we only use the first two.
		a, b := channelAndTrack[0], channelAndTrack[1]
		// If one value is a valid channel and the other is not, assign accordingly.
		if isChannel(a) && !isChannel(b) {
			channel = a
			track = b
		} else {
			// Otherwise, just assign the first as track and the second as channel.
			track = a
			channel = b
		}
	}

	// NOTE: track is not given to use with default "snap install <name>" so put it to default latest now if we do get it with other variations of command
	// use that and put to default if not passed
	latestRevision := h.StoreClient.GetLatestRevisionByTrackAndChannel(entryName, track, channel)
	if len(latestRevision.Errors) > 0 {
		el.ExtendProtoError(latestRevision.Errors)
		return snapEntry, nil
	}

	return snapEntry, latestRevision
}

func isChannel(s string) bool {
	// List of allowed channels.
	allowedChannels := []string{"stable", "candidate", "beta", "edge"}
	return slices.Contains(allowedChannels, s)
}
