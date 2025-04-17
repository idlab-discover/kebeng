package repository

import (
	"github.com/google/uuid"
	cerror "github.com/idlab-discover/kebeng/common/cerror"
	"github.com/idlab-discover/kebeng/services/store/internal/models"
	"github.com/stretchr/testify/mock"
)

// MockSnapsRepository is a mock implementation of ISnapsRepository for testing.
type MockSnapsRepository struct {
	mock.Mock
}

var _ ISnapsRepository = (*MockSnapsRepository)(nil)

// CREATE
func (m *MockSnapsRepository) AddChannel(snapEntryId uuid.UUID, snapTrackId uuid.UUID, channelName string, el *cerror.ErrorList) (*models.SnapChannel, *cerror.CustomError) {
	args := m.Called(snapEntryId, snapTrackId, channelName)
	if args.Get(0) != nil {
		return args.Get(0).(*models.SnapChannel), nil
	}
	return nil, args.Get(1).(*cerror.CustomError)
}

func (m *MockSnapsRepository) AddDefaultChannels(snapEntryId uuid.UUID, snapTrackId uuid.UUID, el *cerror.ErrorList) *cerror.CustomError {
	args := m.Called(snapEntryId, snapTrackId)
	if args.Get(0) != nil {
		return args.Get(0).(*cerror.CustomError)
	}
	return nil
}

func (m *MockSnapsRepository) AddRevision(entryId uuid.UUID, trackId uuid.UUID, channelId uuid.UUID, snapName string, size uint64, sequenceNumber uint, architectures []string, sha3_384 string, minioFilePath string, el *cerror.ErrorList) (*models.SnapRevision, *cerror.CustomError) {
	args := m.Called(entryId, trackId, channelId, snapName, size, sequenceNumber, architectures, sha3_384, minioFilePath)
	if args.Get(0) != nil {
		return args.Get(0).(*models.SnapRevision), nil
	}
	return nil, args.Get(1).(*cerror.CustomError)
}

func (m *MockSnapsRepository) AddTrack(entryId uuid.UUID, trackName string, el *cerror.ErrorList) (*models.SnapTrack, *cerror.CustomError) {
	args := m.Called(entryId, trackName, el)
	if args.Get(0) != nil {
		return args.Get(0).(*models.SnapTrack), nil
	}
	return nil, args.Get(1).(*cerror.CustomError)
}

func (m *MockSnapsRepository) AddUpload(snapName string, entryId uuid.UUID, status string, accountId uuid.UUID, unscannedFileName string) (*models.SnapUpload, *cerror.CustomError) {
	args := m.Called(snapName, entryId, status, accountId, unscannedFileName)
	if args.Get(0) != nil {
		return args.Get(0).(*models.SnapUpload), nil
	}
	return nil, args.Get(1).(*cerror.CustomError)
}

func (m *MockSnapsRepository) RegisterSnap(snapName string, isPrivate bool, storeName string, accountId uuid.UUID) (*models.SnapEntry, *cerror.CustomError) {
	args := m.Called(snapName, isPrivate, storeName, accountId)
	if args.Get(0) != nil {
		return args.Get(0).(*models.SnapEntry), nil
	}
	return nil, args.Get(1).(*cerror.CustomError)
}

// READ
func (m *MockSnapsRepository) GetAllSnapEntries() (*[]models.SnapEntry, *cerror.CustomError) {
	args := m.Called()
	if args.Get(0) != nil {
		return args.Get(0).(*[]models.SnapEntry), nil
	}
	return nil, args.Get(1).(*cerror.CustomError)
}

func (m *MockSnapsRepository) GetChannelsByTrackId(trackId uuid.UUID) ([]*models.SnapChannel, *cerror.CustomError) {
	args := m.Called(trackId)
	if args.Get(0) != nil {
		return args.Get(0).([]*models.SnapChannel), nil
	}
	return nil, args.Get(1).(*cerror.CustomError)
}

