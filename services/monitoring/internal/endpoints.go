package internal

import "github.com/gin-gonic/gin"

func SetupEndpoints(r *gin.Engine) {
	r.GET("/register-name", RegisterName)
}
