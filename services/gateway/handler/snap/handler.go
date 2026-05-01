package snap

import (
	"crypto"
	"crypto/hmac"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

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

	var resp model.RefreshSnapResults

	for _, action := range req.Actions {
		switch action.Action {
		case "download":
			res, cerr := h.refreshInstallOrDownload(action, el)
			if cerr != nil {
				c.JSON(el.GetHTTPStatus(), gin.H{"error_list": el})
				return
			}
			res.Result = "download"
			resp.Results = append(resp.Results, res)

		case "install":
			res, cerr := h.refreshInstallOrDownload(action, el)
			if cerr != nil {
				c.JSON(el.GetHTTPStatus(), gin.H{"error_list": el})
				logrus.Errorf("error installing snap: %v", cerr)
				return
			}
			res.Result = "install"
			resp.Results = append(resp.Results, res)

		case "fetch-assertions":
			resp.Results = append(resp.Results, &model.RefreshSnapResult{
				Key: action.Key,
				// WARN: haven't yet been able to catch a network packet where the AssertionStreamURLs field not empty.
				// Not immediately able to deduct its function from the snapd source code either.
				AssertionStreamURLs: make([]string, 0),
				Result:              "fetch-assertions",
			})

		case "refresh":
			res, cerr := h.refreshRefresh(action, req.Context, el)
			if cerr != nil {
				c.JSON(el.GetHTTPStatus(), gin.H{"error-list": el})
				return
			}

			res.Result = "refresh"
			resp.Results = append(resp.Results, res)

		default:
			el.Add(cerror.NotImplemented, "Action not implemented")
			c.JSON(el.GetHTTPStatus(), gin.H{"error_list": el})
			return
		}
	}

	c.JSON(http.StatusOK, resp)
}

func (h *Handler) refreshInstallOrDownload(action *model.Action, el *cerror.ErrorList) (*model.RefreshSnapResult, *cerror.CustomError) {
	var res model.RefreshSnapResult
	var revision *storepb.GetRevisionResponse

	entry, latestRevision := h.getLatestRevisionByEntryName(el, action.Name, action.Channel)
	if el.HasError() || entry == nil {
		res.Result = "error"
		return &res, cerror.NewCustomError(cerror.ResourceNotFound, fmt.Sprintf("snap not found: %s", action.Name))
	}

	pub := h.AccountClient.GetAccountByID(entry.PublisherId)
	if len(pub.Errors) > 0 {
		res.Result = "error"
		return &res, cerror.NewCustomError(cerror.InternalServerError, "error occurred fetching publisher")
	}

	if action.CohortKey != "" {
		ckey, err := cohortKeyFromString(action.CohortKey)
		if err != nil {
			res.Result = "error"
			return &res, cerror.NewCustomError(cerror.BadRequest, err.Error())
		}

		valid, err := h.verifyCohortKey(*ckey)
		if err != nil {
			res.Result = "error"
			return &res, cerror.NewCustomError(cerror.InternalServerError, err.Error())
		}

		if !valid {
			res.Result = "error"
			return &res, cerror.NewCustomError(cerror.BadRequest, "invalid cohort key provided")
		}

		if entry.Id != ckey.SnapID {
			res.Result = "error"
			return &res, cerror.NewCustomError(cerror.BadRequest, "the provided cohort key does not apply to the provided snap")
		}

		// Find the closest 90-day milestone based on the cohort creation date
		closestMilestone := getMostRecent90DayMilestone(ckey.CreatedAt)
		revision = h.StoreClient.GetLatestRevisionBeforeDateById(closestMilestone, entry.Id)

		if len(revision.Errors) > 0 {
			res.Result = "error"
			return &res, cerror.NewCustomError(cerror.InternalServerError, fmt.Sprintf("error getting revision: %v", revision.Errors))
		}
	} else {
		if latestRevision == nil || el.HasError() {
			return nil, cerror.NewCustomError(cerror.ResourceNotFound, fmt.Sprintf("latest revision not found for: %s", action.Name))
		}
		revision = latestRevision
	}

	downloadUrl := fmt.Sprintf("%s/download/%s", h.Config.StoreUrl, revision.Id)

	raw, err := base64.RawURLEncoding.DecodeString(revision.Sha3_384Encoded)
	if err != nil {
		el.Add(cerror.InternalServerError, fmt.Sprintf("error decoding sha3_384: %s", err.Error()))
	}

	hexSum := hex.EncodeToString(raw)

	res.InstanceKey = action.InstanceKey
	res.SnapId = entry.Id
	res.Name = entry.SnapName
	res.Snap = &model.RefreshSnap{
		Architectures: revision.Architectures,
		SnapId:        entry.Id,
		Name:          entry.SnapName,
		Publisher: &model.Publisher{
			ID:          pub.Id,
			DisplayName: pub.DisplayName,
			Username:    pub.Username,
			Validation:  pub.Validation,
		},
		Download: &model.Download{
			URL:      &downloadUrl,
			Sha3_384: &hexSum,
			Size:     &revision.Size,
		},
		Version:     revision.Version,
		Revision:    revision.SequenceNumber,
		Confinement: entry.Confinement,
		Type:        entry.Type,
		Base:        entry.Base,
	}

	return &res, nil
}

