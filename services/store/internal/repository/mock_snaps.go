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
	el.Add(cerror.InternalServerError, "")
	return nil, args.Get(1).(*cerror.CustomError)
}

func (m *MockSnapsRepository) AddDefaultChannels(snapEntryId uuid.UUID, snapTrackId uuid.UUID, el *cerror.ErrorList) *cerror.CustomError {
	args := m.Called(snapEntryId, snapTrackId)
	if args.Get(0) != nil {
		return args.Get(0).(*cerror.CustomError)
	}
	el.Add(cerror.InternalServerError, "")
	return nil
}

func (m *MockSnapsRepository) AddRevision(entryId uuid.UUID, trackId uuid.UUID, channelId uuid.UUID, snapName string, size uint64, sequenceNumber uint32, architectures []string, sha3_384 string, minioFilePath string, el *cerror.ErrorList) (*models.SnapRevision, *cerror.CustomError) {
	args := m.Called(entryId, trackId, channelId, snapName, size, sequenceNumber, architectures, sha3_384, minioFilePath, el)
	if args.Get(0) != nil {
		return args.Get(0).(*models.SnapRevision), nil
	}
	el.Add(cerror.InternalServerError, "")
	return nil, args.Get(1).(*cerror.CustomError)
}

func (m *MockSnapsRepository) AddTrack(entryId uuid.UUID, trackName string, el *cerror.ErrorList) (*models.SnapTrack, *cerror.CustomError) {
	args := m.Called(entryId, trackName, el)
	if args.Get(0) != nil {
		return args.Get(0).(*models.SnapTrack), nil
	}
	el.Add(cerror.InternalServerError, "")
	return nil, args.Get(1).(*cerror.CustomError)
}

func (m *MockSnapsRepository) AddUpload(entryId uuid.UUID, accountId uuid.UUID, snapName, status, unscannedFileName string, revision uint32, el *cerror.ErrorList) (*models.SnapUpload, *cerror.CustomError) {
	args := m.Called(entryId, accountId, snapName, status, unscannedFileName, revision, el)
	if args.Get(0) != nil {
		return args.Get(0).(*models.SnapUpload), nil
	}
	el.Add(cerror.InternalServerError, "")
	return nil, args.Get(1).(*cerror.CustomError)
}

func (m *MockSnapsRepository) RegisterSnap(snapName string, snapType string, confinement string, base string, isPrivate bool, status string, price float64, storeName string, iconURL string, accountId uuid.UUID, el *cerror.ErrorList) (*models.SnapEntry, *cerror.CustomError) {
	args := m.Called(snapName, snapType, confinement, base, isPrivate, status, price, storeName, iconURL, accountId, el)
	if args.Get(0) != nil {
		return args.Get(0).(*models.SnapEntry), nil
	}
	el.Add(cerror.InternalServerError, "")
	return nil, args.Get(1).(*cerror.CustomError)
}

// READ
func (m *MockSnapsRepository) GetAllSnapEntries(el *cerror.ErrorList) (*[]models.SnapEntry, *cerror.CustomError) {
	args := m.Called(el)
	if args.Get(0) != nil {
		return args.Get(0).(*[]models.SnapEntry), nil
	}
	el.Add(cerror.InternalServerError, "")
	return nil, args.Get(1).(*cerror.CustomError)
}

func (m *MockSnapsRepository) GetChannelsByTrackId(trackId uuid.UUID, el *cerror.ErrorList) ([]*models.SnapChannel, *cerror.CustomError) {
	args := m.Called(trackId, el)
	if args.Get(0) != nil {
		return args.Get(0).([]*models.SnapChannel), nil
	}
	el.Add(cerror.InternalServerError, "")
	return nil, args.Get(1).(*cerror.CustomError)
}

