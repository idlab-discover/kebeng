package monitoring

import "github.com/gin-gonic/gin"

func Wrapper(name string, fn gin.HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		requestCount.WithLabelValues(name).Inc()

		defer StartTimer(name)()
		fn(c)
	}
}
