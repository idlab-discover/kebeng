package snap

import (
	"fmt"
	"net/http"

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
		if action.Action == "install" {
			logrus.Infof("%v", action)
			res, err := h.refreshSnapInstall(c, action, el)
			if err != nil {
				logrus.Errorf("Error refreshing snap: %v", err)
				c.JSON(el.GetHTTPStatus(), gin.H{"error_list": el})
				return
			}
			resp.Responses = append(resp.Responses, res)
		} else {
			el.Add(cerror.NotImplemented, "Action not implemented")
			c.JSON(el.GetHTTPStatus(), gin.H{"error_list": el})
			return
		}
	}

	c.JSON(http.StatusOK, resp)
}

// TODO: Implement this function properly
func (h *Handler) refreshSnapInstall(c *gin.Context, action *model.Action, el *cerror.ErrorList) (*model.RefreshSnapResult, error) {
	var res model.RefreshSnapResult

	// should only get 1 entry since name is unique
	snapEntries := h.StoreClient.GetEntries(&storepb.GetEntriesRequest{
		Entries: []*storepb.GetEntryRequest{
			{
				Name:                action.Name,
				PreloadAssociations: []string{"REVISIONS"},
			},
		},
	})
	if len(snapEntries.Errors) > 0 {
		el.ExtendStoreError(snapEntries.Errors)
		res.Result = nil
		return &res, fmt.Errorf("store error: %v", snapEntries.Errors)
	}
	if len(snapEntries.Entries) == 0 {
		el.Add(cerror.ResourceNotFound, "Snap not found")
		res.Result = nil
		return &res, fmt.Errorf("snap not found")
	}
	snapEntry := snapEntries.Entries[0]

	// NOTE: track is not given to use with default "snap install <name>" so put it to default latest now if we do get it with other variations of command
	// use that and put to default if not passed
	latestRevision := h.StoreClient.GetLatestRevision(*action.Name, "latest", action.Channel)
	if len(latestRevision.Errors) > 0 {
		el.ExtendStoreError(latestRevision.Errors)
		res.Result = nil
		return &res, fmt.Errorf("store error: %v", latestRevision.Errors)
	}

	// if publisher not found we should error this is not safe if we don't know who published it
	publisher := h.AccountClient.GetAccountByID(snapEntry.PublisherId)
	if len(publisher.Errors) > 0 {
		el.ExtendAccountError(publisher.Errors)
		res.Result = nil
		return &res, fmt.Errorf("account error: %v", publisher.Errors)
	}

	result := "install"

	sequenceNumber := int(latestRevision.SequenceNumber)

	if h.Config.StoreUrl == "" {
		el.Add(cerror.InternalServerError, "store URL not set")
		res.Result = nil
		return &res, fmt.Errorf("store URL not set")
	}
	downloadUrl := fmt.Sprintf("%s/download/%s", h.Config.StoreUrl, latestRevision.Id)

	res.Result = &result
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
			Sha3_384: &latestRevision.Sha3_384,
			Size:     &latestRevision.Size,
		},
		Version:     &latestRevision.Version,
		Revision:    &sequenceNumber,
		Confinement: snapEntry.Confinement,
		Type:        snapEntry.Type,
		Base:        snapEntry.Base,
	}
	return &res, nil
}

