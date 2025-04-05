package main

import (
	"fmt"
	"net"

	"github.com/idlab-discover/kebeng/services/assertion/internal/config"
	"github.com/idlab-discover/kebeng/services/assertion/internal/database"
	"github.com/idlab-discover/kebeng/services/assertion/internal/logic"
	"github.com/idlab-discover/kebeng/services/assertion/internal/repository"
	proto "github.com/idlab-discover/kebeng/services/assertion/proto"
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
	repo := repository.NewAssertionRepository(db)
	assertionLogic := logic.NewAssertionLogic(repo)

	if cfg.TestMode {
		err := database.LoadTestData(cfg.TestDataFilePath, repo)
		if err != nil {
			logrus.Fatalf("Failed to load test data: %v", err)
		}
	}

	lis, err := net.Listen("tcp", fmt.Sprintf("%s:%d", cfg.GRPCHost, cfg.GRPCPort))
	if err != nil {
		logrus.Fatalf("Failed to listen: %v", err)
	}
	grpcServer := grpc.NewServer()
	// this line will match the logic functionality to the proto interface
	proto.RegisterAssertionServiceServer(grpcServer, assertionLogic)

	logrus.Infof("Starting gRPC server on %s:%d", cfg.GRPCHost, cfg.GRPCPort)
	if err := grpcServer.Serve(lis); err != nil {
		logrus.Fatalf("Failed to serve: %v", err)
	}
}
