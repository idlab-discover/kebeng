package snap

import (
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/http"
	"slices"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	cerror "github.com/idlab-discover/kebeng/common/cerror"
	"github.com/idlab-discover/kebeng/services/gateway/internal/model"
	"github.com/idlab-discover/kebeng/services/gateway/internal/util"
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
			res.Result = &result
			resp.Responses = append(resp.Responses, res)
		case "install": // same as download just different result value, for now
			res, cerr := h.refreshSnapDownload(action, el)
			if cerr != nil {
				c.JSON(el.GetHTTPStatus(), gin.H{"error_list": el})
				return
			}
			result := "install"
			res.Result = &result
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
		res.Result = nil
		return &res, cerror.NewCustomError(cerror.InternalServerError, fmt.Sprintf("account error: %v", publisher.Errors))
	}

	downloadUrl := fmt.Sprintf("%s/download/%s", h.Config.StoreUrl, latestRevision.Id)

	raw, err := base64.RawURLEncoding.DecodeString(latestRevision.Sha3_384Encoded)
	if err != nil {
		el.Add(cerror.InternalServerError, fmt.Sprintf("error decoding sha3_384: %s", err.Error()))
	}

	hexSum := hex.EncodeToString(raw)

	res.InstanceKey = &action.InstanceKey
	res.SnapId = &snapEntry.Id
	res.Name = &snapEntry.SnapName
	res.Snap = &model.RefreshSnap{
		Architectures: &latestRevision.Architectures,
		SnapId:        &snapEntry.Id,
		Name:          &snapEntry.SnapName,
		Publisher: &model.Publisher{
			Username: publisher.Username,
			ID:       publisher.Id,
		},
		Download: &model.Download{
			URL:      &downloadUrl,
			Sha3_384: &hexSum,
			Size:     &latestRevision.Size,
		},
		Version:     &latestRevision.Version,
		Revision:    &latestRevision.SequenceNumber,
		Confinement: &snapEntry.Confinement,
		Type:        &snapEntry.Type,
		Base:        &snapEntry.Base,
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

	resp := h.StoreClient.RegisterSnapName(req.SnapName, "", "", "", req.IsPrivate, "", 0.0, req.Store, "", dryRun, accountUUID) // TODO: fill in empty fields once we know where to get the information from
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

func (h *Handler) ProcessSnapBuildAssertion(c *gin.Context) {
	el := cerror.NewErrorList()
	var req *model.SnapBuildAssertionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		el.Add(cerror.BadRequest, cerror.FormatBindError(err))
		c.JSON(el.GetHTTPStatus(), gin.H{"error_list": el})
		return
	}
	// SnapID is a path parameter
	snapID := c.Param("snap_id")
	if snapID == "" {
		el.Add(cerror.BadRequest, "snap_id is required")
		c.JSON(el.GetHTTPStatus(), gin.H{"error_list": el})
		return
	}

	// Process the snap build assertion with the snapID and req
	resp := h.AssertionClient.ProcessSnapBuildAssertion(req.Assertion)
	if len(resp.Errors) > 0 {
		el.ExtendProtoError(resp.Errors)
		//c.JSON(el.GetHTTPStatus(), gin.H{"error_list": el}) // This would be the prefered way to return the error, but the documentation handles this error differently
		c.JSON(http.StatusBadRequest, gin.H{"succes": false, "cerror": el})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"headers": gin.H{
			"authority-id":      resp.AuthorityId,
			"grade":             resp.Grade,
			"sign-key-sha3-384": resp.SignKeySha3_384,
			"snap-id":           resp.SnapId,
			"snap-sha3-384":     resp.SnapSha3_384,
			"snap-size":         resp.SnapSize,
			"timestamp":         resp.Timestamp,
			"type":              resp.Type,
		},
	})
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
		"status_details_url": fmt.Sprintf("%s/dev/api/snaps/%s/status", h.Config.StoreUrl, upload.Id), // FIX: change ClientIP to value in config
	})

	// After the status details URL is returned, we can proceed with creating a new revision
	// We need the sha3_384_encoded hash of the snap package
	metadata := h.StoreClient.GetObjectCustomMetadata("unscanned", req.UnscannedFileName)
	if len(metadata.Errors) > 0 {
		el.ExtendProtoError(metadata.Errors)
		return
	}

	logrus.Debugf("Metadata: %v", metadata)

	// Create a new revision for the snap upload
	revision := h.StoreClient.AddRevision(entry.SnapName, metadata.GetSha3_384Encoded(), uint64(req.BinaryFileSize), []string{"amd64"}, req.Channels, req.UnscannedFileName) // FIX: architectures should be passed from the request
	if len(revision.Errors) > 0 {
		el.ExtendProtoError(revision.Errors)
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

	binaryFile, err := c.FormFile("binary")
	if err != nil {
		el.Add(cerror.BadRequest, fmt.Sprintf("binary file not found: %s", err.Error()))
		c.JSON(el.GetHTTPStatus(), gin.H{"error-list": el})
		return
	}

	file, err := binaryFile.Open()
	if err != nil {
		el.Add(cerror.InternalServerError, fmt.Sprintf("error opening file: %s", err.Error()))
		c.JSON(el.GetHTTPStatus(), gin.H{"error-list": el})
		return
	}
	defer func() {
		if err := file.Close(); err != nil {
			el.Add(cerror.InternalServerError, fmt.Sprintf("error closing file: %s", err.Error()))
			c.JSON(el.GetHTTPStatus(), gin.H{"error-list": el})
		}
	}()

	// Upload file to the unscanned bucket, waiting for revision to be created
	resp := h.StoreClient.UnscannedUpload(c, file)
	if len(resp.Errors) > 0 {
		el.ExtendProtoError(resp.Errors)
		c.JSON(el.GetHTTPStatus(), gin.H{"error-list": el})
		return
	}

	if resp.GetTempFileName() == "" {
		el.Add(cerror.InternalServerError, "Upload failed: no ID returned")
		c.JSON(el.GetHTTPStatus(), gin.H{"error-list": el})
		return
	}

	// Return the upload ID and file information
	c.JSON(http.StatusOK, gin.H{
		"successful": true,
		"upload_id":  resp.GetTempFileName(),
		"filename":   binaryFile.Filename,
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

	response := h.StoreClient.SnapDownload(revisionId)
	if len(response.Errors) > 0 {
		el.ExtendProtoError(response.Errors)
		c.JSON(el.GetHTTPStatus(), gin.H{"error_list": el})
		return
	}

	filename := "downloaded.snap"
	if response.Revision != nil && response.Revision.SnapName != "" {
		// Optionally, you can incorporate the revision number or sequence too.
		filename = fmt.Sprintf("%s_%d.snap", response.Revision.SnapName, response.Revision.SequenceNumber)
	}

	// … write the body …
	c.Writer.Header().Set("Content-Disposition",
		fmt.Sprintf(`attachment; filename="%s"`, filename))
	c.Writer.Header().Set("Content-Type", "application/octet-stream")
	if _, err := c.Writer.Write(response.Data); err != nil {
		logrus.Error("error writing snap to response: ", err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error_list": el})
		return
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
