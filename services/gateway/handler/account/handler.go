package account

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/idlab-discover/kebeng/common/cerror"
	"github.com/idlab-discover/kebeng/services/gateway/handler/auth"
	"github.com/idlab-discover/kebeng/services/gateway/internal/model"
	"github.com/idlab-discover/kebeng/services/gateway/internal/util"
	storepb "github.com/idlab-discover/kebeng/services/store/proto"
	"gopkg.in/macaroon.v2"
)

type Handler struct {
	*util.BaseHandler
}

func (h *Handler) AddAccount(c *gin.Context) {
	var req model.CreateAccountRequest
	el := cerror.NewErrorList()
	if err := c.ShouldBindJSON(&req); err != nil {
		el.Add(cerror.BadRequest, cerror.FormatBindError(err))
		c.JSON(el.GetHTTPStatus(), gin.H{"error_list": el})
		return
	}
	account := h.AccountClient.AddAccount(req.DisplayName, req.Username, req.Email, req.HashedPassword)
	if len(account.Errors) > 0 {
		el.ExtendProtoError(account.Errors)
		c.JSON(el.GetHTTPStatus(), gin.H{"error_list": el})
		return
	}

	c.JSON(http.StatusOK, model.CreateAccountResponse{Id: account.Id})
}

func (h *Handler) GetAccount(c *gin.Context) {
	el := cerror.NewErrorList()
	email, isThere := c.Get("email")
	if !isThere {
		el.Add(cerror.BadRequest, "missing email")
		c.JSON(el.GetHTTPStatus(), gin.H{"error_list": el})
		return
	}

	account := h.AccountClient.GetAccountByEmail(email.(string))
	if len(account.Errors) > 0 {
		el.ExtendProtoError(account.Errors)
		c.JSON(http.StatusInternalServerError, gin.H{"error_list": el})
		return
	}
	accId, err := uuid.Parse(account.Id)
	if err != nil {
		// should never happen really
		el.Add(cerror.InternalServerError, err.Error())
		c.JSON(el.GetHTTPStatus(), gin.H{"error_list": el})
		return
	}

	keys := h.AccountClient.GetAccountKeysByAccountID(account.Id)
	if len(keys.Errors) > 0 {
		el.ExtendProtoError(keys.Errors)
		c.JSON(http.StatusInternalServerError, gin.H{"error_list": el})
		return
	}
	// convert to model.AccountKey
	messageKeys := make([]model.AccountKey, len(keys.Keys))
	for i, k := range keys.Keys {
		messageKeys[i] = model.AccountKey{
			Name:            k.Name,
			PublicKeySHA384: k.Sha3384,
			Since:           k.Since.AsTime(),
			Until:           k.Until.AsTime(),
		}
	}

	// Get snaps

	entries := h.StoreClient.GetEntriesByAccountID(account.Id)
	if len(entries.Errors) > 0 {
		el.ExtendProtoError(entries.Errors)
		c.JSON(http.StatusInternalServerError, gin.H{"error_list": el})
		return
	}

	// Get all revisions for every snapEntry
	snapEntries := make([]*storepb.GetRevisionsByEntryIdRequest, len(entries.Entries))
	for i, e := range entries.Entries {
		snapEntries[i] = &storepb.GetRevisionsByEntryIdRequest{EntryId: e.Id}
	}

	// do 1 request where we get all the snapRevisions for every snapEntry
	revisions := h.StoreClient.GetRevisionsByEntryIds(
		&storepb.GetRevisionsByEntryIdRequests{
			Requests: snapEntries,
		})
	if len(revisions.Errors) > 0 {
		el.ExtendProtoError(revisions.Errors)
		c.JSON(http.StatusInternalServerError, gin.H{"error_list": el})
		return
	}

	// according to the docs a Publisher is only linked to a snapEntry
	// get all publishers per snap
	accountIds := make([]string, len(entries.Entries))
	for i, e := range entries.Entries {
		accountIds[i] = e.PublisherId
	}
	publishers := h.AccountClient.GetAccountsByIds(accountIds)
	if len(publishers.Errors) > 0 {
		el.ExtendProtoError(publishers.Errors)
		c.JSON(http.StatusInternalServerError, gin.H{"error_list": el})
		return
	}

	// Now we have all the data we need to fill in the snaps object
	// start filling in
	snaps := make(map[string]map[string]model.Snap)

	// map publishers by ID for quick lookup
	publisherMap := make(map[string]*model.Publisher)
	for _, p := range publishers.Accounts {
		publisherMap[p.Id] = &model.Publisher{
			ID:          p.Id,
			DisplayName: p.DisplayName,
			Username:    p.Username,
			Validation:  util.GetString(p.Validation),
		}
	}

	// map revisions by entry ID for quick lookup
	revisionMap := make(map[string][]model.SnapRevision)
	for _, revs := range revisions.Responses {
		entryID := revs.EntryId
		for _, rev := range revs.Revisions {
			revisionMap[entryID] = append(revisionMap[entryID], model.SnapRevision{
				Revision:      int(rev.SequenceNumber),
				Version:       rev.Version,
				Status:        rev.Status,
				Architectures: rev.Architectures,
				// Channels:      rev.Channels, // TODO: see how to get them should not be in revision object
			})
		}
	}

	for _, e := range entries.Entries {
		series := e.Base       // Example: "16" //TODO: check if this is series normally but i think its the same as base?
		snapName := e.SnapName // Example: "hello-published"

		// Ensure series exists in the map
		if snaps[series] == nil {
			snaps[series] = make(map[string]model.Snap)
		}

		// Get publisher
		publisher, exists := publisherMap[e.PublisherId]
		if !exists {
			// should always exist normally otherwise something went wrong in previous processes
			publisher = &model.Publisher{
				ID:          e.PublisherId, // Fallback to ID only
				DisplayName: "Unknown Publisher",
				Username:    "unknown",
				Validation:  "unverified",
			}
		}

		// Get revisions for this snap entry
		latestRevisions := revisionMap[e.Id]

		// TODO: fix this
		snap := model.Snap{
			Status:          e.Status,
			Price:           e.Price,
			Since:           e.Since.AsTime(),
			SnapID:          e.Id,
			Private:         e.Private, // this isn't the best yet but don't know what to do if its a nil value set to true or false?
			IconURL:         e.IconUrl,
			Publisher:       *publisher,
			LatestComments:  []model.SnapComment{}, // No comments available yet
			LatestRevisions: latestRevisions,
		}

		// Assign snap to the correct series and snap name
		snaps[series][snapName] = snap
	}

	// leaving out the deprecated fields for now
	resp := model.AccountResponse{
		ID:          accId,
		DisplayName: account.DisplayName,
		Email:       account.Email,
		Username:    account.Username,

		AccountKeys: messageKeys,
		Snaps:       snaps,
		Stores:      nil,
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) PatchAccount(c *gin.Context) {
	el := cerror.NewErrorList()
	email, ok := c.Get("email")
	if !ok {
		el.Add(cerror.BadRequest, "missing email")
		c.JSON(el.GetHTTPStatus(), gin.H{"error_list": el})
		return
	}

	rootMacaroon, ok := c.Get("macaroon")
	if !ok {
		el.Add(cerror.BadRequest, "missing macaroon")
		c.JSON(el.GetHTTPStatus(), gin.H{"error_list": el})
		return
	}

	// check permissions of macaroon to include "edit_account" permission
	if !auth.HasPermission(rootMacaroon.(*macaroon.Macaroon), "edit_account") {
		el.Add(cerror.ResourceForbidden, "missing permission to edit account")
		c.JSON(el.GetHTTPStatus(), gin.H{"error_list": el})
		return
	}

	var req *model.AccountPatchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		el.Add(cerror.BadRequest, cerror.FormatBindError(err))
		c.JSON(el.GetHTTPStatus(), gin.H{"error_list": el})
		return
	}

	// patch account by email
	account := h.AccountClient.PatchAccountByEmail(email.(string), req.ShortNameSpace)
	if len(account.Errors) > 0 {
		el.ExtendProtoError(account.Errors)
		c.JSON(el.GetHTTPStatus(), gin.H{"error_list": el})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}
