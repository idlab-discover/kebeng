package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"strings"

	"github.com/idlab-discover/kebeng/pkg/crypto"
	"github.com/idlab-discover/kebeng/services/store/internal/config"
	"github.com/idlab-discover/kebeng/services/store/internal/database"
	logic "github.com/idlab-discover/kebeng/services/store/internal/logic"
	"github.com/idlab-discover/kebeng/services/store/internal/objectstore"
	repositories "github.com/idlab-discover/kebeng/services/store/internal/repository"
	proto "github.com/idlab-discover/kebeng/services/store/proto"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
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

	minioClient := getMinioClient(cfg)

	exists, err := minioClient.BucketExists(context.Background(), "root")
	if err != nil {
		panic(err)
	}
	if exists {
		fmt.Printf("Bucket '%s' exists, please use destroy command if you are sure you want to start over.", "root")
		return
	}

	exists, err = minioClient.BucketExists(context.Background(), "generic")
	if err != nil {
		panic(err)
	}

	if exists {
		fmt.Printf("Bucket '%s' exists, please use destroy command if you are sure you want to start over.", "generic")
		return
	}

	makeBucketAndAddKey(minioClient, "root", cfg.RootKeyPath, "private-key.pem")
	makeBucketAndAddKey(minioClient, "generic", cfg.GenericKeyPath, "private-key.pem")
	minioClient.MakeBucket(context.Background(), "snaps", minio.MakeBucketOptions{})

	objectstore := objectstore.NewObjectStore(minioClient)
	// start grpc server
	repo := repositories.NewSnapsRepository(db)
	storeLogic := logic.NewStoreLogic(repo, cfg, objectstore)

	if cfg.TestMode {
		logrus.Infof("Running in test mode, using test data file: %s", cfg.TestDataFilePath)
		err := database.LoadTestData(cfg.TestDataFilePath, repo)
		if err != nil {
			logrus.Fatalf("Failed to load test data: %v", err)
		}

		logrus.Infof("Loaded test data, uploading to minio: %s", cfg.TestDataMinioPath)
		err = objectstore.LoadTestData(minioClient, repo, cfg.TestDataMinioPath)
		if err != nil {
			logrus.Fatalf("Failed to load test data to minio: %v", err)
		}
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

func makeBucketAndAddKey(minioClient *minio.Client, bucketName string, keyPath string, keyName string) {
	// Make root bucket
	fmt.Printf("*************************************\nCreating bucket: %s\n, keyPath: %s\n *************************", bucketName, keyPath)
	ctx, cancel := context.WithCancel(context.Background())

	defer cancel()

	objectCh := minioClient.ListObjects(ctx, bucketName, minio.ListObjectsOptions{
		Recursive: true,
	})
	for object := range objectCh {
		logrus.Tracef("object: %s", object.Key)
	}

	err := minioClient.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{})
	if err != nil {
		logrus.Error(err)
	}

	bytes, err := os.ReadFile(keyPath)
	if err != nil {
		logrus.Error(err)
	}
	rootPrivateKey, err := crypto.ParseRSAPrivateKeyFromPEM(bytes)
	if err != nil {
		logrus.Error(err)
	}
	keyString := crypto.ExportRsaPrivateKeyAsPemStr(rootPrivateKey)

	_, err = minioClient.PutObject(ctx, bucketName, keyName, strings.NewReader(keyString), int64(len(keyString)), minio.PutObjectOptions{})
	if err != nil {
		panic(err)
	}
}

func getMinioClient(cfg *config.Config) *minio.Client {
	accessKey := cfg.MinioAccessKey
	secretKey := cfg.MinioSecretKey
	minioHost := cfg.MinioHost
	minioSecure := cfg.MinioSecure

	logrus.Infof("Minio host=%s, accessKey=%s, secretKey=%s", minioHost, accessKey, secretKey)

	// Initialize minio client object.
	minioClient, err := minio.New(minioHost, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: minioSecure,
	})
	if err != nil {
		logrus.Errorf("Failed to initialize new minio client: %+v", err)
		return nil
	}

	return minioClient
}

func deleteAllItemsInBucket(minioClient *minio.Client, bucketName string) {
	objectsCh := make(chan minio.ObjectInfo)

	// Send object names that are needed to be removed to objectsCh
	go func() {
		defer close(objectsCh)
		// List all objects from a bucket-name with a matching prefix.
		for object := range minioClient.ListObjects(context.Background(), bucketName, minio.ListObjectsOptions{}) {
			if object.Err != nil {
				log.Fatalln(object.Err)
			}
			objectsCh <- object
		}
	}()

	opts := minio.RemoveObjectsOptions{
		GovernanceBypass: true,
	}

	for rErr := range minioClient.RemoveObjects(context.Background(), bucketName, objectsCh, opts) {
		fmt.Println("Error detected during deletion: ", rErr)
	}
}
