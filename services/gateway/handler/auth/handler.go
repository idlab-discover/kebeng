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
func (h *Handler) VerifyMacaroon(c *gin.Context) {
	// not implemented for now
	c.JSON(http.StatusNotImplemented, gin.H{"error_list": cerror.NewError(cerror.NotImplemented, "not implemented too many unknowns of implementation")})
	/*
			var req *message.VerifyRequest
			el := cerror.NewErrorList()
			if err := c.ShouldBindJSON(&req); err != nil {
				el.Add(cerror.BadRequest, cerror.FormatBindError(err))
				c.JSON(el.GetHTTPStatus(),gin.H{"error_list": el,})
				return
			}

		    // TODO: Don't know if you can get email from this actually or not (from discharge macaroon normally?)
			userEmail := auth.VerifyAndGetEmail(h.config, el, req.AuthData.Authorization)
		    // need userEmail to continue
			if userEmail == nil {
		        c.JSON(el.GetHTTPStatus(), gin.H{"error_list": el})
		        return
			}

		    // TODO: change account client to use errorList
			user, _ := h.AccountClient.GetAccountByEmail(*userEmail)
		    if user == nil {
		        el.Add(cerror.ResourceNotFound, fmt.Sprintf("user with email %s not found", *userEmail))
		        c.JSON(el.GetHTTPStatus(), gin.H{"error_list": el})
		        return
		    }

		    // TODO: check wtf these values have to be
		    // probably get them from the macaroon
		    _ = message.VerifyResponse{
		        Allowed:               true, // idk what dertemines this
		        DeviceRefreshRequired: false, // obsolote
		        RefreshRequired:       false, // check with expiry time of macaroon
		        Account: &message.VerifyAccount{
		            Email:       user.Email,
		            DisplayName: user.DisplayName,
		            OpenId:      "oid1234", // what id is this?
		            Verified:    true, // idk what this is
		        },
		        Device:      nil, // obsolete
		        LastAuth:    "2016-05-26T12:53:23Z", // should be from macaroon
		        Permissions: &[]string{"package_access", "package_manage", "package_push", "package_register", "package_release", "package_update"}, // should be from macaroon
		        SnapIds:     nil, // should be from macaroon
		        Channels:    nil, // should be from macaroon
		    }
		    return
	*/
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
	if len(*el) > 0 {
		c.JSON(http.StatusBadRequest,
			gin.H{
				"error_list": el,
			})
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
		el.ExtendStoreError(entriesResponse.Errors)
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
