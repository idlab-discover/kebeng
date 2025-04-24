package main

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/google/uuid"
	"github.com/idlab-discover/kebeng/services/account/internal/config"
	"github.com/idlab-discover/kebeng/services/account/internal/database"
	"github.com/idlab-discover/kebeng/services/account/internal/logic"
	"github.com/idlab-discover/kebeng/services/account/internal/models"
	"github.com/idlab-discover/kebeng/services/account/internal/repository"
	"github.com/idlab-discover/kebeng/services/account/monitoring"
	proto "github.com/idlab-discover/kebeng/services/account/proto"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

func main() {
	ctx := context.Background()
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

	// if in test mode, load test data
	if cfg.TestMode {
		logrus.Infof("Test mode enabled")
	}

	// create root account
	parsedRootAccountID, err := uuid.Parse(cfg.RootAccountID)
	if err != nil {
		logrus.Fatalf("Failed to parse root account ID: %v", err)
	}

	validation := "certified"
	// idk what these values should be maybe best in config but hardcoded for now
	rootAccount := &models.Account{
		ID:           parsedRootAccountID,
		DisplayName:  "kebeng",
		Username:     "kebeng",
		Email:        "kebeng@gmail.com",
		PasswordHash: "password",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		DeletedAt:    nil,
		Validation:   validation,
	}

	_, cerr := repo.AddAccount(ctx, rootAccount)
	if cerr != nil {
		logrus.Fatalf("failed to create root account: %v", cerr)
	}

	// create metrics endpoint
	if cfg.Monitoring {
		logrus.Infof("Monitoring enabled")
		monitoring.CreateMetricsEndpoint()
	}

	lis, err := net.Listen("tcp", fmt.Sprintf("%s:%d", cfg.GRPCHost, cfg.GRPCPort))
	if err != nil {
		logrus.Fatalf("Failed to listen: %v", err)
	}
	grpcServer := grpc.NewServer()
	// register health check service
	hs := health.NewServer()
	hs.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)

	// register services
	healthpb.RegisterHealthServer(grpcServer, hs)
	proto.RegisterAccountServiceServer(grpcServer, accountService)

	logrus.Infof("Starting gRPC server on %s:%d", cfg.GRPCHost, cfg.GRPCPort)
	if err := grpcServer.Serve(lis); err != nil {
		logrus.Fatalf("Failed to serve: %v", err)
	}
}