func (m *MockSnapsRepository) GetCommentsByEntryId(entryId uuid.UUID) ([]*models.SnapComment, *cerror.CustomError) {
	args := m.Called(entryId)
	if args.Get(0) != nil {
		return args.Get(0).([]*models.SnapComment), nil
	}
	return nil, args.Get(1).(*cerror.CustomError)
}

func (m *MockSnapsRepository) GetEntriesByAccountId(accountId uuid.UUID, preloadAssociations []string, el *cerror.ErrorList) ([]*models.SnapEntry, *cerror.CustomError) {
	args := m.Called(accountId, preloadAssociations)
	if args.Get(0) != nil {
		return args.Get(0).([]*models.SnapEntry), nil
	}
	return nil, args.Get(1).(*cerror.CustomError)
}

func (m *MockSnapsRepository) GetEntryById(id uuid.UUID, preloadAssociations []string, el *cerror.ErrorList) (*models.SnapEntry, *cerror.CustomError) {
	args := m.Called(id, preloadAssociations)
	if args.Get(0) != nil {
		return args.Get(0).(*models.SnapEntry), nil
	}
	return nil, args.Get(1).(*cerror.CustomError)
}

func (m *MockSnapsRepository) GetEntryByName(name string, preloadAssociations []string, el *cerror.ErrorList) (*models.SnapEntry, *cerror.CustomError) {
	// if preloadAssociations == nil {
	// 	preloadAssociations = []string{}
	// }
	args := m.Called(name, preloadAssociations)
	if args.Get(0) != nil {
		return args.Get(0).(*models.SnapEntry), nil
	}
	return nil, args.Get(1).(*cerror.CustomError)
}

func (m *MockSnapsRepository) GetRevisionsByEntryId(entryId uuid.UUID) ([]*models.SnapRevision, *cerror.CustomError) {
	args := m.Called(entryId)
	if args.Get(0) != nil {
		return args.Get(0).([]*models.SnapRevision), nil
	}
	return nil, args.Get(1).(*cerror.CustomError)
}

func (m *MockSnapsRepository) GetRevisionById(id uuid.UUID) (*models.SnapRevision, *cerror.CustomError) {
	args := m.Called(id)
	if args.Get(0) != nil {
		return args.Get(0).(*models.SnapRevision), nil
	}
	return nil, args.Get(1).(*cerror.CustomError)
}

func (m *MockSnapsRepository) GetRevisionByNameAndSequence(name string, sequence uint) (*models.SnapRevision, *cerror.CustomError) {
	args := m.Called(name, sequence)
	if args.Get(0) != nil {
		return args.Get(0).(*models.SnapRevision), nil
	}
	return nil, args.Get(1).(*cerror.CustomError)
}

func (m *MockSnapsRepository) GetRevisionBySHA(SHA3_384 string, encoded bool, el *cerror.ErrorList) (*models.SnapRevision, *cerror.CustomError) {
	args := m.Called(SHA3_384, encoded)
	if args.Get(0) != nil {
		return args.Get(0).(*models.SnapRevision), nil
	}
	return nil, args.Get(1).(*cerror.CustomError)
}

func (m *MockSnapsRepository) GetSections() (*[]string, *cerror.CustomError) {
	args := m.Called()
	if args.Get(0) != nil {
		return args.Get(0).(*[]string), nil
	}
	return nil, args.Get(1).(*cerror.CustomError)
}

func (m *MockSnapsRepository) GetTracksByEntryId(snapId uuid.UUID) ([]*models.SnapTrack, *cerror.CustomError) {
	args := m.Called(snapId)
	if args.Get(0) != nil {
		return args.Get(0).([]*models.SnapTrack), nil
	}
	return nil, args.Get(1).(*cerror.CustomError)
}