func (h *Handler) refreshRefresh(action *model.Action, context []*model.Context, el *cerror.ErrorList) (*model.RefreshSnapResult, *cerror.CustomError) {
	// The workflow differs slightly from refreshInstallOrDownload: in this endpoint we only recieve the snap-id, not name.
	// Depending on the Snap, we may also need to add deltas

	var res model.RefreshSnapResult

	entry := h.StoreClient.GetEntryById(&storepb.GetEntryRequest{
		Id: &action.SnapID,
	})

	if len(entry.Errors) > 0 {
		el.ExtendProtoError(entry.Errors)
		res.Result = "error"
		return &res, cerror.NewCustomError(cerror.InternalServerError, fmt.Sprintf("store error: %v", entry.Errors))
	}

	publisher := h.AccountClient.GetAccountByID(entry.PublisherId)
	if len(publisher.Errors) > 0 {
		el.ExtendProtoError(publisher.Errors)
		res.Result = "error"
		return &res, cerror.NewCustomError(cerror.InternalServerError, fmt.Sprintf("account error: %v", publisher.Errors))
	}

	var latestRevision *storepb.GetRevisionResponse

	if action.CohortKey != "" {
		ckey, err := cohortKeyFromString(action.CohortKey)
		if err != nil {
			res.Result = "error"
			return &res, cerror.NewCustomError(cerror.BadRequest, err.Error())
		}

		valid, err := h.verifyCohortKey(*ckey)
		if err != nil {
			res.Result = "error"
			return &res, cerror.NewCustomError(cerror.InternalServerError, err.Error())
		}

		if !valid {
			res.Result = "error"
			return &res, cerror.NewCustomError(cerror.BadRequest, "invalid cohort key provided")
		}

		if entry.Id != ckey.SnapID {
			res.Result = "error"
			return &res, cerror.NewCustomError(cerror.BadRequest, "the provided cohort key does not apply to the provided snap")
		}

		closestMilestone := getMostRecent90DayMilestone(ckey.CreatedAt.UTC())
		latestRevision = h.StoreClient.GetLatestRevisionBeforeDateById(closestMilestone, entry.Id)

		if len(latestRevision.Errors) > 0 {
			res.Result = "error"
			return &res, cerror.NewCustomError(cerror.InternalServerError, fmt.Sprintf("error getting revision: %v", latestRevision.Errors))
		}
	} else {
		_, latestRevision = h.getLatestRevisionByEntryName(el, entry.SnapName)

		if el.HasError() {
			return nil, cerror.NewCustomError(cerror.ResourceNotFound, fmt.Sprintf("latest revision not found by name: %s", action.Name))
		}
	}

	downloadUrl := fmt.Sprintf("%s/download/%s", h.Config.StoreUrl, latestRevision.Id)

	raw, err := base64.RawURLEncoding.DecodeString(latestRevision.Sha3_384Encoded)
	if err != nil {
		el.Add(cerror.InternalServerError, fmt.Sprintf("error decoding sha3_384: %s", err.Error()))
	}

	deltas := make([]model.Delta, 0)

	// What revision does the client have right now?
	for _, c := range context {
		if c.SnapID == entry.Id {
			userRevision := h.StoreClient.GetRevisionByNameAndSequence(entry.SnapName, uint32(c.Revision))
			if len(userRevision.Errors) > 0 {
				res.Result = "error"
				return &res, cerror.NewCustomError(cerror.InternalServerError, fmt.Sprintf("error getting revision: %v", userRevision.Errors))
			}
			if latestRevision.SequenceNumber-userRevision.SequenceNumber != 1 {
				// NOTE: Delta saving
				// Currently, we only store deltas for steps of 1 revision at a time
				// Bigger steps can be ignored for now
				break
			}

			userRevisionId, erra := uuid.Parse(userRevision.Id)
			latestRevisionId, errb := uuid.Parse(latestRevision.Id)
			if erra != nil || errb != nil {
				res.Result = "error"
				return &res, cerror.NewCustomError(cerror.BadRequest, fmt.Sprintf("Unable to parse source and target revision IDs to uuids: %v, %v", erra, errb))
			}
			deltaInfo := h.StoreClient.GetDeltaByRevisionPair(userRevisionId, latestRevisionId, "xdelta3", el)
			if len(deltaInfo.Errors) > 0 {
				res.Result = "error"
				return &res, cerror.NewCustomError(cerror.InternalServerError, fmt.Sprintf("error getting delta details: %v", deltaInfo.Errors))
			}

			deltaRaw, err := base64.RawURLEncoding.DecodeString(deltaInfo.Sha3_384Encoded)
			if err != nil {
				el.Add(cerror.InternalServerError, fmt.Sprintf("error decoding sha3_384: %s", err.Error()))
			}

			deltaHexSum := hex.EncodeToString(deltaRaw)

			deltas = append(deltas, model.Delta{
				Format:   "xdelta3", // NOTE: we only support this type, canonical has more
				Sha3_384: deltaHexSum,
				Size:     latestRevision.Size,
				Source:   uint64(userRevision.SequenceNumber),
				Target:   uint64(latestRevision.SequenceNumber),
				URL:      fmt.Sprintf("%s/download-delta/%s/%s/%d-%d.xdelta3", h.Config.StoreUrl, "xdelta3", entry.SnapName, userRevision.SequenceNumber, latestRevision.SequenceNumber),
			})

			break
		}
	}

	hexSum := hex.EncodeToString(raw)

	res.InstanceKey = action.InstanceKey
	res.SnapId = entry.Id
	res.Name = entry.SnapName
	res.Snap = &model.RefreshSnap{
		Architectures: latestRevision.Architectures,
		SnapId:        entry.Id,
		Name:          entry.SnapName,
		Publisher: &model.Publisher{
			Username: publisher.Username,
			ID:       publisher.Id,
		},
		Download: &model.Download{
			Deltas:   deltas,
			URL:      &downloadUrl,
			Sha3_384: &hexSum,
			Size:     &latestRevision.Size,
		},
		Version:     latestRevision.Version,
		Revision:    latestRevision.SequenceNumber,
		Confinement: entry.Confinement,
		Type:        entry.Type,
		Base:        entry.Base,
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
	result := *model.NewFindSnapResponse()

	architecture, architectureIsPresent := c.GetQuery("architecture")
	if !architectureIsPresent {
		el.Add(cerror.BadRequest, "architecture is required")
		c.JSON(el.GetHTTPStatus(), gin.H{"error_list": el})
		return
	}
	architectureList := strings.Split(architecture, ",")

	channel, _ := c.GetQuery("channel")
	channelList := strings.Split(channel, ",")

	confinement, confinementIsPresent := c.GetQuery("confinement")
	if !confinementIsPresent {
		el.Add(cerror.BadRequest, "confinement is required")
		c.JSON(el.GetHTTPStatus(), gin.H{"error_list": el})
		return
	}
	confinementsList := strings.Split(confinement, ",")

	query, queryIsPresent := c.GetQuery("q")
	if !queryIsPresent {
		// TODO: The snap store itself does support a `snap find` invocation without query
		// But for this it returns "featured" snaps, which we don't mark in our database yet
		// For now we indicate query as required
		el.Add(cerror.BadRequest, "q (query) is required")
		c.JSON(el.GetHTTPStatus(), gin.H{"error_list": el})
		return
	}

	fields, _ := c.GetQuery("fields")
	fieldsList := strings.Split(fields, ",")

	private, _ := c.GetQuery("private")

	// NOTE: Once auth is implemented, a present email should mean properly logged in
	// No email present means not logged in
	email, emailIsPresent := c.Get("email")

	privateBool := private == "true"
	if !emailIsPresent && privateBool {
		el.Add(cerror.Unauthorized, "cannot filter on private snaps when not logged in!")
		c.JSON(el.GetHTTPStatus(), gin.H{"error_list": el})
		return
	}

	publisher_id := ""
	if emailIsPresent {
		acc := h.AccountClient.GetAccountByEmail(email.(string))
		if len(acc.Errors) > 0 {
			el.ExtendProtoError(acc.Errors)
			c.JSON(el.GetHTTPStatus(), gin.H{"error_list": el})
			return
		}

		publisher_id = acc.Id
	}

	entries := h.StoreClient.GetEntriesByQuery(
		query,
		architectureList,
		channelList,
		confinementsList,
		fieldsList,
		privateBool,
		publisher_id,
	)

	if len(entries.Errors) > 0 {
		el.ExtendProtoError(entries.Errors)
		c.JSON(el.GetHTTPStatus(), gin.H{"error_list": el})
		return
	}

	for _, entry := range entries.Entries {
		pub := h.AccountClient.GetAccountByID(entry.GetPublisherId())
		if len(pub.Errors) > 0 {
			el.ExtendProtoError(entries.Errors)
			c.JSON(el.GetHTTPStatus(), gin.H{"error_list": el})
			return
		}

		// GetLatestRevisionByTrackAndChannel is invoked with defaults
		lastRev := h.StoreClient.GetLatestRevisionByTrackAndChannel(
			entry.SnapName,
			"",
			"",
		)
		if len(lastRev.Errors) > 0 {
			el.ExtendProtoError(lastRev.Errors)
			c.JSON(el.GetHTTPStatus(), gin.H{"error_list": el})
			return
		}

		result.Results = append(
			result.Results,
			model.FindSnapResult{
				Name:   entry.SnapName,
				SnapID: entry.Id,
				Snap: model.Snap{
					SnapID:      entry.Id,
					Title:       entry.SnapName,
					Summary:     entry.Summary,
					Description: entry.Description,
					Publisher: model.Publisher{
						ID:          entry.PublisherId,
						DisplayName: pub.DisplayName,
						Username:    pub.Username,
						Validation:  pub.Validation,
					},
				},
				Revision: model.SnapRevision{
					Base:        entry.Base,
					Channel:     "stable",
					Confinement: entry.Confinement,
					Revision:    int(lastRev.SequenceNumber),
					Version:     entry.Version,
					Status:      entry.Status,
					Download: model.Download{
						Size: &lastRev.Size,
					},
				},
			},
		)
	}

	// Snap Find fails the content-type is `application/json;charset=utf-8`
	c.Header("Content-Type", "application/json")
	c.JSON(200, result)
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

func (h *Handler) CreateCohorts(c *gin.Context) {
	el := cerror.NewErrorList()

	var req model.CreateCohortsRequest
	var res model.CreateCohortsResult
	res.CohortKeys = make(map[string]string)

	if err := c.ShouldBindJSON(&req); err != nil {
		el.Add(cerror.BadRequest, cerror.FormatBindError(err))
		c.JSON(el.GetHTTPStatus(), gin.H{"error-list": el})
		return
	}

	var getEntryRequests storepb.GetEntriesRequest
	for _, snapname := range req.SnapNames {
		getEntryRequests.Entries = append(getEntryRequests.Entries, &storepb.GetEntryRequest{
			Name: &snapname,
		})
	}

	getEntriesResponse := h.StoreClient.GetEntries(&getEntryRequests)
	if len(getEntriesResponse.Errors) > 0 {
		el.ExtendProtoError(getEntriesResponse.Errors)
		c.JSON(el.GetHTTPStatus(), gin.H{"error-list": el})
		return
	}

	for _, entry := range getEntriesResponse.Entries {
		ckey := model.CohortKey{
			Version:   1,
			SnapID:    entry.Id,
			CreatedAt: time.Now(),
		}
		signedCkey, err := h.signCohortKey(ckey)
		if err != nil {
			el.Add(cerror.InternalServerError, "Failure generating cohort key signature")
			c.JSON(el.GetHTTPStatus(), gin.H{"error-list": el})
		}
		res.CohortKeys[entry.SnapName] = cohortKeyToString(signedCkey)
	}

	c.JSON(http.StatusOK, res)
}

func (h *Handler) DownloadDelta(c *gin.Context) {
	el := cerror.NewErrorList()
	deltaFormat := c.Param("delta-format")
	snapName := c.Param("snap-name")
	deltaName := c.Param("delta-name")
	if snapName == "" {
		el.Add(cerror.BadRequest, "snap name is required")
		c.JSON(el.GetHTTPStatus(), gin.H{"error-list": el})
		return
	}
	if deltaName == "" {
		el.Add(cerror.BadRequest, "delta name is required")
		c.JSON(el.GetHTTPStatus(), gin.H{"error-list": el})
		return
	}
	h.downloadDelta(c, deltaFormat, snapName, deltaName, el)
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

func (h *Handler) downloadDelta(c *gin.Context, deltaFormat string, snapName string, deltaName string, el *cerror.ErrorList) {
	if snapName == "" {
		el.Add(cerror.BadRequest, "snap name is required")
		c.JSON(el.GetHTTPStatus(), gin.H{"error-list": el})
		return
	}
	if deltaName == "" {
		el.Add(cerror.BadRequest, "delta name is required")
		c.JSON(el.GetHTTPStatus(), gin.H{"error-list": el})
		return
	}

	stream, err := h.StoreClient.DeltaDownloadStream(snapName, deltaName, deltaFormat)
	if err != nil {
		el.Add(cerror.InternalServerError, err.Error())
		c.JSON(el.GetHTTPStatus(), gin.H{"error-list": el})
		return
	}
	defer func() {
		err := stream.CloseSend()
		if err != nil {
			logrus.Errorf("failed to close stream: %v", err)
		}
	}()

	c.Writer.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, deltaName))
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
			splitChannelAndTrack := strings.Split(channelAndTrack[0], "/")
			if len(splitChannelAndTrack) == 1 {
				track = splitChannelAndTrack[0]
			} else if len(splitChannelAndTrack) == 2 || len(splitChannelAndTrack) == 3 {
				channel = splitChannelAndTrack[0]
				track = splitChannelAndTrack[1]
			} else {
				el.Add(cerror.InternalServerError, fmt.Sprintf("could not deduce track, risk and branch from given channel %s", channelAndTrack[0]))
				return nil, nil
			}
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

func cohortKeyToString(ck *model.CohortKey) string {
	cKey := fmt.Sprintf("%d %s %d %s", ck.Version, ck.SnapID, ck.CreatedAt.Unix(), ck.Signature)
	return base64.StdEncoding.EncodeToString([]byte(cKey))
}

func cohortKeyFromString(e string) (*model.CohortKey, error) {

	b, err := base64.StdEncoding.DecodeString(e)
	s := string(b)
	if err != nil {
		return nil, fmt.Errorf("Could not base64 decode cohort key")
	}
	var res model.CohortKey

	// NOTE: If future version of cohort keys are introduced
	parts := strings.Split(s, " ")
	if len(parts) != 4 {
		return nil, fmt.Errorf("The presented string \"%s\" is not a valid cohort key representation", s)
	}

	if version, err := strconv.ParseUint(parts[0], 10, 8); err != nil {
		return nil, fmt.Errorf("Could not parse \"%s\" into valid cohort version", parts[0])
	} else {
		res.Version = uint8(version)
	}

	res.SnapID = parts[1]

	if unixTime, err := strconv.ParseInt(parts[2], 10, 64); err != nil {
		return nil, fmt.Errorf("Could not represent \"%s\" as a valid unix timestamp.", parts[2])
	} else {
		res.CreatedAt = time.Unix(unixTime, 0)
	}

	res.Signature = parts[3]

	return &res, nil
}

func (h *Handler) signCohortKey(ckey model.CohortKey) (*model.CohortKey, error) {
	mac := hmac.New(crypto.SHA1.New, []byte(h.Config.CohortSigningKey))
	versionByteArray := make([]byte, 0)
	versionByteArray = append(versionByteArray, ckey.Version)
	_, err := mac.Write(versionByteArray)
	if err != nil {
		return nil, err
	}
	_, err = mac.Write([]byte(ckey.SnapID))
	if err != nil {
		return nil, err
	}

	signature := mac.Sum(nil)
	ckey.Signature = hex.EncodeToString(signature)
	return &ckey, nil
}

func (h *Handler) verifyCohortKey(ckey model.CohortKey) (bool, error) {
	if ckey.Signature == "" {
		return false, fmt.Errorf("an unsigned cohort key cannot be verified")
	}
	signedCkey, err := h.signCohortKey(ckey)
	if err != nil {
		return false, err
	}
	return ckey.Signature == signedCkey.Signature, nil
}

// getMostRecent90DayMilestone calculates the closest 90-day window starting point since a date
func getMostRecent90DayMilestone(origin time.Time) time.Time {
	const ninetyDays = 90 * 24 * time.Hour
	elapsed := time.Since(origin)
	k := int(elapsed / ninetyDays)
	return origin.Add(time.Duration(k) * ninetyDays)
}