func (m *MockSnapsRepository) GetCommentsByEntryId(entryId uuid.UUID, el *cerror.ErrorList) ([]*models.SnapComment, *cerror.CustomError) {
	args := m.Called(entryId, el)
	if args.Get(0) != nil {
		return args.Get(0).([]*models.SnapComment), nil
	}
	el.Add(cerror.InternalServerError, "")
	return nil, args.Get(1).(*cerror.CustomError)
}

func (m *MockSnapsRepository) GetEntriesByAccountId(accountId uuid.UUID, preloadAssociations []string, el *cerror.ErrorList) ([]*models.SnapEntry, *cerror.CustomError) {
	args := m.Called(accountId, preloadAssociations, el)
	if args.Get(0) != nil {
		return args.Get(0).([]*models.SnapEntry), nil
	}
	el.Add(cerror.InternalServerError, "")
	return nil, args.Get(1).(*cerror.CustomError)
}

func (m *MockSnapsRepository) GetEntryById(id uuid.UUID, preloadAssociations []string, el *cerror.ErrorList) (*models.SnapEntry, *cerror.CustomError) {
	args := m.Called(id, preloadAssociations, el)
	if args.Get(0) != nil {
		return args.Get(0).(*models.SnapEntry), nil
	}
	el.Add(cerror.InternalServerError, "")
	return nil, args.Get(1).(*cerror.CustomError)
}

func (m *MockSnapsRepository) GetEntryByName(name string, preloadAssociations []string, el *cerror.ErrorList) (*models.SnapEntry, *cerror.CustomError) {
	args := m.Called(name, preloadAssociations, el)
	if args.Get(0) != nil {
		return args.Get(0).(*models.SnapEntry), nil
	}
	el.Add(cerror.InternalServerError, "")
	return nil, args.Get(1).(*cerror.CustomError)
}

func (m *MockSnapsRepository) GetRevisionsByEntryId(entryId uuid.UUID, el *cerror.ErrorList) ([]*models.SnapRevision, *cerror.CustomError) {
	args := m.Called(entryId, el)
	if args.Get(0) != nil {
		return args.Get(0).([]*models.SnapRevision), nil
	}
	el.Add(cerror.InternalServerError, "")
	return nil, args.Get(1).(*cerror.CustomError)
}

func (m *MockSnapsRepository) GetRevisionById(id uuid.UUID, el *cerror.ErrorList) (*models.SnapRevision, *cerror.CustomError) {
	args := m.Called(id, el)
	if args.Get(0) != nil {
		return args.Get(0).(*models.SnapRevision), nil
	}
	el.Add(cerror.InternalServerError, "")
	return nil, args.Get(1).(*cerror.CustomError)
}

func (m *MockSnapsRepository) GetRevisionByNameAndSequence(name string, sequence uint32, el *cerror.ErrorList) (*models.SnapRevision, *cerror.CustomError) {
	args := m.Called(name, sequence, el)
	if args.Get(0) != nil {
		return args.Get(0).(*models.SnapRevision), nil
	}
	el.Add(cerror.InternalServerError, "")
	return nil, args.Get(1).(*cerror.CustomError)
}

func (m *MockSnapsRepository) GetRevisionBySHA(SHA3_384 string, el *cerror.ErrorList) (*models.SnapRevision, *cerror.CustomError) {
	args := m.Called(SHA3_384, el)
	if args.Get(0) != nil {
		return args.Get(0).(*models.SnapRevision), nil
	}
	el.Add(cerror.InternalServerError, "")
	return nil, args.Get(1).(*cerror.CustomError)
}

func (m *MockSnapsRepository) GetTracksByEntryId(snapId uuid.UUID, el *cerror.ErrorList) ([]*models.SnapTrack, *cerror.CustomError) {
	args := m.Called(snapId, el)
	if args.Get(0) != nil {
		return args.Get(0).([]*models.SnapTrack), nil
	}
	el.Add(cerror.InternalServerError, "")
	return nil, args.Get(1).(*cerror.CustomError)
}

