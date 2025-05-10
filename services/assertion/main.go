package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/idlab-discover/kebeng/services/assertion/internal/config"
	"github.com/idlab-discover/kebeng/services/assertion/internal/database"
	"github.com/idlab-discover/kebeng/services/assertion/internal/logic"
	"github.com/idlab-discover/kebeng/services/assertion/internal/repository"
	proto "github.com/idlab-discover/kebeng/services/assertion/proto"

	"github.com/idlab-discover/kebeng/common/monitoring"
	"github.com/sirupsen/logrus"
	"github.com/snapcore/snapd/asserts"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func main() {
	ctx := context.Background()
	cfg, err := config.LoadConfig()
	if err != nil {
		logrus.Fatalf("Failed to load configuration: %v", err)
	}
	logrus.Infof("Loaded configuration: %+v", cfg)

	// Connect to MongoDB
	mongoDB, err := database.NewMongoDBConnection(cfg)
	if err != nil {
		logrus.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	logrus.Infof("Connected to MongoDB")

	// Assertion DB for root key management
	assertionDB, err := asserts.OpenDatabase(&asserts.DatabaseConfig{
		KeypairManager: asserts.NewMemoryKeypairManager(),
	})
	if err != nil {
		logrus.Fatalf("Failed to open assertion database: %s", err)
	}

	if err := assertionDB.ImportKey(cfg.RootKey); err != nil {
		logrus.Fatalf("Failed to import root key: %s in assertions db", err)
	}

	// Repositories for PostgreSQL and MongoDB
	repo := repository.NewAssertionRepository(mongoDB)

	// Business Logic
	assertionLogic := logic.NewAssertionLogic(cfg, repo, assertionDB)

	if cfg.TestMode {
		logrus.Infof("Running in test mode")
	}

	// Create account key assertion
	now := time.Now()
	serializedPub, err := asserts.EncodePublicKey(cfg.RootKey.PublicKey())
	if err != nil {
		logrus.Fatalf("Failed to serialize public key: %v", err)
	}

	b64pub := base64.StdEncoding.EncodeToString(serializedPub)

	req := &proto.AddAccountKeyAssertionRequest{
		EncodedPublicKey:         b64pub,
		PublicKeySha3_384Encoded: cfg.RootKey.PublicKey().ID(),
		AccountId:                cfg.RootAccountID.String(),
		Name:                     "kebeng",
		Since:                    &timestamppb.Timestamp{Seconds: now.Unix()},
		Until:                    &timestamppb.Timestamp{Seconds: now.Add(24 * time.Hour).Unix()},
	}
	accountKeyAssertion, _ := assertionLogic.AddAccountKeyAssertion(ctx, req)
	if len(accountKeyAssertion.Errors) != 0 {
		logrus.Fatalf("Failed to create account key assertion: %v", accountKeyAssertion.Errors)
	}

	// Create root account assertion
	req2 := &proto.AddAccountAssertionRequest{
		AccountId:   cfg.RootAccountID.String(),
		DisplayName: "kebeng",
		Username:    "kebeng",
		Validation:  "certified",
		Timestamp:   &timestamppb.Timestamp{Seconds: now.Unix()},
	}
	accountAssertion, _ := assertionLogic.AddAccountAssertion(ctx, req2)
	if len(accountAssertion.Errors) != 0 {
		logrus.Fatalf("Failed to create account assertion: %v", accountAssertion.Errors)
	}

	// Monitoring and metrics setup
	if cfg.Monitoring {
		logrus.Infof("Creating metrics endpoint")
		go func() {
			logrus.Infof("Starting pprof endpoint on :6060")
			if err := http.ListenAndServe(":6060", nil); err != nil {
				logrus.Fatalf("pprof ListenAndServe: %v", err)
			}
		}()
		monitoring.CreateMetricsEndpoint()
	}

	// Start gRPC server
	lis, err := net.Listen("tcp", fmt.Sprintf("%s:%d", cfg.GRPCHost, cfg.GRPCPort))
	if err != nil {
		logrus.Fatalf("Failed to listen: %v", err)
	}
	grpcServer := grpc.NewServer()
	// Register health check service
	hs := health.NewServer()
	hs.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)

	// Register services
	healthpb.RegisterHealthServer(grpcServer, hs)
	proto.RegisterAssertionServiceServer(grpcServer, assertionLogic)

	logrus.Infof("Starting gRPC server on %s:%d", cfg.GRPCHost, cfg.GRPCPort)
	if err := grpcServer.Serve(lis); err != nil {
		logrus.Fatalf("Failed to serve: %v", err)
	}
}
