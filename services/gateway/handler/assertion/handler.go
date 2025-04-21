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
	_, err := io.WriteString(c.Writer, resp.Signature)
	if err != nil {
		logrus.Errorf("Error writing response: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to write response"})
		return
	}
}

func (h *Handler) GetSnapDeclarationAssertion(c *gin.Context) {
	el := cerror.NewErrorList()

	snapID := c.Param("snap_id")
	if snapID == "" {
		el.Add(cerror.BadRequest, "missing snap_id")
		c.JSON(el.GetHTTPStatus(), gin.H{"error_list": el})
		return
	}

	// TODO: figure out what to do with the series maybe check specifically for the series
	series := c.Param("series")
	if series == "" {
		el.Add(cerror.BadRequest, "missing series")
		c.JSON(el.GetHTTPStatus(), gin.H{"error_list": el})
		return
	}

	resp := h.AssertionClient.GetSnapDeclarationAssertionBySnapID(snapID)
	if len(resp.Errors) > 0 {
		logrus.Errorf("GetSnapDeclarationAssertion error: %v", resp.Errors)
		c.JSON(http.StatusInternalServerError, gin.H{"error_list": resp.Errors})
		return
	}
	c.Writer.Header().Set("Content-Type", "application/x.ubuntu.assertion")
	c.Writer.Header().Set("Content-Disposition",
		fmt.Sprintf(`attachment; filename="%s.assert"`, snapID),
	)
	c.Writer.WriteHeader(http.StatusOK)
	_, err := io.WriteString(c.Writer, resp.Signature)
	if err != nil {
		logrus.Errorf("Error writing response: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to write response"})
		return
	}
}

func (h *Handler) GetAccountAssertion(c *gin.Context) {
	el := cerror.NewErrorList()

	accountID := c.Param("account_id")
	if accountID == "" {
		el.Add(cerror.BadRequest, "missing account_id")
		c.JSON(el.GetHTTPStatus(), gin.H{"error_list": el})
		return
	}

	maxFormat := c.Query("max-format")
	if maxFormat == "" {
		el.Add(cerror.BadRequest, "missing max-format")
		c.JSON(el.GetHTTPStatus(), gin.H{"error_list": el})
		return
	}
	resp := h.AssertionClient.GetAccountAssertionByAccountID(accountID)
	if len(resp.Errors) > 0 {
		logrus.Errorf("GetSnapDeclarationAssertion error: %v", resp.Errors)
		c.JSON(http.StatusInternalServerError, gin.H{"error_list": resp.Errors})
		return
	}
	c.Writer.Header().Set("Content-Type", "application/x.ubuntu.assertion")
	c.Writer.Header().Set("Content-Disposition",
		fmt.Sprintf(`attachment; filename="%s.assert"`, accountID),
	)
	c.Writer.WriteHeader(http.StatusOK)
	_, err := io.WriteString(c.Writer, resp.Signature)
	if err != nil {
		logrus.Errorf("Error writing response: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to write response"})
		return
	}
}

func (h *Handler) GetAccountKeyAssertion(c *gin.Context) {
	el := cerror.NewErrorList()

	publicKeySha := c.Param("public_key_sha3_384")
	if publicKeySha == "" {
		el.Add(cerror.BadRequest, "missing publicKeySha")
		c.JSON(el.GetHTTPStatus(), gin.H{"error_list": el})
		return
	}

	maxFormat := c.Query("max-format")
	if maxFormat == "" {
		el.Add(cerror.BadRequest, "missing max-format")
		c.JSON(el.GetHTTPStatus(), gin.H{"error_list": el})
		return
	}
	resp := h.AssertionClient.GetAccountKeyAssertionByPublicKeySha(publicKeySha)
	if len(resp.Errors) > 0 {
		logrus.Errorf("GetSnapDeclarationAssertion error: %v", resp.Errors)
		c.JSON(http.StatusInternalServerError, gin.H{"error_list": resp.Errors})
		return
	}
	c.Writer.Header().Set("Content-Type", "application/x.ubuntu.assertion")
	c.Writer.Header().Set("Content-Disposition",
		fmt.Sprintf(`attachment; filename="%s.assert"`, publicKeySha),
	)
	c.Writer.WriteHeader(http.StatusOK)
	_, err := io.WriteString(c.Writer, resp.Signature)
	if err != nil {
		logrus.Errorf("Error writing response: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to write response"})
		return
	}
}
