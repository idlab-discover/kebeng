package repository

import (
	"github.com/google/uuid"
	cerror "github.com/idlab-discover/kebeng/common/cerror"
	"github.com/idlab-discover/kebeng/services/store/internal/models"
	"github.com/stretchr/testify/mock"
)

// MockStoreRepository is a mock implementation of ISnapsRepository for testing.
type MockStoreRepository struct {
	mock.Mock
}

var _ ISnapsRepository = (*MockStoreRepository)(nil)

// CREATE
func (m *MockStoreRepository) AddChannel(snapEntryId uuid.UUID, snapTrackId uuid.UUID, channelName string) (*models.SnapChannel, *cerror.CustomError) {
	args := m.Called(snapEntryId, snapTrackId, channelName)
	if args.Get(0) != nil {
		return args.Get(0).(*models.SnapChannel), nil
	}
	return nil, args.Get(1).(*cerror.CustomError)
}

func (m *MockStoreRepository) AddDefaultChannels(snapEntryId uuid.UUID, snapTrackId uuid.UUID) *cerror.CustomError {
	args := m.Called(snapEntryId, snapTrackId)
	return args.Get(0).(*cerror.CustomError)
}

func (m *MockStoreRepository) AddRevision(entryId uuid.UUID, trackId uuid.UUID, channelId uuid.UUID, size uint64, sequenceNumber uint) (*models.SnapRevision, *cerror.CustomError) {
	args := m.Called(entryId, trackId, channelId, size, sequenceNumber)
	if args.Get(0) != nil {
		return args.Get(0).(*models.SnapRevision), nil
	}
	return nil, args.Get(1).(*cerror.CustomError)
}

func (m *MockStoreRepository) AddTrack(entryId uuid.UUID, trackName string) (*models.SnapTrack, *cerror.CustomError) {
	args := m.Called(entryId, trackName)
	if args.Get(0) != nil {
		return args.Get(0).(*models.SnapTrack), nil
	}
	return nil, args.Get(1).(*cerror.CustomError)
}

func (m *MockStoreRepository) RegisterSnap(snapName string, isPrivate bool) (*models.SnapEntry, *cerror.CustomError) {
	args := m.Called(snapName, isPrivate)
	if args.Get(0) != nil {
		return args.Get(0).(*models.SnapEntry), nil
	}
	return nil, args.Get(1).(*cerror.CustomError)
}

// READ
func (m *MockStoreRepository) GetAllSnapEntries() (*[]models.SnapEntry, *cerror.CustomError) {
	args := m.Called()
	if args.Get(0) != nil {
		return args.Get(0).(*[]models.SnapEntry), nil
	}
	return nil, args.Get(1).(*cerror.CustomError)
}

func (m *MockStoreRepository) GetChannelsByTrackId(trackId uuid.UUID) ([]*models.SnapChannel, *cerror.CustomError) {
	args := m.Called(trackId)
	if args.Get(0) != nil {
		return args.Get(0).([]*models.SnapChannel), nil
	}
	return nil, args.Get(1).(*cerror.CustomError)
}

func (m *MockStoreRepository) GetCommentsByEntryId(entryId uuid.UUID) ([]*models.SnapComment, *cerror.CustomError) {
	args := m.Called(entryId)
	if args.Get(0) != nil {
		return args.Get(0).([]*models.SnapComment), nil
	}
	return nil, args.Get(1).(*cerror.CustomError)
}

func (m *MockStoreRepository) GetEntriesByAccountId(accountId uuid.UUID, preloadAssociations []string) ([]*models.SnapEntry, *cerror.CustomError) {
	args := m.Called(accountId, preloadAssociations)
	if args.Get(0) != nil {
		return args.Get(0).([]*models.SnapEntry), nil
	}
	return nil, args.Get(1).(*cerror.CustomError)
}

func (m *MockStoreRepository) GetEntryById(id uuid.UUID, preloadAssociations []string) (*models.SnapEntry, *cerror.CustomError) {
	args := m.Called(id, preloadAssociations)
	if args.Get(0) != nil {
		return args.Get(0).(*models.SnapEntry), nil
	}
	return nil, args.Get(1).(*cerror.CustomError)
}

func (m *MockStoreRepository) GetEntryByName(name string, preloadAssociations []string) (*models.SnapEntry, *cerror.CustomError) {
	args := m.Called(name, preloadAssociations)
	if args.Get(0) != nil {
		return args.Get(0).(*models.SnapEntry), nil
	}
	return nil, args.Get(1).(*cerror.CustomError)
}

func (m *MockStoreRepository) GetRevisionsByEntryId(entryId uuid.UUID) ([]*models.SnapRevision, *cerror.CustomError) {
	args := m.Called(entryId)
	if args.Get(0) != nil {
		return args.Get(0).([]*models.SnapRevision), nil
	}
	return nil, args.Get(1).(*cerror.CustomError)
}

func (m *MockStoreRepository) GetRevisionById(id uuid.UUID) (*models.SnapRevision, *cerror.CustomError) {
	args := m.Called(id)
	if args.Get(0) != nil {
		return args.Get(0).(*models.SnapRevision), nil
	}
	return nil, args.Get(1).(*cerror.CustomError)
}

func (m *MockStoreRepository) GetRevisionByNameAndSequence(name string, sequence uint) (*models.SnapRevision, *cerror.CustomError) {
	args := m.Called(name, sequence)
	if args.Get(0) != nil {
		return args.Get(0).(*models.SnapRevision), nil
	}
	return nil, args.Get(1).(*cerror.CustomError)
}

func (m *MockStoreRepository) GetRevisionBySHA(SHA3_384 string, encoded bool) (*models.SnapRevision, *cerror.CustomError) {
	args := m.Called(SHA3_384, encoded)
	if args.Get(0) != nil {
		return args.Get(0).(*models.SnapRevision), nil
	}
	return nil, args.Get(1).(*cerror.CustomError)
}

func (m *MockStoreRepository) GetSections() (*[]string, *cerror.CustomError) {
	args := m.Called()
	if args.Get(0) != nil {
		return args.Get(0).(*[]string), nil
	}
	return nil, args.Get(1).(*cerror.CustomError)
}

func (m *MockStoreRepository) GetTracksBySnapId(snapId uuid.UUID) ([]*models.SnapTrack, *cerror.CustomError) {
	args := m.Called(snapId)
	if args.Get(0) != nil {
		return args.Get(0).([]*models.SnapTrack), nil
	}
	return nil, args.Get(1).(*cerror.CustomError)
}

// UPDATE
func (m *MockStoreRepository) ReleaseSnap(channels []string, snapEntryId uuid.UUID, revisionId uuid.UUID) *cerror.CustomError {
	args := m.Called(channels, snapEntryId, revisionId)
	return args.Get(0).(*cerror.CustomError)
}

func (m *MockStoreRepository) UpdateRevision(revision *models.SnapRevision, revisionBytes *[]byte) (*models.SnapRevision, *cerror.CustomError) {
	args := m.Called(revision, revisionBytes)
	if args.Get(0) != nil {
		return args.Get(0).(*models.SnapRevision), nil
	}
	return nil, args.Get(1).(*cerror.CustomError)
}
