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
			res := h.refreshSnapInstall(c, action, el)
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
func (h *Handler) refreshSnapInstall(c *gin.Context, action *model.Action, el *cerror.ErrorList) *model.RefreshSnapResult {
	var res model.RefreshSnapResult

	// Get snap entry -> will return 1 snap entry
	snapEntries := h.StoreClient.GetEntries(&storepb.GetEntriesRequest{
		Entries: []*storepb.GetEntryRequest{
			{
				Name:                action.Name,
				Id:                  action.SnapID,
				PreloadAssociations: []string{"REVISIONS"},
			},
		},
	})
	if len(snapEntries.Errors) > 0 {
		el.ExtendStoreError(snapEntries.Errors)
		res.Result = nil
		return &res
	}

	// Get the snap entry -> should only be 1
	snapEntry := snapEntries.Entries[0]

	// Get a set of all the architectures of the snapEntry's revisions
	architectureSet := make(map[string]struct{})
	for _, snapRevision := range snapEntry.Revisions {
		for _, arch := range snapRevision.Architectures {
			architectureSet[arch] = struct{}{}
		}
	}

	// Convert the set to a slice
	architectures := make([]string, 0, len(architectureSet))
	for arch := range architectureSet {
		architectures = append(architectures, arch)
	}

	latest_revision := float64(snapEntry.Revisions[len(snapEntry.Revisions)-1].Sequence)

	// Get the publisher of the snap entry
	publisher := h.AccountClient.GetAccountByID(snapEntry.PublisherId)

	result := "install"

	res.Result = &result
	res.InstanceKey = &action.InstanceKey
	res.SnapId = &snapEntry.Id
	res.Name = &snapEntry.SnapName
	res.Snap = &model.RefreshSnap{
		Architectures: &architectures,
		SnapId:        &snapEntry.Id,
		Name:          &snapEntry.SnapName,
		Publisher: &model.Publisher{
			Username: publisher.Username,
			ID:       publisher.Id,
		},
		Download: &model.Download{
			URL:      nil, // TODO: implement
			Sha3_384: nil, // TODO: implement
			Size:     nil, // TODO: implement
		},
		Version:     nil, // TODO: implement
		Revision:    &latest_revision,
		Confinement: snapEntry.Confinement,
		Type:        snapEntry.Type,
		Base:        snapEntry.Base,
	}
	return &res
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

	resp := h.StoreClient.RegisterSnapName(req.Name, false, "", true, accountUUID)
	if len(resp.Errors) > 0 {
		el.ExtendStoreError(resp.Errors)
		c.JSON(el.GetHTTPStatus(), gin.H{"error-list": el})
		return
	}

	if resp.SnapName == "" {
		el.AddCustomError(cerror.NewCustomError(cerror.ResourceNotFound, "Snap name not found for name="+req.Name))
		c.JSON(el.GetHTTPStatus(), gin.H{"error-list": el})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "snap_name": resp.SnapName})
}

func (h *Handler) UnscannedUpload(c *gin.Context) {
	el := cerror.NewErrorList()

	c.Request.ParseMultipartForm(100 << 20) // 100 MB limiet

	if c.Request.MultipartForm != nil {
		for key, files := range c.Request.MultipartForm.File {
			fmt.Printf("Multipart file field: %s\n", key)
			for _, f := range files {
				fmt.Printf("  File name: %s, size: %d bytes\n", f.Filename, f.Size)
			}
		}
	}

	header, err := c.FormFile("binary") // vervang 'snap' indien nodig
	if err != nil {
		el.Add(cerror.BadRequest, "Missing file in form data: "+err.Error())
		c.JSON(el.GetHTTPStatus(), gin.H{"error_list": el})
		return
	}

	// Test: gewoon even bevestigen dat het gelukt is
	c.JSON(http.StatusOK, gin.H{
		"success":  true,
		"filename": header.Filename,
		"size":     header.Size,
	})
}