func (h *Handler) DownloadSnap(c *gin.Context) {
	el := cerror.NewErrorList()
	revisionID := c.Param("revision_id")
	if revisionID == "" {
		el.Add(cerror.BadRequest, "revision_id is required")
		c.JSON(el.GetHTTPStatus(), gin.H{"error_list": el})
		return
	}

	response := h.StoreClient.SnapDownload(revisionID)
	if len(response.Errors) > 0 {
		el.ExtendStoreError(response.Errors)
		c.JSON(el.GetHTTPStatus(), gin.H{"error_list": el})
		return
	}

	filename := "downloaded.snap"
	logrus.Infof("*************************** revision: %+v", response.Revision)
	if response.Revision != nil && response.Revision.SnapName != "" {
		// Optionally, you can incorporate the revision number or sequence too.
		filename = fmt.Sprintf("%s_%d.snap", response.Revision.SnapName, response.Revision.SequenceNumber)
	}

	// set correct headers for file download
	c.Writer.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
	c.Writer.Header().Set("Content-Type", "application/octet-stream")
	_, err := c.Writer.Write(response.Data)
	if err != nil {
		logrus.Error("error writing snap to response: ", err)
		el.Add(cerror.InternalServerError, "error writing snap to response")
		c.JSON(el.GetHTTPStatus(), gin.H{"error_list": el})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "snap downloaded successfully"})
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
	account := h.BaseHandler.AccountClient.GetAccountByEmail(email.(string))
	if len(account.Errors) > 0 {
		el.ExtendAccountError(account.Errors)
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

	resp := h.StoreClient.RegisterSnapName(req.SnapName, req.IsPrivate, req.Store, dryRun, accountUUID)
	if len(resp.Errors) > 0 {
		el.ExtendStoreError(resp.Errors)
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
		el.ExtendAssertionError(resp.Errors)
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
		c.JSON(el.GetHTTPStatus(), gin.H{"error_list": el})
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
	account := h.BaseHandler.AccountClient.GetAccountByEmail(email.(string))
	if len(account.Errors) > 0 {
		el.ExtendAccountError(account.Errors)
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
	entry := h.StoreClient.RegisterSnapName(req.Name, false, "", true, accountUUID)
	if len(entry.Errors) > 0 {
		el.ExtendStoreError(entry.Errors)
		c.JSON(el.GetHTTPStatus(), gin.H{"error-list": el})
		return
	}

	if entry.SnapName == "" {
		el.AddCustomError(cerror.NewCustomError(cerror.ResourceNotFound, "Snap name not found for name="+req.Name))
		c.JSON(el.GetHTTPStatus(), gin.H{"error-list": el})
		return
	}

	parsedEntryUUID, err := uuid.Parse(entry.Id)
	if err != nil {
		el.Add(cerror.BadRequest, "invalid entry ID format")
		c.JSON(el.GetHTTPStatus(), gin.H{"error-list": el})
		return
	}

	// Create a new snap upload with status "pending"
	upload := h.StoreClient.AddUpload(entry.SnapName, parsedEntryUUID, "pending", accountUUID)
	if len(upload.Errors) > 0 {
		el.ExtendStoreError(upload.Errors)
		c.JSON(el.GetHTTPStatus(), gin.H{"error-list": el})
		return
	}

	logrus.Infof("status details url: %s", upload.StatusDetailsUrl)

	c.JSON(http.StatusOK, gin.H{
		"success":            true,
		"snap_name":          entry.SnapName,
		"status_details_url": fmt.Sprintf("https://%s%s", c.ClientIP(), upload.StatusDetailsUrl),
	})
}

func (h *Handler) UnscannedUpload(c *gin.Context) {
	el := cerror.NewErrorList()

	header, err := c.FormFile("binary")
	if err != nil {
		el.Add(cerror.BadRequest, "Missing file in form data: "+err.Error())
		c.JSON(el.GetHTTPStatus(), gin.H{"error-list": el})
		return
	}

	file, err := header.Open()
	if err != nil {
		el.Add(cerror.InternalServerError, "Error opening file: "+err.Error())
		c.JSON(el.GetHTTPStatus(), gin.H{"error-list": el})
		return
	}
	defer func() {
		if err := file.Close(); err != nil {
			el.Add(cerror.InternalServerError, "Error closing file: "+err.Error())
			c.JSON(el.GetHTTPStatus(), gin.H{"error-list": el})
		}
	}()

	// Upload file to the unscanned bucket, waiting for revision to be created
	resp := h.StoreClient.UnscannedUpload(file)
	if len(resp.Errors) > 0 {
		el.ExtendStoreError(resp.Errors)
		c.JSON(el.GetHTTPStatus(), gin.H{"error-list": el})
		return
	}

	if resp.GetTempFileName() == "" {
		el.Add(cerror.InternalServerError, "Upload failed: no ID returned")
		c.JSON(el.GetHTTPStatus(), gin.H{"error-list": el})
		return
	}

	// Test: gewoon even bevestigen dat het gelukt is
	c.JSON(http.StatusOK, gin.H{
		"successful": true,
		"upload_id":  uuid.New().String(), // FIX: this should be the ID of the revision that is created
		"filename":   header.Filename,
		"size":       header.Size,
	})
}

// func (h *Handler) GetStatus(c *gin.Context) {
// 	el := cerror.NewErrorList()

// 	// Get the revision ID from the URL
// 	revisionID := c.Param("rev_id")
// 	if revisionID == "" {
// 		el.Add(cerror.BadRequest, "revision_id is required")
// 		c.JSON(el.GetHTTPStatus(), gin.H{"error-list": el})
// 		return
// 	}

// 	// Get the status of the revision
// 	status, err := h.StoreClient.GetRevisionStatus(revisionID)
// 	if err != nil {
// 		el.Add(cerror.InternalServerError, "Failed to get revision status: "+err.Error())
// 		c.JSON(el.GetHTTPStatus(), gin.H{"error-list": el})
// 		return
// 	}

// 	c.JSON(http.StatusOK, gin.H{
// 		"status": status,
// 	})
// }
