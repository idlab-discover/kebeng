package main

import (
	"context"
	"fmt"
	"net"

	"github.com/idlab-discover/kebeng/services/store/internal/config"
	"github.com/idlab-discover/kebeng/services/store/internal/database"
	logic "github.com/idlab-discover/kebeng/services/store/internal/logic"
	"github.com/idlab-discover/kebeng/services/store/internal/objectstore"
	repositories "github.com/idlab-discover/kebeng/services/store/internal/repository"
	"github.com/idlab-discover/kebeng/services/store/monitoring"
	proto "github.com/idlab-discover/kebeng/services/store/proto"
	"github.com/minio/minio-go/v7"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

func main() {
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

	exists, err := minioClient.BucketExists(context.Background(), "root")
	if err != nil {
		panic(err)
	}
	if exists {
		logrus.Warnf("Bucket exists, please use destroy command if you are sure you want to start over.")
	}

	exists, err = minioClient.BucketExists(context.Background(), "generic")
	if err != nil {
		panic(err)
	}

	if exists {
		logrus.Warnf("Bucket exists, please use destroy command if you are sure you want to start over.")
	}

	if !exists {
		objectstore.MakeBucketAndAddKey("root", cfg.RootKeyPath, "private-key.pem")
		objectstore.MakeBucketAndAddKey("generic", cfg.GenericKeyPath, "private-key.pem")
		err := minioClient.MakeBucket(context.Background(), "snaps", minio.MakeBucketOptions{})
		if err != nil {
			logrus.Errorf("Failed to create bucket: %v", err)
		}
	}
	// start grpc server
	repo := repositories.NewSnapsRepository(db)
	storeLogic := logic.NewStoreLogic(repo, cfg, objectstore)

	if cfg.TestMode {
		logrus.Infof("Running in test mode")
	}

	if cfg.Monitoring {
		logrus.Infof("Starting metrics endpoint on port 9100")
		monitoring.CreateMetricsEndpoint()
	}

	lis, err := net.Listen("tcp", fmt.Sprintf("%s:%d", cfg.GRPCHost, cfg.GRPCPort))
	if err != nil {
		logrus.Fatalf("Failed to listen: %v", err)
	}
	grpcServer := grpc.NewServer()
	// this line will match the service functionality to the proto interface
	proto.RegisterStoreServiceServer(grpcServer, storeLogic)

	logrus.Infof("Starting gRPC server on %s:%d", cfg.GRPCHost, cfg.GRPCPort)
	if err := grpcServer.Serve(lis); err != nil {
		logrus.Fatalf("Failed to serve: %v", err)
	}

}