func (m *MockSnapsRepository) GetLatestRevisionByTrackAndChannel(snapName string, track string, channel string, el *cerror.ErrorList) (*models.SnapRevision, *cerror.CustomError) {
	args := m.Called(snapName, track, channel, el)
	if args.Get(0) != nil {
		return args.Get(0).(*models.SnapRevision), nil
	}
	el.Add(cerror.InternalServerError, "")
	return nil, args.Get(1).(*cerror.CustomError)
}

func (m *MockSnapsRepository) GetChannelById(id uuid.UUID, el *cerror.ErrorList) (*models.SnapChannel, *cerror.CustomError) {
	args := m.Called(id, el)
	if args.Get(0) != nil {
		return args.Get(0).(*models.SnapChannel), nil
	}
	el.Add(cerror.InternalServerError, "")
	return nil, args.Get(1).(*cerror.CustomError)
}

func (m *MockSnapsRepository) GetTrackById(id uuid.UUID, el *cerror.ErrorList) (*models.SnapTrack, *cerror.CustomError) {
	args := m.Called(id, el)
	if args.Get(0) != nil {
		return args.Get(0).(*models.SnapTrack), nil
	}
	el.Add(cerror.InternalServerError, "")
	return nil, args.Get(1).(*cerror.CustomError)
}

func (m *MockSnapsRepository) GetPreloadAssociations(entry *models.SnapEntry, preloadAssociations *[]string, el *cerror.ErrorList) *cerror.CustomError {
	args := m.Called(entry, preloadAssociations, el)
	if args.Get(0) != nil {
		return nil
	}
	el.Add(cerror.InternalServerError, "")
	return args.Get(1).(*cerror.CustomError)
}

func (m *MockSnapsRepository) GetUploadById(id uuid.UUID, el *cerror.ErrorList) (*models.SnapUpload, *cerror.CustomError) {
	args := m.Called(id, el)
	if args.Get(0) != nil {
		return args.Get(0).(*models.SnapUpload), nil
	}
	el.Add(cerror.InternalServerError, "")
	return nil, args.Get(1).(*cerror.CustomError)
}

func (m *MockSnapsRepository) GetLatestRevisionByEntryId(entryId uuid.UUID, el *cerror.ErrorList) (*models.SnapRevision, *cerror.CustomError) {
	args := m.Called(entryId, el)
	if args.Get(0) != nil {
		return args.Get(0).(*models.SnapRevision), nil
	}
	el.Add(cerror.InternalServerError, "")
	return nil, args.Get(1).(*cerror.CustomError)
}

func (m *MockSnapsRepository) GetTrackByEntryIdAndName(entryId uuid.UUID, trackName string, el *cerror.ErrorList) (*models.SnapTrack, *cerror.CustomError) {
	args := m.Called(entryId, trackName, el)
	if args.Get(0) != nil {
		return args.Get(0).(*models.SnapTrack), nil
	}
	el.Add(cerror.InternalServerError, "")
	return nil, args.Get(1).(*cerror.CustomError)
}

func (m *MockSnapsRepository) GetChannelByTrackIdAndName(trackId uuid.UUID, channelName string, el *cerror.ErrorList) (*models.SnapChannel, *cerror.CustomError) {
	args := m.Called(trackId, channelName, el)
	if args.Get(0) != nil {
		return args.Get(0).(*models.SnapChannel), nil
	}
	el.Add(cerror.InternalServerError, "")
	return nil, args.Get(1).(*cerror.CustomError)
}

// UPDATE

func (m *MockSnapsRepository) UpdateUploadStatus(uploadId uuid.UUID, status string, revision uint32, el *cerror.ErrorList) *cerror.CustomError {
	args := m.Called(uploadId, status, revision, el)
	if args.Get(0) != nil {
		return args.Get(0).(*cerror.CustomError)
	}
	el.Add(cerror.InternalServerError, "")
	return nil
}
