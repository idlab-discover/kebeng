package auth

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/idlab-discover/kebeng/common/cerror"
	"github.com/idlab-discover/kebeng/services/gateway/internal/model"
	"github.com/idlab-discover/kebeng/services/gateway/internal/util"
	storepb "github.com/idlab-discover/kebeng/services/store/proto"
)

type Handler struct {
	*util.BaseHandler
}

// TODO: implement correctly to many unknows of what has to be included and what not, need good source
// or develop own authentication service, instead of using Ubuntu One SSO
func (h *Handler) VerifyMacaroon(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"error_list": cerror.NewError(cerror.NotImplemented, "not implemented too many unknowns of implementation")})
}

func (h *Handler) GenerateMacaroon(c *gin.Context) {
	el := cerror.NewErrorList()
	var req *model.GenerateMacaroonRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		el.Add(cerror.BadRequest, cerror.FormatBindError(err))
		c.JSON(el.GetHTTPStatus(), gin.H{"error_list": el})
		return
	}

	// validate input
	ValidateGenerateMacaroonRequest(req, el)
	if el.HasError() {
		c.JSON(http.StatusBadRequest, gin.H{"error_list": el})
		return
	}

	// check whether snapEntry exists else 404 not found
	entries := make([]*storepb.GetEntryRequest, len(req.Packages))
	for i, p := range req.Packages {
		entries[i] = &storepb.GetEntryRequest{
			Name: &p.Name,
			Id:   &p.SnapId,
		}
	}

	entriesResponse := h.StoreClient.GetEntries(&storepb.GetEntriesRequest{Entries: entries})
	if len(entriesResponse.Errors) > 0 {
		el.ExtendProtoError(entriesResponse.Errors)
		c.JSON(el.GetHTTPStatus(), gin.H{"error_list": el})
	}
	// mimic set to prevent duplicate ids in macaroon
	snapIDs := make(map[string]bool, len(entriesResponse.Entries))
	for _, e := range entriesResponse.Entries {
		snapIDs[e.Id] = true
	}

	macaroon := GenerateMacaroon(c, req, snapIDs, h.Config.MacaroonConfig)
	if len(macaroon.Errors) > 0 {
		el.Extend(macaroon.Errors)
		c.JSON(el.GetHTTPStatus(), gin.H{"error_list": el})
	}
	c.JSON(http.StatusOK, gin.H{"macaroon": macaroon.Macaroon})
}
