package accounts

import (
	database "github.com/idlab-discover/kebeng/services/accounts/internal"
	"github.com/idlab-discover/kebeng/services/accounts/internal/config"
	"github.com/sirupsen/logrus"
)

func main() {
    cfg, err := config.LoadConfig("app/config.yaml")
    if err != nil {
        logrus.Fatalf("Failed to load configuration: %v", err)
    }

    db, err := database.NewDatabase()
    if err != nil {
        logrus.Fatalf("Failed to connect to database: %v", err)
    }
    
    logrus.Infof("Connected to database: %v", db)

    // start service

}

