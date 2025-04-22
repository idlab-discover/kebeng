package main

import (
	"monitoring/internal"
	"monitoring/internal/config"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		logrus.Fatalf("Error loading config: %v", err)
	}

	logrus.Infof("Loaded config: %+v", cfg)

	r := gin.Default()
	internal.SetupEndpoints(r)

	r.NoRoute(func(c *gin.Context) {
		c.JSON(404, gin.H{"message": "endpoint does not exist"})
	})

	err = r.Run()
	if err != nil {
		logrus.Fatalf("Failed to start server: %v", err)
	}
}
