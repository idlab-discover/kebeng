package main

import (
	"monitoring/handler"
	"monitoring/internal/config"
	"monitoring/monitoring"

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
	h := handler.NewHandler(cfg)

	h.SetupEndpoints(r)

	r.NoRoute(func(c *gin.Context) {
		c.JSON(404, gin.H{"message": "endpoint does not exist"})
	})

	monitoring.CreateMetricsEndpoint()

	logrus.Infof("Starting monitoring service")

	err = r.Run()
	if err != nil {
		logrus.Fatalf("Failed to start server: %v", err)
	}
}
