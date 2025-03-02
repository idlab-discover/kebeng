package main

import (
    "net"
    "fmt"

	database "github.com/idlab-discover/kebeng/services/account/internal"
	"github.com/idlab-discover/kebeng/services/account/internal/config"
	"github.com/idlab-discover/kebeng/services/account/internal/repository"
	"github.com/idlab-discover/kebeng/services/account/internal/logic"
	proto "github.com/idlab-discover/kebeng/services/account/proto"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

func main() {
    cfg, err := config.LoadConfig()
    if err != nil {
        logrus.Fatalf("Failed to load configuration: %v", err)
    }
    logrus.Infof("Loaded configuration: %+v", cfg)

    db, err := database.NewDatabase(cfg)
    if err != nil {
        logrus.Fatalf("Failed to connect to database: %v", err)
    }
    
    logrus.Infof("Connected to database: %v", db)


    // start grpc server
    repo := repository.NewAccountRepository(db)
    accountService := logic.NewAccountService(repo, cfg)
    
    lis, err := net.Listen("tcp", fmt.Sprintf("%s:%d",cfg.GRPCHost,cfg.GRPCPort))
    if err != nil {
        logrus.Fatalf("Failed to listen: %v", err)
    }
    grpcServer := grpc.NewServer()
    // this line will match the logic functionality to the proto interface
    proto.RegisterAccountServiceServer(grpcServer, accountService)

    logrus.Infof("Starting gRPC server on %s:%d", cfg.GRPCHost, cfg.GRPCPort)
    if err := grpcServer.Serve(lis); err != nil {
        logrus.Fatalf("Failed to serve: %v", err)
    }
}

