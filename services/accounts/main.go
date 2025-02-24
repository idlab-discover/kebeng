package accounts

import (
    "net"
    "fmt"

	database "github.com/idlab-discover/kebeng/services/accounts/internal"
	"github.com/idlab-discover/kebeng/services/accounts/internal/config"
	"github.com/idlab-discover/kebeng/services/accounts/internal/repository"
	"github.com/idlab-discover/kebeng/services/accounts/internal/service"
	proto "github.com/idlab-discover/kebeng/services/accounts/proto"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
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
    repo := repository.NewAccountRepository(db)
    accountService := service.NewAccountService(repo)
    
    lis, err := net.Listen("tcp", fmt.Sprintf("%s:%d",cfg.GRPCHost,cfg.GRPCPort))
    if err != nil {
        logrus.Fatalf("Failed to listen: %v", err)
    }
    grpcServer := grpc.NewServer()
    proto.RegisterAccountServiceServer(grpcServer, accountService)

    logrus.Infof("Starting gRPC server on %s:%d", cfg.GRPCHost, cfg.GRPCPort)
    if err := grpcServer.Serve(lis); err != nil {
        logrus.Fatalf("Failed to serve: %v", err)
    }
}

