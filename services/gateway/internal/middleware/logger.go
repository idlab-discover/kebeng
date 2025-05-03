package middleware

import (
	"bytes"
	"io"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// bodyLogWriter wraps gin.ResponseWriter to capture the response body.
type bodyLogWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

// Write captures the response body (only for text/JSON) and writes it out.
func (w *bodyLogWriter) Write(b []byte) (int, error) {
	ct := w.Header().Get("Content-Type")
	if strings.HasPrefix(ct, "application/json") || strings.HasPrefix(ct, "text/") {
		// Only buffer JSON/text responses
		w.body.Write(b)
	}
	return w.ResponseWriter.Write(b)
}

func RequestLoggerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		method := c.Request.Method
		path := c.Request.URL.RequestURI()
		reqID := time.Now().UnixNano()

		logrus.Infof("=== START REQUEST %d %s %s ===", reqID, method, path)

		// log request body (unless it’s a binary multipart or octet stream)
		ct := c.ContentType()
		if ct == "multipart/form-data" || ct == "application/octet-stream" {
			logrus.Infof("[%d] Skipping request body for %s", reqID, ct)
		} else {
			var buf bytes.Buffer
			tee := io.TeeReader(c.Request.Body, &buf)
			if body, _ := io.ReadAll(tee); len(body) > 0 {
				logrus.Infof("Request Body:\n%s", string(body))
			}
			c.Request.Body = io.NopCloser(&buf)
		}

		// log headers
		hdr := c.Request.Header.Clone()
		hdr.Del("Authorization")
		logrus.Infof("Request Headers: %v", hdr)

		// swap in a ResponseWriter that captures the body
		blw := &bodyLogWriter{
			ResponseWriter: c.Writer,
			body:           new(bytes.Buffer),
		}
		c.Writer = blw

		// process handlers
		c.Next()

		// after handlers
		status := c.Writer.Status()
		logrus.Infof("Response Status: %d", status)

		// only log the _buffered_ body if it’s something human-readable
		if blw.body.Len() > 0 {
			logrus.Infof("Response Body:\n%s", blw.body.String())
		}
	}
}
