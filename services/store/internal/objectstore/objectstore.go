package objectstore

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"path"
	"strings"

	"github.com/idlab-discover/kebeng/services/store/internal/config"
	"github.com/idlab-discover/kebeng/services/store/internal/model"

	"github.com/idlab-discover/kebeng/common/cerror"
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
	RemoveObject(ctx context.Context, bucketName, objectName string, opts minio.RemoveObjectOptions) error
	StatObject(ctx context.Context, bucketName, objectName string, opts minio.GetObjectOptions) (minio.ObjectInfo, error)
}

type IObjectStore interface {
	SaveFileToBucket(bucket string, filePath string, sha3_384_encoded string, name string, version string, summary string, description string, confinement string, base string, grade string, architectures, refreshControl []string, plugs model.Plugs, slots model.Slots) (*model.Metadata, error)
	GetSnapFileReader(ctx context.Context, filePath string) (io.ReadCloser, error)
	Move(sourceBucket, destinationBucket, objectName string, newObjectName string) error
	GetObjectCustomMetadata(bucket string, objectName string) (*model.Metadata, error)
	DeleteFileFromBucket(bucket string, filePath string) *cerror.CustomError
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
func (obs *ObjectStore) GetSnapFileReader(ctx context.Context, filePath string) (io.ReadCloser, error) {
	logrus.Infof("Getting file reader for file path: %s", filePath)

	objectPtr, err := obs.MinioClient.GetObject(ctx, "snaps", filePath, minio.GetObjectOptions{})
	if err != nil {
		logrus.Errorf("error getting object from bucket 'snaps', file path: %s, err: %v", filePath, err)
		return nil, err
	}

	return objectPtr, nil
}

func (obs *ObjectStore) Move(sourceBucket, destinationBucket, objectName, newObjectName string) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	// Copy object to destination with the new name
	_, err := obs.MinioClient.CopyObject(ctx,
		minio.CopyDestOptions{
			Bucket: destinationBucket,
			Object: newObjectName,
		},
		minio.CopySrcOptions{
			Bucket: sourceBucket,
			Object: objectName,
		},
	)
	if err != nil {
		return err
	}

	// Delete the original object after successful copy
	err = obs.MinioClient.RemoveObject(ctx, sourceBucket, objectName, minio.RemoveObjectOptions{})
	if err != nil {
		return err
	}

	return nil
}

func (obs *ObjectStore) SaveFileToBucket(bucket string, filePath string, sha3_384_encoded string, name string, version string, summary string, description string, confinement string, base string, grade string, architectures, refreshControl []string, plugs model.Plugs, slots model.Slots) (*model.Metadata, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	exists, _ := obs.MinioClient.BucketExists(ctx, bucket)
	if !exists {
		err := obs.MinioClient.MakeBucket(ctx, bucket, minio.MakeBucketOptions{})
		if err != nil {
			logrus.Errorf("error creating bucket %s, err: %v", bucket, err)
			return nil, err
		}
	}

	baseFileName := path.Base(filePath)

	serializedPlugs, cerr := serializeNestedMap(plugs)
	if cerr != nil {
		// Already logged in SerializePlugMap
		return nil, errors.New(cerr.GetMessage())
	}

	serializedSlots, cerr := serializeNestedMap(slots)
	if cerr != nil {
		// Already logged in SerializePlugMap
		return nil, errors.New(cerr.GetMessage())
	}

	// Prepare user metadata
	userMetadata := map[string]string{
		"Sha3-384-Encoded": sha3_384_encoded,
		"Name":             name,
		"Version":          version,
		"Summary":          summary,
		"Description":      "description", // TODO: use discription of snapcraft.yaml but make sure description is sanitized (no new lines, no special chars, no markdown etc)
		"Confinement":      confinement,
		"Base":             base,
		"Grade":            grade,
		"Architectures":    strings.Join(architectures, ","),
		"Plugs":            serializedPlugs,
		"Slots":            serializedSlots,
		"Refresh-Control":  strings.Join(refreshControl, ","),
	}

	uploadInfo, err := obs.MinioClient.FPutObject(ctx, bucket, baseFileName, filePath, minio.PutObjectOptions{
		UserMetadata: userMetadata,
	})
	if err != nil {
		logrus.Errorf("error uploading file to bucket %s, file path: %s, err: %v", bucket, filePath, err)
		return nil, err
	}

	metadata := &model.Metadata{
		UploadInfo:       &uploadInfo,
		SHA3_384_Encoded: sha3_384_encoded,
		Name:             name,
		Version:          version,
		Summary:          summary,
		Description:      description,
		Confinement:      confinement,
		Base:             base,
		Grade:            grade,
		Architectures:    architectures,
		Plugs:            plugs,
		Slots:            slots,
		RefreshControl:   refreshControl,
	}

	return metadata, nil
}

