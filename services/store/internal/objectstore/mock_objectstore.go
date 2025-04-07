package objectstore

import (
	"io"

	"github.com/idlab-discover/kebeng/services/store/internal/repository"
	"github.com/minio/minio-go/v7"
	"github.com/stretchr/testify/mock"
)

type MockObjectStore struct {
	mock.Mock
}

var _ IObjectStore = (*MockObjectStore)(nil)

func (m *MockObjectStore) GetSnapFileReader(filePath string) (io.ReadCloser, error) {
	args := m.Called(filePath)
	return args.Get(0).(io.ReadCloser), args.Error(1)
}

// SaveFileToBucket implements ObjectStore.
func (m *MockObjectStore) SaveFileToBucket(bucket string, filePath string) (uint64, error) {
	args := m.Called(bucket, filePath)
	return args.Get(0).(uint64), args.Error(1)
}

// LoadTestData implements ObjectStore.
func (m *MockObjectStore) LoadTestData(client *minio.Client, repo repository.ISnapsRepository, minioPath string) error {
	args := m.Called(client, repo, minioPath)
	return args.Error(0)
}
