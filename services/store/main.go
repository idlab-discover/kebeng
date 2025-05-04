package main

import (
	"context"
	"fmt"
	"net"

	"net/http"
	_ "net/http/pprof"

	"github.com/idlab-discover/kebeng/services/store/internal/config"
	"github.com/idlab-discover/kebeng/services/store/internal/database"
	logic "github.com/idlab-discover/kebeng/services/store/internal/logic"
	"github.com/idlab-discover/kebeng/services/store/internal/objectstore"
	repositories "github.com/idlab-discover/kebeng/services/store/internal/repository"
	proto "github.com/idlab-discover/kebeng/services/store/proto"

	"github.com/idlab-discover/kebeng/common/monitoring"
	"github.com/minio/minio-go/v7"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	// set logrus to use text formatter
	logrus.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04:05",
		DisableColors:   true,
	})

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

	minioClient := objectstore.GetMinioClient(cfg)
	objectstore := objectstore.NewObjectStore(minioClient, cfg)

	err = minioClient.MakeBucket(ctx, "snaps", minio.MakeBucketOptions{})
	if err != nil {
		logrus.Fatalf("Failed to create snapsbucket: %v", err)
	}
	err = minioClient.MakeBucket(ctx, "unscanned", minio.MakeBucketOptions{})
	if err != nil {
		logrus.Fatalf("Failed to create unscanned bucket: %v", err)
	}

	// start grpc server
	repo := repositories.NewSnapsRepository(db)
	storeLogic := logic.NewStoreLogic(repo, cfg, objectstore)

	if cfg.TestMode {
		logrus.Infof("Running in test mode")
	}

	lis, err := net.Listen("tcp", fmt.Sprintf("%s:%d", cfg.GRPCHost, cfg.GRPCPort))
	if err != nil {
		logrus.Fatalf("Failed to listen: %v", err)
	}
	var grpcServer *grpc.Server
	if cfg.Monitoring {
		logrus.Infof("Starting metrics endpoint on port 9100")
		monitoring.CreateMetricsEndpoint()
		grpcServer = grpc.NewServer(
			grpc.StreamInterceptor(monitoring.StreamingInterceptor),
		)
	} else {
		grpcServer = grpc.NewServer()

	}
	// register health check service
	hs := health.NewServer()
	hs.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)

	// register services
	healthpb.RegisterHealthServer(grpcServer, hs)
	// this line will match the service functionality to the proto interface
	proto.RegisterStoreServiceServer(grpcServer, storeLogic)

	// also start pprof on :6060
	go func() {
		logrus.Infof("Starting pprof endpoint on :6060")
		if err := http.ListenAndServe(":6060", nil); err != nil {
			logrus.Fatalf("pprof ListenAndServe: %v", err)
		}
	}()

	logrus.Infof("Starting gRPC server on %s:%d", cfg.GRPCHost, cfg.GRPCPort)
	if err := grpcServer.Serve(lis); err != nil {
		logrus.Fatalf("Failed to serve: %v", err)
	}

}
