package assertion

import (
	"fmt"
	"io"
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

	revSHA := c.Param("rev_sha3_384")
	if revSHA == "" {
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

	resp := h.AssertionClient.GetSnapRevisionAssertionBySHA3_384(revSHA)
	if len(resp.Errors) > 0 {
		logrus.Errorf("GetSnapRevisionAssertion error: %v", resp.Errors)
		c.JSON(http.StatusInternalServerError, gin.H{"error_list": resp.Errors})
		return
	}

	c.Writer.Header().Set("Content-Type", "application/x.ubuntu.assertion")
	c.Writer.Header().Set("Content-Disposition",
		fmt.Sprintf(`attachment; filename="%s.assert"`, revSHA),
	)
	c.Writer.WriteHeader(http.StatusOK)
	io.WriteString(c.Writer, resp.Signature)
}

func (h *Handler) GetSnapDeclarationAssertion(c *gin.Context) {
	el := cerror.NewErrorList()

	snapID := c.Param("snap_id")
	if snapID == "" {
		el.Add(cerror.BadRequest, "missing snap_id")
		c.JSON(el.GetHTTPStatus(), gin.H{"error_list": el})
		return
	}
	maxFormat := c.Query("max-format")
	if maxFormat == "" {
		el.Add(cerror.BadRequest, "missing max-format")
		c.JSON(el.GetHTTPStatus(), gin.H{"error_list": el})
		return
	}

	/*
		resp := h.AssertionClient.GetSnapDeclarationAssertion(snapID)
		if len(resp.Errors) > 0 {
			logrus.Errorf("GetSnapDeclarationAssertion error: %v", resp.Errors)
			c.JSON(http.StatusInternalServerError, gin.H{"error_list": resp.Errors})
			return
		}
	*/
	c.Writer.Header().Set("Content-Type", "application/x.ubuntu.assertion")
	c.Writer.Header().Set("Content-Disposition",
		fmt.Sprintf(`attachment; filename="%s.assert"`, snapID),
	)
	c.Writer.WriteHeader(http.StatusOK)
	// io.WriteString(c.Writer, resp.Signature)
}
