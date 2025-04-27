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

	"store/internal/config"
	"store/internal/models"
	"store/internal/objectstore"

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

	metadata, err := testObjectStore.SaveFileToBucket("test-bucket", "some/path/testfile.txt", "sha3_384_hash")
	assert.NoError(t, err)
	assert.NotNil(t, metadata)
	assert.Equal(t, int64(456), metadata.Size)

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
	newObjectName := "newfile.snap"

	// Expectations
	mockMinio.On("BucketExists", mock.Anything, srcBucket).Return(true, nil)
	mockMinio.On("BucketExists", mock.Anything, dstBucket).Return(true, nil)
	mockMinio.On("CopyObject", mock.Anything,
		mock.AnythingOfType("minio.CopyDestOptions"),
		mock.AnythingOfType("minio.CopySrcOptions"),
	).Return(minio.UploadInfo{}, nil)
	mockMinio.On("RemoveObject", mock.Anything, srcBucket, object, mock.Anything).Return(nil)

	// Act
	err := store.Move(srcBucket, dstBucket, object, newObjectName)

	// Assert
	assert.NoError(t, err)
	mockMinio.AssertExpectations(t)
}

func TestMove_BucketDoesNotExist(t *testing.T) {
	mockMinio := new(objectstore.MockObjectStore)
	store := &objectstore.ObjectStore{MinioClient: mockMinio}

	mockMinio.On("BucketExists", mock.Anything, "source").Return(false, nil)
	mockMinio.On("BucketExists", mock.Anything, "destination").Return(true, nil)

	err := store.Move("source", "destination", "file.snap", "newfile.snap")
	assert.Error(t, err)
	assert.EqualError(t, err, "source or destination bucket does not exist")

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

func TestGetObjectCustomMetadata(t *testing.T) {
	mockMinio := new(objectstore.MockObjectStore)
	cfg := &config.Config{}
	store := &objectstore.ObjectStore{MinioClient: mockMinio, Cfg: cfg}

	bucket := "test-bucket"
	object := "test-object"

	mockObjectInfo := minio.ObjectInfo{
		UserMetadata: map[string]string{
			"Sha3-384-encoded": "",
		},
	}

	sha3_384_encoded := mockObjectInfo.UserMetadata["Sha3-384-encoded"]

	expectedMetadata := &models.Metadata{
		SHA3_384_Encoded: sha3_384_encoded,
	}

	mockMinio.On("StatObject", mock.Anything, bucket, object, mock.Anything).
		Return(mockObjectInfo, nil)

	metadata, err := store.GetObjectCustomMetadata(bucket, object)
	assert.NoError(t, err)
	assert.NotNil(t, metadata)
	assert.Equal(t, expectedMetadata, metadata)

	mockMinio.AssertExpectations(t)
}

func TestGetObjectMetadata_Error(t *testing.T) {
	mockMinio := new(objectstore.MockObjectStore)
	cfg := &config.Config{}
	store := &objectstore.ObjectStore{MinioClient: mockMinio, Cfg: cfg}

	bucket := "test-bucket"
	object := "non-existent-object"

	expectedError := errors.New("object not found")

	mockMinio.On("StatObject", mock.Anything, bucket, object, mock.Anything).
		Return(minio.ObjectInfo{}, expectedError)

	metadata, err := store.GetObjectCustomMetadata(bucket, object)
	assert.Error(t, err)
	assert.Nil(t, metadata)
	assert.EqualError(t, err, "object not found")

	mockMinio.AssertExpectations(t)
}

func TestDeleteFileFromBucket(t *testing.T) {
	mockMinio := new(objectstore.MockObjectStore)
	cfg := &config.Config{}

	store := &objectstore.ObjectStore{
		Cfg:         cfg,
		MinioClient: mockMinio,
	}

	bucket := "test-bucket"
	object := "test-object"

	mockMinio.On("BucketExists", mock.Anything, bucket).Return(true, nil)
	mockMinio.On("RemoveObject", mock.Anything, bucket, object, mock.Anything).Return(nil)

	cerr := store.DeleteFileFromBucket(bucket, object)
	assert.Nil(t, cerr)

	mockMinio.AssertExpectations(t)
}
func TestDeleteFileFromBucket_Error(t *testing.T) {
	mockMinio := new(objectstore.MockObjectStore)
	cfg := &config.Config{}

	store := &objectstore.ObjectStore{
		Cfg:         cfg,
		MinioClient: mockMinio,
	}

	bucket := "test-bucket"
	object := "non-existent-object"

	expectedError := errors.New("object not found")

	mockMinio.On("BucketExists", mock.Anything, bucket).Return(true, nil)
	mockMinio.On("RemoveObject", mock.Anything, bucket, object, mock.Anything).Return(expectedError)

	cerr := store.DeleteFileFromBucket(bucket, object)
	assert.NotNil(t, cerr)
	assert.Equal(t, cerr.GetMessage(), "object not found")

	mockMinio.AssertExpectations(t)
}

func TestDeleteFileFromBucket_BucketDoesNotExist(t *testing.T) {
	mockMinio := new(objectstore.MockObjectStore)
	cfg := &config.Config{}

	store := &objectstore.ObjectStore{
		Cfg:         cfg,
		MinioClient: mockMinio,
	}

	bucket := "non-existent-bucket"
	object := "test-object"

	expectedError := errors.New("bucket does not exist")

	mockMinio.On("BucketExists", mock.Anything, bucket).Return(false, expectedError)

	cerr := store.DeleteFileFromBucket(bucket, object)
	assert.NotNil(t, cerr)
	assert.Equal(t, cerr.GetMessage(), "bucket does not exist")

	mockMinio.AssertExpectations(t)
}
