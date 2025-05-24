package main

import (
	"net/http"

	"github.com/idlab-discover/kebeng/services/monitoring/handler"
	"github.com/idlab-discover/kebeng/services/monitoring/internal/config"
	"github.com/idlab-discover/kebeng/services/monitoring/internal/logic"

	"github.com/idlab-discover/kebeng/common/monitoring"

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
	client := &http.Client{}
	l := logic.NewLogic(cfg, client)

	h := handler.NewHandler(l)

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
