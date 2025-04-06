package middleware

import (
	"bytes"
	"fmt"
	"io"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// bodyLogWriter wraps gin.ResponseWriter to capture the response body.
type bodyLogWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

// Write captures the response body and writes it to the original ResponseWriter.
func (w *bodyLogWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

func RequestLoggerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {

		if c.ContentType() == "multipart/form-data" {
			logrus.Infof("Skipping body for multipart/form-data")

		} else {
			var buf bytes.Buffer
			tee := io.TeeReader(c.Request.Body, &buf)
			body, _ := io.ReadAll(tee)
			c.Request.Body = io.NopCloser(&buf)
			fmt.Println("Body:")

			fmt.Println(string(body))
		}

		fmt.Println("Header:")
		fmt.Println(c.Request.Header)

		blw := &bodyLogWriter{
			ResponseWriter: c.Writer,
			body:           bytes.NewBufferString(""),
		}
		c.Writer = blw

		// Execute the request and capture the response.
		c.Next()

		// After endpoint execution, log the captured response body.
		fmt.Printf("Response body: %s", blw.body.String())
	}
}
