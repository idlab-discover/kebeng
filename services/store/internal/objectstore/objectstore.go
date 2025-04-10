package objectstore

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"strings"

	"github.com/idlab-discover/kebeng/common/crypto"
	"github.com/idlab-discover/kebeng/services/store/internal/config"
	"github.com/idlab-discover/kebeng/services/store/internal/repository"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/sirupsen/logrus"
)

type IMinioClient interface {
	GetObject(ctx context.Context, bucketName, objectName string, opts minio.GetObjectOptions) (*minio.Object, error)
	BucketExists(ctx context.Context, bucketName string) (bool, error)
	MakeBucket(ctx context.Context, bucketName string, opts minio.MakeBucketOptions) error
	FPutObject(ctx context.Context, bucketName, objectName, filePath string, opts minio.PutObjectOptions) (minio.UploadInfo, error)
	PutObject(ctx context.Context, bucketName, objectName string, reader io.Reader, objectSize int64, opts minio.PutObjectOptions) (minio.UploadInfo, error)
	CopyObject(ctx context.Context, dst minio.CopyDestOptions, src minio.CopySrcOptions) (minio.UploadInfo, error)
	ListObjects(ctx context.Context, bucketName string, opts minio.ListObjectsOptions) <-chan minio.ObjectInfo
}

type IObjectStore interface {
	SaveFileToBucket(bucket string, filePath string) (uint64, error)
	GetSnapFileReader(filePath string) (io.ReadCloser, error)
	LoadTestData(client *minio.Client, repo repository.ISnapsRepository, minioPath string) error
	MakeBucketAndAddKey(bucketName string, keyPath string, keyName string)
	Move(sourceBucket, destinationBucket, objectName string) error
}

type ObjectStore struct {
	Cfg         *config.Config
	MinioClient IMinioClient
}

func NewObjectStore(minio *minio.Client, cfg *config.Config) IObjectStore {
	if minio == nil {
		return &ObjectStore{Cfg: cfg, MinioClient: GetMinioClient(cfg)}
	}
	return &ObjectStore{MinioClient: minio}
}

// don't forget to close the reader after use
func (obs *ObjectStore) GetSnapFileReader(filePath string) (io.ReadCloser, error) {
	logrus.Infof("Getting file reader for file path: %s", filePath)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	objectPtr, err := obs.MinioClient.GetObject(ctx, "snaps", filePath, minio.GetObjectOptions{})
	if err != nil {
		cancel()
		logrus.Errorf("error getting object from bucket 'snaps', file path: %s, err: %v", filePath, err)
		return nil, err
	}
	if objectPtr == nil {
		cancel()
		logrus.Errorf("error getting object from bucket 'snaps', file path: %s, err: %v", filePath, err)
		return nil, errors.New("object not found")
	}

	return objectPtr, nil
}

func (obs *ObjectStore) Move(sourceBucket, destinationBucket, objectName string) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sourceBucketExists, err := obs.MinioClient.BucketExists(ctx, sourceBucket)
	if err != nil {
		return err
	}
	destinationBucketExists, err := obs.MinioClient.BucketExists(ctx, destinationBucket)
	if err != nil {
		return err
	}

	if sourceBucketExists && destinationBucketExists {
		_, err := obs.MinioClient.CopyObject(ctx, minio.CopyDestOptions{Bucket: destinationBucket, Object: objectName}, minio.CopySrcOptions{Bucket: sourceBucket, Object: objectName})
		if err != nil {
			return err
		}

		return nil
	}

	return errors.New("something went wrong")
}

func (obs *ObjectStore) SaveFileToBucket(bucket string, filePath string) (uint64, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	exists, _ := obs.MinioClient.BucketExists(ctx, bucket)
	if !exists {
		err := obs.MinioClient.MakeBucket(ctx, bucket, minio.MakeBucketOptions{})
		if err != nil {
			logrus.Error(err)
		}
	}

	base := path.Base(filePath)

	uploadInfo, err := obs.MinioClient.FPutObject(ctx, bucket, base, filePath, minio.PutObjectOptions{})
	if err != nil {
		return 0, err
	}

	logrus.Infof("Saved to bucket: %+v", uploadInfo)

	return uint64(uploadInfo.Size), nil
}

func GetMinioClient(cfg *config.Config) *minio.Client {
	accessKey := cfg.MinioAccessKey
	secretKey := cfg.MinioSecretKey
	minioHost := cfg.MinioHost

	logrus.Infof("Minio host=%s, accessKey=%s, secretKey=%s", minioHost, accessKey, secretKey)

	// Initialize minio client object.
	minioClient, err := minio.New(minioHost, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: false,
	})
	if err != nil {
		log.Fatalln(err)
		return nil
	}

	return minioClient
}

func (obs *ObjectStore) MakeBucketAndAddKey(bucketName string, keyPath string, keyName string) {
	// Make root bucket
	fmt.Printf("*************************************\nCreating bucket: %s\n, keyPath: %s\n *************************", bucketName, keyPath)
	ctx, cancel := context.WithCancel(context.Background())

	defer cancel()

	objectCh := obs.MinioClient.ListObjects(ctx, bucketName, minio.ListObjectsOptions{
		Recursive: true,
	})
	for object := range objectCh {
		logrus.Tracef("object: %s", object.Key)
	}

	err := obs.MinioClient.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{})
	if err != nil {
		logrus.Error(err)
	}

	bytes, err := os.ReadFile(keyPath)
	if err != nil {
		logrus.Error(err)
	}
	rootPrivateKey, cerr := crypto.ParseRSAPrivateKeyFromPEM(bytes)
	if cerr != nil {
		logrus.Error(cerr)
	}
	keyString := crypto.ExportRsaPrivateKeyAsPemStr(rootPrivateKey)

	_, err = obs.MinioClient.PutObject(ctx, bucketName, keyName, strings.NewReader(keyString), int64(len(keyString)), minio.PutObjectOptions{})
	if err != nil {
		panic(err)
	}
}
