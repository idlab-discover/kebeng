package objectstore_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"os"
	"testing"

	"github.com/idlab-discover/kebeng/services/store/internal/config"
	"github.com/idlab-discover/kebeng/services/store/internal/objectstore"
	"github.com/minio/minio-go/v7"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestSaveFileToBucket(t *testing.T) {
	mockMinio := new(objectstore.MockObjectStore)
	cfg := &config.Config{}

	testObjectStore := &objectstore.ObjectStore{
		MinioClient: mockMinio,
		Cfg:         cfg,
	}

	mockMinio.On("BucketExists", mock.Anything, "test-bucket").Return(false, nil)
	mockMinio.On("MakeBucket", mock.Anything, "test-bucket", mock.Anything).Return(nil)
	mockMinio.On("FPutObject", mock.Anything, "test-bucket", "testfile.txt", "some/path/testfile.txt", mock.Anything).
		Return(minio.UploadInfo{Size: 456}, nil)

	size, err := testObjectStore.SaveFileToBucket("test-bucket", "some/path/testfile.txt")
	assert.NoError(t, err)
	assert.Equal(t, uint64(456), size)

	mockMinio.AssertExpectations(t)
}

// objectstore_test.go
func TestMove_Success(t *testing.T) {
	mockMinio := new(objectstore.MockObjectStore)
	cfg := &config.Config{}

	store := &objectstore.ObjectStore{
		Cfg:         cfg,
		MinioClient: mockMinio,
	}

	srcBucket := "source"
	dstBucket := "destination"
	object := "file.snap"

	// Expectations
	mockMinio.On("BucketExists", mock.Anything, srcBucket).Return(true, nil)
	mockMinio.On("BucketExists", mock.Anything, dstBucket).Return(true, nil)
	mockMinio.On("CopyObject", mock.Anything,
		mock.AnythingOfType("minio.CopyDestOptions"),
		mock.AnythingOfType("minio.CopySrcOptions"),
	).Return(minio.UploadInfo{}, nil)

	// Act
	err := store.Move(srcBucket, dstBucket, object)

	// Assert
	assert.NoError(t, err)
	mockMinio.AssertExpectations(t)
}

func TestMove_BucketDoesNotExist(t *testing.T) {
	mockMinio := new(objectstore.MockObjectStore)
	store := &objectstore.ObjectStore{MinioClient: mockMinio}

	mockMinio.On("BucketExists", mock.Anything, "source").Return(false, nil)
	mockMinio.On("BucketExists", mock.Anything, "destination").Return(true, nil)

	err := store.Move("source", "destination", "file.snap")
	assert.Error(t, err)
	assert.EqualError(t, err, "something went wrong")

	mockMinio.AssertExpectations(t)
}

func TestGetSnapFileReader_Success(t *testing.T) {
	mockMinio := new(objectstore.MockObjectStore)
	cfg := &config.Config{}

	store := &objectstore.ObjectStore{
		Cfg:         cfg,
		MinioClient: mockMinio,
	}

	filePath := "some/path/testfile.txt"
	mockObject := &minio.Object{}
	mockMinio.On("GetObject", mock.Anything, "snaps", filePath, mock.Anything).
		Return(mockObject, nil)

	reader, err := store.GetSnapFileReader(context.Background(), filePath)
	assert.NoError(t, err)
	assert.NotNil(t, reader)

	mockMinio.AssertExpectations(t)
}

func TestGetSnapFileReader_Error(t *testing.T) {
	mockMinio := new(objectstore.MockObjectStore)
	cfg := &config.Config{}

	store := &objectstore.ObjectStore{
		Cfg:         cfg,
		MinioClient: mockMinio,
	}

	filePath := "some/path/testfile.txt"
	expectedError := errors.New("failed to get object")

	mockMinio.On("GetObject", mock.Anything, "snaps", filePath, mock.Anything).
		Return(&minio.Object{}, expectedError)

	reader, err := store.GetSnapFileReader(context.Background(), filePath)
	assert.Error(t, err)
	assert.Nil(t, reader)
	assert.EqualError(t, err, "failed to get object")

	mockMinio.AssertExpectations(t)
}

func TestGetSnapFileReader_InvalidBucket(t *testing.T) {
	mockMinio := new(objectstore.MockObjectStore)
	cfg := &config.Config{}

	store := &objectstore.ObjectStore{
		Cfg:         cfg,
		MinioClient: mockMinio,
	}

	filePath := "some/path/testfile.txt"
	expectedError := errors.New("bucket does not exist")

	mockMinio.On("GetObject", mock.Anything, "snaps", filePath, mock.Anything).
		Return(&minio.Object{}, expectedError)

	reader, err := store.GetSnapFileReader(context.Background(), filePath)
	assert.Error(t, err)
	assert.Nil(t, reader)
	assert.EqualError(t, err, "bucket does not exist")

	mockMinio.AssertExpectations(t)
}

func createTempPEMFile(t *testing.T, content string) string {
	t.Helper()
	tmpFile, err := os.CreateTemp("", "testkey*.pem")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tmpFile.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}
	if err := tmpFile.Close(); err != nil {
		t.Fatal(err)
	}
	return tmpFile.Name()
}

func generateTestRSAPEM() string {
	priv, _ := rsa.GenerateKey(rand.Reader, 2048)
	privPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(priv),
	})
	return string(privPEM)
}

func TestMakeBucketAndAddKey(t *testing.T) {
	mockMinio := new(objectstore.MockObjectStore)
	cfg := &config.Config{}
	store := &objectstore.ObjectStore{MinioClient: mockMinio, Cfg: cfg}

	// Dynamisch gegenereerde geldige RSA private key
	dynamicKey := generateTestRSAPEM()
	tmpFile := createTempPEMFile(t, dynamicKey)
	defer os.Remove(tmpFile)

	// Simuleer ListObjects
	ch := make(chan minio.ObjectInfo)
	close(ch)
	mockMinio.On("ListObjects", mock.Anything, "my-bucket", mock.Anything).Return((<-chan minio.ObjectInfo)(ch))

	// Simuleer MakeBucket
	mockMinio.On("MakeBucket", mock.Anything, "my-bucket", mock.Anything).Return(nil)

	// Simuleer PutObject
	mockMinio.On("PutObject", mock.Anything, "my-bucket", "my-key.pem", mock.Anything, mock.AnythingOfType("int64"), mock.Anything).
		Return(minio.UploadInfo{Size: 123}, nil)

	// Act
	store.MakeBucketAndAddKey("my-bucket", tmpFile, "my-key.pem")

	// Assert
	mockMinio.AssertExpectations(t)
}
