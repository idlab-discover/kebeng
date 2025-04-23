package main

import (
	"github.com/idlab-discover/kebeng/services/gateway/internal/config"
	"github.com/idlab-discover/kebeng/services/gateway/internal/util"
	loadtestdata "github.com/idlab-discover/kebeng/services/gateway/load_test_data"
	"github.com/sirupsen/logrus"

	"github.com/gin-gonic/gin"
	accClient "github.com/idlab-discover/kebeng/services/account/client"
	assertionClient "github.com/idlab-discover/kebeng/services/assertion/client"
	"github.com/idlab-discover/kebeng/services/gateway/handler"
	"github.com/idlab-discover/kebeng/services/gateway/internal/middleware"
	storeClient "github.com/idlab-discover/kebeng/services/store/client"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		logrus.Fatalf("Error loading config: %v", err)
	}

	logrus.Infof("Loaded config: %+v", cfg)

	// TODO: if can't connect to service or create client TRY AGAIN

	//TODO: check how bigger load works and multiple requests
	// need more connections? or how does grpc handle this
	accountClient, err := accClient.NewAccountClient(cfg.AccountServiceHost, cfg.AccountServicePort)
	if err != nil {
		logrus.Fatalf("could not create account client: %v", err)
	}

	storeClient, err := storeClient.NewStoreClient(cfg.StoreServiceHost, cfg.StoreServicePort)
	if err != nil {
		logrus.Fatalf("could not create store client: %v", err)
	}

	// TODO: don't give pointer back here
	assertionClient, err := assertionClient.NewAssertionClient(cfg.AssertionServiceHost, cfg.AssertionServicePort)
	if err != nil {
		logrus.Fatalf("could not create assertion client: %v", err)
	}

	handler := handler.NewHandler(accountClient, storeClient, assertionClient, cfg)

	if cfg.TestMode {
		baseHandler := &util.BaseHandler{
			AccountClient:   accountClient,
			StoreClient:     storeClient,
			AssertionClient: assertionClient,
			Config:          cfg,
		}
		err := loadtestdata.CreateTestData(baseHandler)
		if err != nil {
			logrus.Fatalf("could not load test data: %v", err)
		}
	}

	// Setup gin and routes
	r := gin.Default()
	if cfg.DebugMode {
		logrus.Info("Debug mode enabled")
		logrus.SetLevel(logrus.DebugLevel)
		r.Use(middleware.RequestLoggerMiddleware())
	} else {
		logrus.Info("Debug mode disabled")
	}

	r.NoRoute(func(c *gin.Context) {
		c.JSON(404, gin.H{"code": "KEBENG STORE: PAGE_NOT_FOUND", "message": "Page not found"})
	})

	handler.SetupEndpoints(r)

	err = r.Run()
	if err != nil {
		logrus.Fatalf("Error running server: %v", err)
	}
}
