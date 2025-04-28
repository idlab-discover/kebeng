package main

import (
	"net/http"
	_ "net/http/pprof"

	"github.com/idlab-discover/kebeng/common/monitoring"
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

	//TODO: check how bigger load works and multiple requests
	// need more connections? or how does grpc handle this
	accountClient, cerr := accClient.NewAccountClient(cfg.AccountServiceHost, cfg.AccountServicePort)
	if cerr != nil {
		logrus.Fatalf("could not create account client: %v", cerr)
	}

	storeClient, cerr := storeClient.NewStoreClient(cfg.StoreServiceHost, cfg.StoreServicePort)
	if cerr != nil {
		logrus.Fatalf("could not create store client: %v", cerr)
	}

	// TODO: don't give pointer back here
	assertionClient, cerr := assertionClient.NewAssertionClient(cfg.AssertionServiceHost, cfg.AssertionServicePort)
	if cerr != nil {
		logrus.Fatalf("could not create assertion client: %v", cerr)
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

	if cfg.Monitoring {
		logrus.Info("Monitoring enabled")
		// can be used to see the heap allocation
		go func() {
			logrus.Infof("Starting pprof endpoint on :6060")
			if err := http.ListenAndServe(":6060", nil); err != nil {
				logrus.Fatalf("pprof ListenAndServe: %v", err)
			}
		}()

		monitoring.CreateMetricsEndpoint()
		handler.SetupEndpointsWithMonitoring(r)

	} else {
		handler.SetupEndpoints(r)
	}

	err = r.Run()
	if err != nil {
		logrus.Fatalf("Error running server: %v", err)
	}
}
