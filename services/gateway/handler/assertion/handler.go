package assertion

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/idlab-discover/kebeng/common/cerror"
	"github.com/idlab-discover/kebeng/services/gateway/internal/util"
	"github.com/sirupsen/logrus"
)

type Handler struct {
	*util.BaseHandler
}

func (h *Handler) GetSnapRevisionAssertion(c *gin.Context) {
	el := cerror.NewErrorList()
	rev_sha3_384 := c.Param("rev_sha3_384") // sha3_384 hash of the revision
	if rev_sha3_384 == "" {
		el.Add(cerror.BadRequest, "missing rev_sha3_384")
		c.JSON(el.GetHTTPStatus(), gin.H{"error_list": el})
		return
	}
	maxFormat := c.Query("max-format")
	if maxFormat == "" {
		el.Add(cerror.BadRequest, "missing max-format")
		c.JSON(el.GetHTTPStatus(), gin.H{"error_list": el})
		return
	}

	logrus.Infof("GetSnapRevisionAssertion: rev_sha3_384=%s, max-format=%s", rev_sha3_384, maxFormat)
	/*
		snapRevisionAssertion, err := h.GetSnapRevisionAssertion(rev_sha3_384)
		if err != nil {
			el.Add(cerror.InternalServerError, err.Error())
			c.JSON(el.GetHTTPStatus(), gin.H{"error_list": el})
			return
		}
	*/
	c.JSON(http.StatusOK, gin.H{"snap_revision_assertion": "TODO"}) // TODO: replace with actual data
}
