package main

import (

    "github.com/idlab-discover/kebeng/services/gateway/internal/config"
    "github.com/sirupsen/logrus"

    accClient "github.com/idlab-discover/kebeng/services/account/client"
    storeClient "github.com/idlab-discover/kebeng/services/store/client"
    "github.com/idlab-discover/kebeng/services/gateway/handler"
    "github.com/idlab-discover/kebeng/services/gateway/internal/middleware"
	"github.com/gin-gonic/gin"

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
        logrus.Errorf("could not create account client: %v", err)
    }
        
 
    storeClient, err := storeClient.NewStoreClient()
    if err != nil {
        logrus.Errorf("could not create store client: %v", err)
    }
    
    handler := handler.NewHandler(accountClient, storeClient)
    
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


