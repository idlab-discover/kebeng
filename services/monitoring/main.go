package main

import (
	"github.com/gin-gonic/gin"
	"github.com/idlab-discover/kebeng/services/monitoring/internal"
	"github.com/sirupsen/logrus"
)

func main() {
	r := gin.Default()
	internal.SetupEndpoints(r)

	r.NoRoute(func(c *gin.Context) {
		c.JSON(404, gin.H{"message": "endpoint does not exist"})
	})

	err := r.Run()
	if err != nil {
		logrus.Fatalf("Failed to start server: %v", err)
	}
}
