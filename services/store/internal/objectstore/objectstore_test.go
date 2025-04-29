package objectstore_test

import (
	"context"
	"errors"
	"testing"

	"store/internal/config"
	"store/internal/models"
	"store/internal/objectstore"

	"github.com/minio/minio-go/v7"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestSaveFileToBucket_succes(t *testing.T) {
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

	metadata, err := testObjectStore.SaveFileToBucket(
		"test-bucket",
		"some/path/testfile.txt",
		"sha3_384_hash",
		"test-snap",
		"1.0.0",
		"Test Summary",
		"Test Description",
		"strict",
		"core18",
		"test-grade",
		[]string{"amd64", "arm64"},
	)
	assert.NoError(t, err)
	assert.NotNil(t, metadata)
	assert.Equal(t, int64(456), metadata.Size)
	assert.Equal(t, "test-snap", metadata.Name)
	assert.Equal(t, "1.0.0", metadata.Version)
	assert.Equal(t, "Test Summary", metadata.Summary)
	assert.Equal(t, "Test Description", metadata.Description)
	assert.Equal(t, "strict", metadata.Confinement)
	assert.Equal(t, "core18", metadata.Base)
	assert.Equal(t, "test-grade", metadata.Grade)
	assert.ElementsMatch(t, []string{"amd64", "arm64"}, metadata.Architectures)

	mockMinio.AssertExpectations(t)
}

func TestSaveFileToBucket_ErrorCreatingBucket(t *testing.T) {
	mockMinio := new(objectstore.MockObjectStore)
	cfg := &config.Config{}

	testObjectStore := &objectstore.ObjectStore{
		MinioClient: mockMinio,
		Cfg:         cfg,
	}

	mockMinio.On("BucketExists", mock.Anything, "test-bucket").Return(false, nil)
	mockMinio.On("MakeBucket", mock.Anything, "test-bucket", mock.Anything).Return(errors.New("bucket creation error"))

	metadata, err := testObjectStore.SaveFileToBucket(
		"test-bucket",
		"some/path/testfile.txt",
		"sha3_384_hash",
		"test-snap",
		"1.0.0",
		"Test Summary",
		"Test Description",
		"strict",
		"core18",
		"test-grade",
		[]string{"amd64", "arm64"},
	)
	assert.Error(t, err)
	assert.Nil(t, metadata)

	mockMinio.AssertExpectations(t)
}

func TestSaveFileToBucket_ErrorUploadingFile(t *testing.T) {
	mockMinio := new(objectstore.MockObjectStore)
	cfg := &config.Config{}

	testObjectStore := &objectstore.ObjectStore{
		MinioClient: mockMinio,
		Cfg:         cfg,
	}

	mockMinio.On("BucketExists", mock.Anything, "test-bucket").Return(true, nil)
	mockMinio.On("FPutObject", mock.Anything, "test-bucket", "testfile.txt", "some/path/testfile.txt", mock.Anything).
		Return(minio.UploadInfo{}, errors.New("upload error"))

	metadata, err := testObjectStore.SaveFileToBucket(
		"test-bucket",
		"some/path/testfile.txt",
		"sha3_384_hash",
		"test-snap",
		"1.0.0",
		"Test Summary",
		"Test Description",
		"strict",
		"core18",
		"test-grade",
		[]string{"amd64", "arm64"},
	)
	assert.Error(t, err)
	assert.Nil(t, metadata)

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

	mockMinio.On("CopyObject", mock.Anything,
		mock.AnythingOfType("minio.CopyDestOptions"),
		mock.AnythingOfType("minio.CopySrcOptions"),
	).Return(minio.UploadInfo{}, errors.New("source or destination bucket does not exist"))

	err := store.Move("source", "destination", "file.snap", "newfile.snap")
	assert.Error(t, err)
	assert.EqualError(t, err, "source or destination bucket does not exist")

	mockMinio.AssertExpectations(t)
}

func TestMove_RemoveObjectError(t *testing.T) {
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
	mockMinio.On("CopyObject", mock.Anything,
		mock.AnythingOfType("minio.CopyDestOptions"),
		mock.AnythingOfType("minio.CopySrcOptions"),
	).Return(minio.UploadInfo{}, nil)
	mockMinio.On("RemoveObject", mock.Anything, srcBucket, object, mock.Anything).Return(errors.New("failed to remove object"))

	err := store.Move(srcBucket, dstBucket, object, newObjectName)
	assert.Error(t, err)
	assert.EqualError(t, err, "failed to remove object")

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

func TestGetObjectCustomMetadata(t *testing.T) {
	mockMinio := new(objectstore.MockObjectStore)
	cfg := &config.Config{}
	store := &objectstore.ObjectStore{MinioClient: mockMinio, Cfg: cfg}

	bucket := "test-bucket"
	object := "test-object"

	mockObjectInfo := minio.ObjectInfo{
		UserMetadata: map[string]string{
			"Sha3-384-encoded": "test-sha3-384",
			"Name":             "test-name",
			"Version":          "1.0.0",
			"Type":             "app",
			"Summary":          "Test Summary",
			"Description":      "Test Description",
			"Confinement":      "strict",
			"Base":             "core18",
			"Architectures":    "amd64,arm64",
		},
	}

	sha3_384_encoded := mockObjectInfo.UserMetadata["Sha3-384-Encoded"]
	name := mockObjectInfo.UserMetadata["Name"]
	version := mockObjectInfo.UserMetadata["Version"]
	fileType := mockObjectInfo.UserMetadata["Type"]
	summary := mockObjectInfo.UserMetadata["Summary"]
	description := mockObjectInfo.UserMetadata["Description"]
	confinement := mockObjectInfo.UserMetadata["Confinement"]
	base := mockObjectInfo.UserMetadata["Base"]
	architectures := []string{"amd64", "arm64"}

	expectedMetadata := &models.Metadata{
		SHA3_384_Encoded: sha3_384_encoded,
		Name:             name,
		Version:          version,
		Type:             fileType,
		Summary:          summary,
		Description:      description,
		Confinement:      confinement,
		Base:             base,
		Architectures:    architectures,
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

func TestGetMinioClient(t *testing.T) {
	cfg := &config.Config{
		MinioHost:      "minio:9000",
		MinioAccessKey: "minioadmin",
		MinioSecretKey: "minioadmin",
	}

	client := objectstore.GetMinioClient(cfg)
	assert.NotNil(t, client)

	// Verify the endpoint URL and scheme
	endpoint := client.EndpointURL()
	assert.Equal(t, "http://minio:9000", endpoint.String())
	assert.Equal(t, "http", endpoint.Scheme)
}