func (obs *ObjectStore) GetObjectCustomMetadata(bucket string, objectName string) (*model.Metadata, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	objectInfo, err := obs.MinioClient.StatObject(ctx, bucket, objectName, minio.GetObjectOptions{})
	if err != nil {
		logrus.Errorf("error getting object info from bucket %s, object name: %s, err: %v", bucket, objectName, err)
		return nil, err
	}

	plugs, cerr := deserializeToPlugs(objectInfo.UserMetadata["Plugs"])
	if cerr != nil {
		// Already logged in DeserializeToNestedMap
		return nil, errors.New(cerr.GetMessage())
	}

	slots, cerr := deserializeToSlots(objectInfo.UserMetadata["Slots"])
	if cerr != nil {
		// Already logged in DeserializeToNestedMap
		return nil, errors.New(cerr.GetMessage())
	}

	metadata := &model.Metadata{
		SHA3_384_Encoded: objectInfo.UserMetadata["Sha3-384-Encoded"],
		Name:             objectInfo.UserMetadata["Name"],
		Version:          objectInfo.UserMetadata["Version"],
		Type:             objectInfo.UserMetadata["Type"],
		Summary:          objectInfo.UserMetadata["Summary"],
		Description:      objectInfo.UserMetadata["Description"],
		Confinement:      objectInfo.UserMetadata["Confinement"],
		Base:             objectInfo.UserMetadata["Base"],
		Grade:            objectInfo.UserMetadata["Grade"],
		Architectures:    strings.Split(objectInfo.UserMetadata["Architectures"], ","),
		Plugs:            plugs,
		Slots:            slots,
		RefreshControl:   strings.Split(objectInfo.UserMetadata["Refresh-Control"], ","),
	}

	return metadata, nil
}

func GetMinioClient(cfg *config.Config) *minio.Client {
	accessKey := cfg.MinioAccessKey
	secretKey := cfg.MinioSecretKey
	minioHost := cfg.MinioHost

	logrus.Debugf("Minio host=%s, accessKey=%s, secretKey=%s", minioHost, accessKey, secretKey)

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

func (obs *ObjectStore) DeleteFileFromBucket(bucket string, filePath string) *cerror.CustomError {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	exists, err := obs.MinioClient.BucketExists(ctx, bucket)
	if err != nil {
		logrus.Error(err)
		return cerror.NewCustomError(cerror.InternalServerError, err.Error())
	}

	if exists {
		err = obs.MinioClient.RemoveObject(ctx, bucket, filePath, minio.RemoveObjectOptions{})
		if err != nil {
			logrus.Error(err)
			return cerror.NewCustomError(cerror.InternalServerError, err.Error())
		}
	}

	return nil
}

func deserializeToPlugs(data string) (model.Plugs, *cerror.CustomError) {
	if data == "" {
		return nil, nil
	}
	var result model.Plugs
	err := json.Unmarshal([]byte(data), &result)
	if err != nil {
		cerr := cerror.NewCustomError(cerror.InternalServerError, fmt.Sprintf("failed to deserialize nested map: %v", err))
		logrus.Errorf(cerr.GetMessage())
		return nil, cerr
	}
	return result, nil
}

func deserializeToSlots(data string) (model.Slots, *cerror.CustomError) {
	if data == "" {
		return nil, nil
	}
	var result model.Slots
	err := json.Unmarshal([]byte(data), &result)
	if err != nil {
		cerr := cerror.NewCustomError(cerror.InternalServerError, fmt.Sprintf("failed to deserialize nested map: %v", err))
		logrus.Errorf(cerr.GetMessage())
		return nil, cerr
	}
	return result, nil
}

func serializeNestedMap[T model.Plugs | model.Slots](data T) (string, *cerror.CustomError) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		cerr := cerror.NewCustomError(cerror.InternalServerError, fmt.Sprintf("failed to serialize nested map: %v", err))
		logrus.Errorf(cerr.GetMessage())
		return "", cerr
	}
	return string(jsonData), nil
}
