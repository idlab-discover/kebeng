package monitoring

import "github.com/gin-gonic/gin"

func Wrapper(name string, fn gin.HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer StartTimer(name)
		fn(c)
	}
}
