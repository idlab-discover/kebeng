// internal/objectstore/mock_minio_client.go
package objectstore

import (
	"context"
	"io"

	"github.com/minio/minio-go/v7"
	"github.com/stretchr/testify/mock"
)

var _ IObjectStore = (*MockObjectStore)(nil)

type MockObjectStore struct {
	mock.Mock
}

// GetSnapFileReader implements IObjectStore.
func (m *MockObjectStore) GetSnapFileReader(ctx context.Context, filePath string) (io.ReadCloser, error) {
	args := m.Called(filePath)
	if args.Get(0) != nil {
		return args.Get(0).(io.ReadCloser), nil
	}
	return nil, args.Get(1).(error)
}

// SaveFileToBucket implements IObjectStore.
func (m *MockObjectStore) SaveFileToBucket(bucket string, filePath string) (*minio.UploadInfo, error) {
	args := m.Called(bucket, filePath)
	if args.Get(0) != nil {
		return args.Get(0).(*minio.UploadInfo), nil
	}
	return &minio.UploadInfo{}, args.Get(1).(error)
}

func (m *MockObjectStore) GetObject(ctx context.Context, bucket, object string, opts minio.GetObjectOptions) (*minio.Object, error) {
	args := m.Called(ctx, bucket, object, opts)
	return args.Get(0).(*minio.Object), args.Error(1)
}

func (m *MockObjectStore) BucketExists(ctx context.Context, bucket string) (bool, error) {
	args := m.Called(ctx, bucket)
	return args.Bool(0), args.Error(1)
}

func (m *MockObjectStore) MakeBucket(ctx context.Context, bucket string, opts minio.MakeBucketOptions) error {
	args := m.Called(ctx, bucket, opts)
	return args.Error(0)
}

func (m *MockObjectStore) MakeBucketAndAddKey(bucketName string, keyPath string, keyName string) {
	m.Called(bucketName, keyPath, keyName)
}

func (m *MockObjectStore) Move(sourceBucket, destinationBucket, objectName string) error {
	args := m.Called(sourceBucket, destinationBucket, objectName)
	return args.Error(0)
}

func (m *MockObjectStore) FPutObject(ctx context.Context, bucket, object, filePath string, opts minio.PutObjectOptions) (minio.UploadInfo, error) {
	args := m.Called(ctx, bucket, object, filePath, opts)
	return args.Get(0).(minio.UploadInfo), args.Error(1)
}

func (m *MockObjectStore) PutObject(ctx context.Context, bucket, object string, reader io.Reader, objectSize int64, opts minio.PutObjectOptions) (minio.UploadInfo, error) {
	args := m.Called(ctx, bucket, object, reader, objectSize, opts)
	return args.Get(0).(minio.UploadInfo), args.Error(1)
}

func (m *MockObjectStore) CopyObject(ctx context.Context, dst minio.CopyDestOptions, src minio.CopySrcOptions) (minio.UploadInfo, error) {
	args := m.Called(ctx, dst, src)
	return args.Get(0).(minio.UploadInfo), args.Error(1)
}

func (m *MockObjectStore) ListObjects(ctx context.Context, bucket string, opts minio.ListObjectsOptions) <-chan minio.ObjectInfo {
	args := m.Called(ctx, bucket, opts)
	return args.Get(0).(<-chan minio.ObjectInfo)
}