// UPDATE
func (m *MockSnapsRepository) ReleaseSnap(channels []string, snapEntryId uuid.UUID, revisionId uuid.UUID) *cerror.CustomError {
	args := m.Called(channels, snapEntryId, revisionId)
	return args.Get(0).(*cerror.CustomError)
}

func (m *MockSnapsRepository) UpdateRevision(revision *models.SnapRevision, revisionBytes *[]byte) (*models.SnapRevision, *cerror.CustomError) {
	args := m.Called(revision, revisionBytes)
	if args.Get(0) != nil {
		return args.Get(0).(*models.SnapRevision), nil
	}
	return nil, args.Get(1).(*cerror.CustomError)
}

func (m *MockSnapsRepository) UpdateUploadStatus(uploadId uuid.UUID, status string, revision uint64, el *cerror.ErrorList) *cerror.CustomError {
	args := m.Called(uploadId, status, revision, el)
	if args.Get(0) != nil {
		return args.Get(0).(*cerror.CustomError)
	}
	return nil
}

func (m *MockSnapsRepository) GetLatestRevisionByTrackAndChannel(snapName string, track string, channel string) (*models.SnapRevision, *cerror.CustomError) {
	args := m.Called(snapName, track, channel)
	if args.Get(0) != nil {
		return args.Get(0).(*models.SnapRevision), nil
	}
	return nil, args.Get(1).(*cerror.CustomError)
}

func (m *MockSnapsRepository) GetChannelById(id uuid.UUID) (*models.SnapChannel, *cerror.CustomError) {
	args := m.Called(id)
	if args.Get(0) != nil {
		return args.Get(0).(*models.SnapChannel), nil
	}
	return nil, args.Get(1).(*cerror.CustomError)
}

func (m *MockSnapsRepository) GetTrackById(id uuid.UUID) (*models.SnapTrack, *cerror.CustomError) {
	args := m.Called(id)
	if args.Get(0) != nil {
		return args.Get(0).(*models.SnapTrack), nil
	}
	return nil, args.Get(1).(*cerror.CustomError)
}

func (m *MockSnapsRepository) GetPreloadAssociations(entry *models.SnapEntry, preloadAssociations *[]string, el *cerror.ErrorList) *cerror.CustomError {
	args := m.Called(entry, preloadAssociations)
	if args.Get(0) != nil {
		return nil
	}
	return args.Get(1).(*cerror.CustomError)
}

func (m *MockSnapsRepository) GetUploadById(id uuid.UUID, el *cerror.ErrorList) (*models.SnapUpload, *cerror.CustomError) {
	args := m.Called(id, el)
	if args.Get(0) != nil {
		return args.Get(0).(*models.SnapUpload), nil
	}
	return nil, args.Get(1).(*cerror.CustomError)
}

func (m *MockSnapsRepository) GetLatestRevisionByEntryId(entryId uuid.UUID, el *cerror.ErrorList) (*models.SnapRevision, *cerror.CustomError) {
	args := m.Called(entryId)
	if args.Get(0) != nil {
		return args.Get(0).(*models.SnapRevision), nil
	}
	return nil, args.Get(1).(*cerror.CustomError)
}

func (m *MockSnapsRepository) GetTrackByEntryIdAndName(entryId uuid.UUID, trackName string, el *cerror.ErrorList) (*models.SnapTrack, *cerror.CustomError) {
	args := m.Called(entryId, trackName)
	if args.Get(0) != nil {
		return args.Get(0).(*models.SnapTrack), nil
	}
	return nil, args.Get(1).(*cerror.CustomError)
}

func (m *MockSnapsRepository) GetChannelByTrackIdAndName(trackId uuid.UUID, channelName string, el *cerror.ErrorList) (*models.SnapChannel, *cerror.CustomError) {
	args := m.Called(trackId, channelName)
	if args.Get(0) != nil {
		return args.Get(0).(*models.SnapChannel), nil
	}
	return nil, args.Get(1).(*cerror.CustomError)
}
