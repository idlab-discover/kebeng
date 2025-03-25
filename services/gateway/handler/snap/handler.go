package snap

import (
	"net/http"

	"github.com/gin-gonic/gin"
	cerror "github.com/idlab-discover/kebeng/common/cerror"
	"github.com/idlab-discover/kebeng/services/gateway/handler/util"
	"github.com/idlab-discover/kebeng/services/gateway/internal/message"
	storepb "github.com/idlab-discover/kebeng/services/store/proto"
	"github.com/sirupsen/logrus"
)

type Handler struct {
	*util.BaseHandler
}

// TODO: Implement this function properly
// Right now it's just a placeholder to make sure Snapcraft can be installed in the lxc container
func (h *Handler) RequestStoreDeviceNonce(c *gin.Context) {
	c.JSON(http.StatusOK, message.RequestStoreDeviceNonceRes{Nonce: "this-nonce"})
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
	var req message.RefreshSnapRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		el.Add(cerror.BadRequest, cerror.FormatBindError(err))
		c.JSON(el.GetHTTPStatus(), gin.H{"error_list": el})
		return
	}

	var resp message.RefreshSnapResponses
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
func (h *Handler) refreshSnapInstall(c *gin.Context, action *message.Action, el *cerror.ErrorList) *message.RefreshSnapResult {
	var res message.RefreshSnapResult

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
	res.Snap = &message.RefreshSnap{
		Architectures: &architectures,
		SnapId:        &snapEntry.Id,
		Name:          &snapEntry.SnapName,
		Publisher: &message.Publisher{
			Username: publisher.Username,
			ID:       publisher.Id,
		},
		Download: &message.Download{
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
	var req message.FindSnapsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		el.Add(cerror.BadRequest, cerror.FormatBindError(err))
		c.JSON(el.GetHTTPStatus(), gin.H{"error_list": el})
		return
	}
}

func (h *Handler) RegisterSnapName(c *gin.Context) {
	el := cerror.NewErrorList()
	var req message.RegisterSnapNameReq
	if err := c.ShouldBindJSON(&req); err != nil {
		el.Add(cerror.BadRequest, cerror.FormatBindError(err))
		c.JSON(el.GetHTTPStatus(), gin.H{"error_list": el})
		return
	}

	dryRun := c.Query("dry_run") == "1"
	resp := h.StoreClient.RegisterSnapName(req.SnapName, req.IsPrivate, req.Store, dryRun)
	if len(resp.Errors) > 0 {
		el.ExtendStoreError(resp.Errors)
		c.JSON(el.GetHTTPStatus(), gin.H{"error_list": el})
		return
	}

	statusCode := http.StatusCreated
	if dryRun {
		statusCode = http.StatusOK
	}

	c.JSON(statusCode, message.RegisterSnapNameRes{SnapId: resp.Id, SnapName: req.SnapName})
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
	var req *message.SnapBuildAssertionReq
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
