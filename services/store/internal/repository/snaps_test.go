package repository_test

import (
	"io"
	"os"
	"testing"
	"time"

	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3" // SQLite driver
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"

	_ "github.com/golang-migrate/migrate/v4/source/file" // needed for file source

	cerror "github.com/idlab-discover/kebeng/common/cerror"
	"github.com/idlab-discover/kebeng/services/store/internal/config"
	storeDB "github.com/idlab-discover/kebeng/services/store/internal/database"
	"github.com/idlab-discover/kebeng/services/store/internal/models"
	"github.com/idlab-discover/kebeng/services/store/internal/repository"
)

var (
	globalRepo repository.ISnapsRepository
	globalDB   *sqlx.DB
	cleanupDB  func()
	mockUUID   = uuid.New()
)

func setupGlobalTestDB() (repository.ISnapsRepository, *sqlx.DB, func()) {
	postgres := embeddedpostgres.NewDatabase(embeddedpostgres.DefaultConfig().
		Port(5433).
		Version(embeddedpostgres.V12).
		Logger(io.Discard),
	)

	if postgresStartErr := postgres.Start(); postgresStartErr != nil {
		logrus.Fatalf("failed to start embedded postgres: %v", postgresStartErr)
	}

	dsn := "postgres://postgres:postgres@localhost:5433/postgres?sslmode=disable"
	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		logrus.Fatalf("failed to connect to embedded postgres: %v", err)
	}

	cfg := config.Config{MigrationPath: "../migrations"}
	err = storeDB.RunMigrations(db, &cfg)
	if err != nil {
		logrus.Fatalf("failed to run migrations: %v", err)
	}

	repo := repository.NewSnapsRepository(db)

	cleanup := func() {
		err := db.Close()
		if err != nil {
			logrus.Fatalf("failed to close db: %v", err)
		}
		if err := postgres.Stop(); err != nil {
			logrus.Fatalf("failed to stop embedded postgres: %v", err)
		}
	}
	return repo, db, cleanup
}

func TestMain(m *testing.M) {
	repo, db, cleanup := setupGlobalTestDB()
	globalRepo = repo
	globalDB = db
	cleanupDB = cleanup

	mockData(globalDB)

	code := m.Run()

	cleanupDB()
	os.Exit(code)
}

func TestAddChannel(t *testing.T) {
	tests := []struct {
		name              string
		snapEntryId       uuid.UUID
		trackId           uuid.UUID
		channelName       string
		el                *cerror.ErrorList
		expectError       bool
		expectedErrorCode string
	}{
		{
			name:        "Success adding channel",
			snapEntryId: mockUUID,
			trackId:     mockUUID,
			channelName: "mock-channel",
			el:          cerror.NewErrorList(),
			expectError: false,
		},
		{
			name:        "Succes adding channel for already existing channel",
			snapEntryId: mockUUID,
			trackId:     mockUUID,
			channelName: "stable",
			el:          cerror.NewErrorList(),
			expectError: false,
		},
		{
			name:              "Fail adding channel for non-existing snap entry",
			snapEntryId:       uuid.New(),
			trackId:           mockUUID,
			channelName:       "mock-channel",
			el:                cerror.NewErrorList(),
			expectError:       true,
			expectedErrorCode: cerror.ResourceNotFound,
		},
		{
			name:              "Fail adding channel for non-existing track",
			snapEntryId:       mockUUID,
			trackId:           uuid.New(),
			channelName:       "mock-channel",
			el:                cerror.NewErrorList(),
			expectError:       true,
			expectedErrorCode: cerror.ResourceNotFound,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := globalRepo.AddChannel(tt.snapEntryId, tt.trackId, tt.channelName, tt.el)
			if tt.expectError {
				assert.NotNil(t, err)
				if err != nil {
					assert.Equal(t, tt.expectedErrorCode, err.GetCode())
				}
			} else {
				assert.Nil(t, err)
				assert.NotNil(t, resp)
			}
		})
	}
}

func TestAddDefaultChannels(t *testing.T) {
	tests := []struct {
		name              string
		snapEntryId       uuid.UUID
		snapTrackId       uuid.UUID
		el                *cerror.ErrorList
		expectError       bool
		expectedErrorCode string
	}{
		{
			name:        "Success adding default channels",
			snapEntryId: mockUUID,
			snapTrackId: mockUUID,
			el:          cerror.NewErrorList(),
			expectError: false,
		},
		{
			name:              "Fail adding default channels for non-existing snap entry",
			snapEntryId:       uuid.New(),
			snapTrackId:       mockUUID,
			el:                cerror.NewErrorList(),
			expectError:       true,
			expectedErrorCode: cerror.ResourceNotFound,
		},
		{
			name:              "Fail adding default channels for non-existing track",
			snapEntryId:       mockUUID,
			snapTrackId:       uuid.New(),
			el:                cerror.NewErrorList(),
			expectError:       true,
			expectedErrorCode: cerror.ResourceNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := globalRepo.AddDefaultChannels(tt.snapEntryId, tt.snapTrackId, tt.el)
			if tt.expectError {
				assert.NotNil(t, err)
				if err != nil {
					assert.Equal(t, tt.expectedErrorCode, err.GetCode())
				}
			} else {
				assert.Nil(t, err)
			}
		})
	}
}

func TestAddTrack(t *testing.T) {
	tests := []struct {
		name              string
		snapEntryId       uuid.UUID
		trackName         string
		expectError       bool
		expectedErrorCode string
	}{
		{
			name:        "Success adding track",
			snapEntryId: mockUUID,
			trackName:   "mock-track",
			expectError: false,
		},
		{
			name:        "Succes adding track for already existing track",
			snapEntryId: mockUUID,
			trackName:   "latest",
			expectError: false,
		},
		{
			name:              "Fail adding track for non-existing snap entry",
			snapEntryId:       uuid.New(),
			trackName:         "mock-track",
			expectError:       true,
			expectedErrorCode: cerror.ResourceNotFound,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := globalRepo.AddTrack(tt.snapEntryId, tt.trackName)
			if tt.expectError {
				assert.NotNil(t, err)
				if err != nil {
					assert.Equal(t, tt.expectedErrorCode, err.GetCode())
				}
			} else {
				assert.Nil(t, err)
				assert.NotNil(t, resp)
			}
		})
	}
}

func TestAddRevision(t *testing.T) {
	tests := []struct {
		name              string
		entryId           uuid.UUID
		trackId           uuid.UUID
		channelId         uuid.UUID
		snapName          string
		size              uint64
		sequenceNumber    uint
		architectures     []string
		sha3384           string
		minioFilePath     string
		el                *cerror.ErrorList
		expectError       bool
		expectedErrorCode string
	}{
		{
			name:           "Success adding revision",
			entryId:        mockUUID,
			trackId:        mockUUID,
			channelId:      mockUUID,
			snapName:       "mock-snap",
			size:           123456,
			sequenceNumber: 1,
			architectures:  []string{"x86_64", "arm64"},
			sha3384:        "mock-sha3-384",
			minioFilePath:  "some/path/mock-snap.snap",
			el:             cerror.NewErrorList(),
			expectError:    false,
		},
		{
			name:              "Fail adding revision for non-existing entry",
			entryId:           uuid.New(),
			trackId:           mockUUID,
			channelId:         mockUUID,
			snapName:          "mock-snap",
			size:              123456,
			sequenceNumber:    1,
			architectures:     []string{"x86_64", "arm64"},
			sha3384:           "mock-sha3-384",
			minioFilePath:     "some/path/mock-snap.snap",
			el:                cerror.NewErrorList(),
			expectError:       true,
			expectedErrorCode: cerror.ResourceNotFound,
		},
		{
			name:              "Fail adding revision for non-existing track",
			entryId:           mockUUID,
			trackId:           uuid.New(),
			channelId:         mockUUID,
			snapName:          "mock-snap",
			size:              123456,
			sequenceNumber:    1,
			architectures:     []string{"x86_64", "arm64"},
			sha3384:           "mock-sha3-384",
			minioFilePath:     "some/path/mock-snap.snap",
			el:                cerror.NewErrorList(),
			expectError:       true,
			expectedErrorCode: cerror.ResourceNotFound,
		},
		{
			name:              "Fail adding revision for non-existing channel",
			entryId:           mockUUID,
			trackId:           mockUUID,
			channelId:         uuid.New(),
			snapName:          "mock-snap",
			size:              123456,
			sequenceNumber:    1,
			architectures:     []string{"x86_64", "arm64"},
			sha3384:           "mock-sha3-384",
			minioFilePath:     "some/path/mock-snap.snap",
			el:                cerror.NewErrorList(),
			expectError:       true,
			expectedErrorCode: cerror.ResourceNotFound,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			revision, err := globalRepo.AddRevision(tt.entryId, tt.trackId, tt.channelId, tt.snapName, tt.size, tt.sequenceNumber, tt.architectures, tt.sha3384, tt.minioFilePath, tt.el)
			if tt.expectError {
				assert.NotNil(t, err)
				if err != nil {
					assert.Equal(t, tt.expectedErrorCode, err.GetCode())
				}
			} else {
				assert.Nil(t, err)
				assert.NotNil(t, revision)
			}
		})
	}
}

func TestRegisterSnap(t *testing.T) {
	tests := []struct {
		name              string
		entryName         string
		entryPrivate      bool
		storeName         string
		accountId         uuid.UUID
		expectError       bool
		expectedErrorCode string
	}{
		{
			name:              "Success adding snap",
			entryName:         "test-1",
			entryPrivate:      false,
			storeName:         "test-store",
			accountId:         mockUUID,
			expectError:       false,
			expectedErrorCode: "",
		},
		{
			name:              "Fail adding snap",
			entryName:         "mock-snap",
			entryPrivate:      false,
			storeName:         "test-store",
			accountId:         mockUUID,
			expectError:       true,
			expectedErrorCode: cerror.AlreadyRegistered,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry, err := globalRepo.RegisterSnap(tt.entryName, tt.entryPrivate, tt.storeName, tt.accountId)
			if tt.expectError {
				assert.NotNil(t, err)
				if err != nil {
					assert.Equal(t, tt.expectedErrorCode, err.GetCode())
				}
			} else {
				assert.Nil(t, err)
				assert.NotNil(t, entry.ID)
			}
		})
	}
}

func TestGetAllSnapEntries(t *testing.T) {
	snaps, err := globalRepo.GetAllSnapEntries()
	assert.Nil(t, err)
	assert.NotNil(t, snaps)
}

func TestGetChannelsByTrackId(t *testing.T) {
	tests := []struct {
		name              string
		trackId           uuid.UUID
		expectError       bool
		expectedErrorCode string
	}{
		{
			name:              "Success getting channels by track id",
			trackId:           mockUUID,
			expectError:       false,
			expectedErrorCode: "",
		},
		{
			name:              "Fail getting channels by track id for non-existing track",
			trackId:           uuid.New(),
			expectError:       true,
			expectedErrorCode: cerror.ResourceNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			channels, err := globalRepo.GetChannelsByTrackId(tt.trackId)
			if tt.expectError {
				assert.NotNil(t, err)
				if err != nil {
					assert.Equal(t, tt.expectedErrorCode, err.GetCode())
				}
			} else {
				assert.Nil(t, err)
				assert.NotNil(t, channels)
			}
		})
	}
}

func TestGetCommentsByEntryId(t *testing.T) {
	tests := []struct {
		name              string
		entryId           uuid.UUID
		expectError       bool
		expectedErrorCode string
	}{
		{
			name:              "Success getting comments by entry id",
			entryId:           mockUUID,
			expectError:       false,
			expectedErrorCode: "",
		},
		{
			name:              "Fail getting comments by entry id for non-existing entry",
			entryId:           uuid.New(),
			expectError:       true,
			expectedErrorCode: cerror.ResourceNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			comments, err := globalRepo.GetCommentsByEntryId(tt.entryId)
			if tt.expectError {
				assert.NotNil(t, err)
				if err != nil {
					assert.Equal(t, tt.expectedErrorCode, err.GetCode())
				}
			} else {
				assert.Nil(t, err)
				assert.NotNil(t, comments)
			}
		})
	}
}

func TestGetEntriesByAccountId(t *testing.T) {
	tests := []struct {
		name                string
		accountId           uuid.UUID
		preloadAssociations []string
		el                  *cerror.ErrorList
		expectError         bool
		expectedErrorCode   string
	}{
		{
			name:                "Success getting entries by account id",
			accountId:           mockUUID,
			preloadAssociations: nil,
			el:                  cerror.NewErrorList(),
			expectError:         false,
			expectedErrorCode:   "",
		},
		{
			name:                "Fail getting entries by account id for non-existing account",
			accountId:           uuid.New(),
			preloadAssociations: nil,
			el:                  cerror.NewErrorList(),
			expectError:         true,
			expectedErrorCode:   cerror.ResourceNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entries, err := globalRepo.GetEntriesByAccountId(tt.accountId, tt.preloadAssociations, tt.el)
			if tt.expectError {
				assert.NotNil(t, err)
				if err != nil {
					assert.Equal(t, tt.expectedErrorCode, err.GetCode())
				}
			} else {
				assert.Nil(t, err)
				assert.NotNil(t, entries)
			}
		})
	}
}

func TestGetEntryById(t *testing.T) {
	tests := []struct {
		name                string
		entryId             uuid.UUID
		preloadAssociations []string
		el                  *cerror.ErrorList
		expectError         bool
		expectedErrorCode   string
	}{
		{
			name:                "Success getting entry by id",
			entryId:             mockUUID,
			preloadAssociations: nil,
			el:                  cerror.NewErrorList(),
			expectError:         false,
			expectedErrorCode:   "",
		},
		{
			name:                "Fail getting entry by id for non-existing entry",
			entryId:             uuid.New(),
			preloadAssociations: nil,
			el:                  cerror.NewErrorList(),
			expectError:         true,
			expectedErrorCode:   cerror.ResourceNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry, err := globalRepo.GetEntryById(tt.entryId, tt.preloadAssociations, tt.el)
			if tt.expectError {
				assert.NotNil(t, err)
				if err != nil {
					assert.Equal(t, tt.expectedErrorCode, err.GetCode())
				}
			} else {
				assert.Nil(t, err)
				assert.NotNil(t, entry)
			}
		})
	}
}

func TestGetEntryByName(t *testing.T) {
	tests := []struct {
		name                string
		entryName           string
		preloadAssociations []string
		el                  *cerror.ErrorList
		expectError         bool
		expectedErrorCode   string
	}{
		{
			name:                "Success getting entry by name",
			entryName:           "mock-snap",
			preloadAssociations: nil,
			el:                  cerror.NewErrorList(),
			expectError:         false,
			expectedErrorCode:   "",
		},
		{
			name:                "Fail getting entry by name",
			entryName:           "nonexistent",
			preloadAssociations: nil,
			el:                  cerror.NewErrorList(),
			expectError:         true,
			expectedErrorCode:   cerror.ResourceNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry, err := globalRepo.GetEntryByName(tt.entryName, tt.preloadAssociations, tt.el)
			if tt.expectError {
				assert.NotNil(t, err)
				if err != nil {
					assert.Equal(t, tt.expectedErrorCode, err.GetCode())
				}
			} else {
				assert.Nil(t, err)
				assert.NotNil(t, entry)
			}
		})
	}
}

func TestGetRevisionsByEntryId(t *testing.T) {
	tests := []struct {
		name              string
		entryId           uuid.UUID
		expectError       bool
		expectedErrorCode string
	}{
		{
			name:              "Success getting revisions by entry id",
			entryId:           mockUUID,
			expectError:       false,
			expectedErrorCode: "",
		},
		{
			name:              "Fail getting revisions by entry id for non-existing entry",
			entryId:           uuid.New(),
			expectError:       true,
			expectedErrorCode: cerror.ResourceNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			revisions, err := globalRepo.GetRevisionsByEntryId(tt.entryId)
			if tt.expectError {
				assert.NotNil(t, err)
				if err != nil {
					assert.Equal(t, tt.expectedErrorCode, err.GetCode())
				}
			} else {
				assert.Nil(t, err)
				assert.NotNil(t, revisions)
			}
		})
	}
}

func TestGetRevisionById(t *testing.T) {
	tests := []struct {
		name              string
		revisionId        uuid.UUID
		expectError       bool
		expectedErrorCode string
	}{
		{
			name:              "Success getting revision by id",
			revisionId:        mockUUID,
			expectError:       false,
			expectedErrorCode: "",
		},
		{
			name:              "Fail getting revision by id for non-existing revision",
			revisionId:        uuid.New(),
			expectError:       true,
			expectedErrorCode: cerror.ResourceNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			revision, err := globalRepo.GetRevisionById(tt.revisionId)
			if tt.expectError {
				assert.NotNil(t, err)
				if err != nil {
					assert.Equal(t, tt.expectedErrorCode, err.GetCode())
				}
			} else {
				assert.Nil(t, err)
				assert.NotNil(t, revision)
			}

		})
	}
}

func TestGetRevisionByNameAndSequence(t *testing.T) {
	tests := []struct {
		name              string
		entryName         string
		sequence          uint
		expectError       bool
		expectedErrorCode string
	}{
		{
			name:              "Success getting revision by name and sequence",
			entryName:         "mock-snap",
			sequence:          1,
			expectError:       false,
			expectedErrorCode: "",
		},
		{
			name:              "Fail getting revision by name and sequence for non-existing entry",
			entryName:         "nonexistent",
			sequence:          1,
			expectError:       true,
			expectedErrorCode: cerror.ResourceNotFound,
		},
		{
			name:              "Fail getting revision by name and sequence for non-existing sequence",
			entryName:         "mock-snap",
			sequence:          999,
			expectError:       true,
			expectedErrorCode: cerror.ResourceNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			revision, err := globalRepo.GetRevisionByNameAndSequence(tt.entryName, tt.sequence)
			if tt.expectError {
				assert.NotNil(t, err)
				if err != nil {
					assert.Equal(t, tt.expectedErrorCode, err.GetCode())
				}
			} else {
				assert.Nil(t, err)
				assert.NotNil(t, revision)
			}
		})
	}
}

func TestGetRevisionBySHA(t *testing.T) {
	tests := []struct {
		name              string
		sha               string
		el                *cerror.ErrorList
		expectError       bool
		expectedErrorCode string
	}{
		{
			name:              "Success getting revision by sha",
			sha:               "mock-sha3-384",
			el:                cerror.NewErrorList(),
			expectError:       false,
			expectedErrorCode: "",
		},
		{
			name:              "Fail getting revision by sha for non-existing revision",
			sha:               "nonexistent",
			el:                cerror.NewErrorList(),
			expectError:       true,
			expectedErrorCode: cerror.ResourceNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			revision, err := globalRepo.GetRevisionBySHA(tt.sha, false, tt.el)
			if tt.expectError {
				assert.NotNil(t, err)
				if err != nil {
					assert.Equal(t, tt.expectedErrorCode, err.GetCode())
				}
			} else {
				assert.Nil(t, err)
				assert.NotNil(t, revision)
			}
		})
	}
}

func TestGetRevisionBySHAEncoded(t *testing.T) {
	tests := []struct {
		name              string
		sha               string
		el                *cerror.ErrorList
		expectError       bool
		expectedErrorCode string
	}{
		{
			name:              "Success getting revision by sha",
			sha:               "mock-sha3-384-encoded",
			el:                cerror.NewErrorList(),
			expectError:       false,
			expectedErrorCode: "",
		},
		{
			name:              "Fail getting revision by sha for non-existing revision",
			sha:               "nonexistent",
			el:                cerror.NewErrorList(),
			expectError:       true,
			expectedErrorCode: cerror.ResourceNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			revision, err := globalRepo.GetRevisionBySHA(tt.sha, true, tt.el)
			if tt.expectError {
				assert.NotNil(t, err)
				if err != nil {
					assert.Equal(t, tt.expectedErrorCode, err.GetCode())
				}
			} else {
				assert.Nil(t, err)
				assert.NotNil(t, revision)
			}
		})
	}
}

func TestGetTracksBySnapId(t *testing.T) {
	tests := []struct {
		name              string
		snapId            uuid.UUID
		expectError       bool
		expectedErrorCode string
	}{
		{
			name:              "Success getting tracks by snap id",
			snapId:            mockUUID,
			expectError:       false,
			expectedErrorCode: "",
		},
		{
			name:              "Fail getting tracks by snap id for non-existing snap",
			snapId:            uuid.New(),
			expectError:       true,
			expectedErrorCode: cerror.ResourceNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tracks, err := globalRepo.GetTracksByEntryId(tt.snapId)
			if tt.expectError {
				assert.NotNil(t, err)
				if err != nil {
					assert.Equal(t, tt.expectedErrorCode, err.GetCode())
				}
			} else {
				assert.Nil(t, err)
				assert.NotNil(t, tracks)
			}
		})
	}
}

func TestGetPreloadAssociations(t *testing.T) {
	tests := []struct {
		name              string
		entry             *models.SnapEntry
		associations      *[]string
		el                *cerror.ErrorList
		expectError       bool
		expectedErrorCode string
	}{
		{
			name: "Success getting preload associations",
			entry: &models.SnapEntry{
				ID: mockUUID,
			},
			associations:      &[]string{models.ALL},
			el:                cerror.NewErrorList(),
			expectError:       false,
			expectedErrorCode: "",
		},
		{
			name: "Fail getting preload associations for non-existing entry",
			entry: &models.SnapEntry{
				ID: uuid.New(),
			},
			associations:      &[]string{models.ALL},
			el:                cerror.NewErrorList(),
			expectError:       true,
			expectedErrorCode: cerror.ResourceNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := globalRepo.GetPreloadAssociations(tt.entry, tt.associations, tt.el)
			if tt.expectError {
				assert.NotNil(t, err)
				if err != nil {
					assert.Equal(t, tt.expectedErrorCode, err.GetCode())
				}
			}
		})
	}
}

func TestReleaseSnap(t *testing.T) {
	tests := []struct {
		name              string
		channels          []string
		snapEntryId       uuid.UUID
		revisionID        uuid.UUID
		expectError       bool
		expectedErrorCode string
	}{
		{
			name:              "Success releasing snap to multiple channels",
			channels:          []string{"stable", "latest/stable"},
			snapEntryId:       mockUUID,
			revisionID:        mockUUID,
			expectError:       false,
			expectedErrorCode: "",
		},
		{
			name:              "Fail releasing snap to multiple channels for non-existing snap entry",
			channels:          []string{"stable", "latest/stable"},
			snapEntryId:       uuid.New(),
			revisionID:        mockUUID,
			expectError:       true,
			expectedErrorCode: cerror.ResourceNotFound,
		},
		{
			name:              "Fail releasing snap to multiple channels for non-existing revision",
			channels:          []string{"stable", "latest/stable"},
			snapEntryId:       mockUUID,
			revisionID:        uuid.New(),
			expectError:       true,
			expectedErrorCode: cerror.ResourceNotFound,
		},
		{
			name:              "Fail releasing snap to multiple channels for non-existing channel",
			channels:          []string{"nonexistent"},
			snapEntryId:       mockUUID,
			revisionID:        mockUUID,
			expectError:       true,
			expectedErrorCode: cerror.ResourceNotFound,
		},
		{
			name:              "Fail releasing snap to multiple channels for non-existing track",
			channels:          []string{"nonexistent/stable"},
			snapEntryId:       mockUUID,
			revisionID:        mockUUID,
			expectError:       true,
			expectedErrorCode: cerror.ResourceNotFound,
		},
		{
			name:              "Fail releasing snap to multiple channels for non-existing track and channel",
			channels:          []string{"nonexistent/nonexistent"},
			snapEntryId:       mockUUID,
			revisionID:        mockUUID,
			expectError:       true,
			expectedErrorCode: cerror.ResourceNotFound,
		},
		{
			name:              "Fail releasing snap because of specifying a branch (not supported yet)",
			channels:          []string{"latest/stable/branch"},
			expectError:       true,
			expectedErrorCode: cerror.NotImplemented,
		},
		{
			name:              "Fail releasing snap because of incorrect channel format",
			channels:          []string{"latest/stable/branch/extra"},
			expectError:       true,
			expectedErrorCode: cerror.InvalidField,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := globalRepo.ReleaseSnap(tt.channels, tt.snapEntryId, tt.revisionID)
			if tt.expectError {
				assert.NotNil(t, err)
				if err != nil {
					assert.Equal(t, tt.expectedErrorCode, err.GetCode())
				}
			}
		})
	}
}

func TestUpdateRevision(t *testing.T) {
	buildAssertionFileName := "mock-build-assertion"
	sha3_384 := "mock-sha3-384"
	sha3_384_Encoded := "mock-sha3-384-encoded"
	size := uint64(999)
	sequenceNumber := uint(1)
	architectures := pq.StringArray{"mock-arch"}
	tests := []struct {
		name              string
		revision          models.SnapRevision
		expectError       bool
		expectedErrorCode string
	}{
		{
			name: "Success updating revision",
			revision: models.SnapRevision{
				ID:                     mockUUID,
				SnapEntryID:            mockUUID,
				BuildAssertionFileName: &buildAssertionFileName,
				SHA3_384:               &sha3_384,
				SHA3_384_Encoded:       &sha3_384_Encoded,
				Size:                   &size,
				SequenceNumber:         &sequenceNumber,
				Architectures:          architectures,
			},
			expectError: false,
		},
		{
			name: "Fail updating revision for non-existing revision",
			revision: models.SnapRevision{
				ID:                     uuid.New(),
				SnapEntryID:            mockUUID,
				BuildAssertionFileName: &buildAssertionFileName,
				SHA3_384:               &sha3_384,
				SHA3_384_Encoded:       &sha3_384_Encoded,
				Size:                   &size,
				SequenceNumber:         &sequenceNumber,
				Architectures:          architectures,
			},
			expectError:       true,
			expectedErrorCode: cerror.ResourceNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			revision, err := globalRepo.UpdateRevision(&tt.revision, nil)
			if tt.expectError {
				assert.NotNil(t, err)
				if err != nil {
					assert.Equal(t, tt.expectedErrorCode, err.GetCode())
				}
			} else {
				assert.Nil(t, err)
				assert.NotNil(t, revision)
			}
		})
	}
}

func TestAddUpload(t *testing.T) {
	tests := []struct {
		name              string
		snapName          string
		entryId           uuid.UUID
		accountId         uuid.UUID
		status            string
		unscannedFileName string
		expectError       bool
		expectedErrorCode string
	}{
		{
			name:              "Success adding upload",
			snapName:          "mock-snap",
			entryId:           mockUUID,
			accountId:         mockUUID,
			status:            "pending",
			unscannedFileName: "mock-file",
			expectError:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			upload, err := globalRepo.AddUpload(tt.snapName, tt.entryId, tt.status, tt.accountId, tt.unscannedFileName)
			if tt.expectError {
				assert.NotNil(t, err)
				if err != nil {
					assert.Equal(t, tt.expectedErrorCode, err.GetCode())
				}
			} else {
				assert.Nil(t, err)
				assert.NotNil(t, upload)
			}
		})
	}
}

func TestGetUploadById(t *testing.T) {
	tests := []struct {
		name              string
		uploadId          uuid.UUID
		expectError       bool
		expectedErrorCode string
	}{
		{
			name:              "Success getting upload by id",
			uploadId:          mockUUID,
			expectError:       false,
			expectedErrorCode: "",
		},
		{
			name:              "Fail getting upload by id for non-existing upload",
			uploadId:          uuid.New(),
			expectError:       true,
			expectedErrorCode: cerror.ResourceNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			upload, err := globalRepo.GetUploadById(tt.uploadId)
			if tt.expectError {
				assert.NotNil(t, err)
				if err != nil {
					assert.Equal(t, tt.expectedErrorCode, err.GetCode())
				}
			} else {
				assert.Nil(t, err)
				assert.NotNil(t, upload)
			}
		})
	}
}

func TestGetChannelByTrackIdAndName(t *testing.T) {
	tests := []struct {
		name              string
		trackId           uuid.UUID
		channelName       string
		el                *cerror.ErrorList
		expectError       bool
		expectedErrorCode string
	}{
		{
			name:              "Success getting channel by track id and name",
			trackId:           mockUUID,
			channelName:       "stable",
			el:                cerror.NewErrorList(),
			expectError:       false,
			expectedErrorCode: "",
		},
		{
			name:              "Fail getting channel by track id and name for non-existing channel",
			trackId:           mockUUID,
			channelName:       "nonexistent",
			el:                cerror.NewErrorList(),
			expectError:       true,
			expectedErrorCode: cerror.ResourceNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			channel, err := globalRepo.GetChannelByTrackIdAndName(tt.trackId, tt.channelName, tt.el)
			if tt.expectError {
				assert.NotNil(t, err)
				if err != nil {
					assert.Equal(t, tt.expectedErrorCode, err.GetCode())
				}
			} else {
				assert.Nil(t, err)
				assert.NotNil(t, channel)
			}
		})
	}
}

func TestGetLatestRevisionByEntryId(t *testing.T) {
	tests := []struct {
		name              string
		entryId           uuid.UUID
		el                *cerror.ErrorList
		expectError       bool
		expectedErrorCode string
	}{
		{
			name:              "Success getting latest revision by entry id",
			entryId:           mockUUID,
			el:                cerror.NewErrorList(),
			expectError:       false,
			expectedErrorCode: "",
		},
		{
			name:              "Fail getting latest revision by entry id for non-existing entry",
			entryId:           uuid.New(),
			el:                cerror.NewErrorList(),
			expectError:       true,
			expectedErrorCode: cerror.ResourceNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			revision, err := globalRepo.GetLatestRevisionByEntryId(tt.entryId, tt.el)
			if tt.expectError {
				assert.NotNil(t, err)
				if err != nil {
					assert.Equal(t, tt.expectedErrorCode, err.GetCode())
				}
			} else {
				assert.Nil(t, err)
				assert.NotNil(t, revision)
			}
		})
	}
}

func TestGetTrackByEntryIdAndName(t *testing.T) {
	tests := []struct {
		name              string
		entryId           uuid.UUID
		trackName         string
		el                *cerror.ErrorList
		expectError       bool
		expectedErrorCode string
	}{
		{
			name:              "Success getting track by entry id and name",
			entryId:           mockUUID,
			trackName:         "latest",
			el:                cerror.NewErrorList(),
			expectError:       false,
			expectedErrorCode: "",
		},
		{
			name:              "Fail getting track by entry id and name for non-existing track",
			entryId:           mockUUID,
			trackName:         "nonexistent",
			el:                cerror.NewErrorList(),
			expectError:       true,
			expectedErrorCode: cerror.ResourceNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			track, err := globalRepo.GetTrackByEntryIdAndName(tt.entryId, tt.trackName, tt.el)
			if tt.expectError {
				assert.NotNil(t, err)
				if err != nil {
					assert.Equal(t, tt.expectedErrorCode, err.GetCode())
				}
			} else {
				assert.Nil(t, err)
				assert.NotNil(t, track)
			}
		})
	}
}

// Helper function to insert mock data
func mockData(db *sqlx.DB) {
	// Mock snap entry
	_, err := db.Exec(`
		INSERT INTO public.entry (id, private, name, type, confinement, status, price, store, icon_url, account_id)
		VALUES ($1, false, 'mock-snap', 'application', 'strict', 'active', 0.0, 'mock-store', 'http://mock-icon-url.com', $2);
	`, mockUUID, mockUUID)
	if err != nil {
		logrus.Fatalf("failed to insert mock data for snap entry: %v", err)
	}

	// Mock snap track
	trackID := mockUUID
	_, err = db.Exec(`
		INSERT INTO public.track (id, name, entry_id)
		VALUES ($1, 'latest', $2);
	`, trackID, mockUUID)
	if err != nil {
		logrus.Fatalf("failed to insert mock data for snap track: %v", err)
	}
	// Mock snap channel
	channelID := mockUUID
	_, err = db.Exec(`
		INSERT INTO public.channel (id, name, snap_track_id, entry_id)
		VALUES ($1, 'stable', $2, $3);
	`, channelID, trackID, mockUUID)
	if err != nil {
		logrus.Fatalf("failed to insert mock data for snap channel: %v", err)
	}

	// Mock snap revision
	revisionID := mockUUID
	_, err = db.Exec(`
		INSERT INTO public.revision (id, entry_id, build_assertion_filename, sha3_384, sha3_384_encoded, size, sequence_number, architectures, snap_track_id, snap_channel_id)
		VALUES ($1, $2, 'mock-build-assertion', 'mock-sha3-384', 'mock-sha3-384-encoded', 999, 1, ARRAY['mock-arch'], $3, $4);
	`, revisionID, mockUUID, trackID, channelID)
	if err != nil {
		logrus.Fatalf("failed to insert mock data for snap revision: %v", err)
	}

	// Mock snap comment
	_, err = db.Exec(`
		INSERT INTO public.comment (id, entry_id, author_id, reason, comment)
		VALUES ($1, $2, $3, 'mock-reason', 'mock-comment');
	`, mockUUID, mockUUID, uuid.New())
	if err != nil {
		logrus.Fatalf("failed to insert mock data for snap comment: %v", err)
	}

	// Mock snap upload
	_, err = db.Exec(`
		INSERT INTO public.upload (id, entry_id, snap_name, status, account_id, unscanned_file_name, revision)
		VALUES ($1, $2, 'test-snap', 'pending', $3, 'mock-file', 1);
	`, mockUUID, mockUUID, mockUUID)
	if err != nil {
		logrus.Fatalf("failed to insert mock data for snap upload: %v", err)
	}
}

func TestGetLatestRevision(t *testing.T) {
	entryID := uuid.New()
	trackID := uuid.New()
	channelID := uuid.New()
	altTrackID := uuid.New()
	altChannelID := uuid.New()

	// Insert mock entry
	_, err := globalDB.Exec(`
		INSERT INTO entry (id, name, private, type, confinement, status, price, store, icon_url, account_id)
		VALUES ($1, 'getLatestRevision', false, 'app', 'strict', 'active', 0.0, 'mock-store', 'http://icon', $1);
	`, entryID)
	assert.NoError(t, err)

	// Insert two tracks
	_, err = globalDB.Exec(`
		INSERT INTO track (id, name, entry_id) 
		VALUES ($1, 'latest', $2), ($3, 'beta', $2);
	`, trackID, entryID, altTrackID)
	assert.NoError(t, err)

	// Insert two channels
	_, err = globalDB.Exec(`
		INSERT INTO channel (id, name, snap_track_id, entry_id) 
		VALUES ($1, 'stable', $2, $3), ($4, 'edge', $5, $3);
	`, channelID, trackID, entryID, altChannelID, altTrackID)
	assert.NoError(t, err)

	// Insert 3 revisions in the correct track/channel
	now := time.Now()
	rev1ID := uuid.New()
	rev2ID := uuid.New()
	rev3ID := uuid.New()
	snapName := "test-snap"
	_, err = globalDB.Exec(`
		INSERT INTO revision (id, entry_id, snap_track_id, snap_channel_id, snap_name, updated_at, sequence_number)
		VALUES 
		($1, $2, $3, $4, $10, $5, 1),
		($6, $2, $3, $4, $10, $7, 2),
		($8, $2, $3, $4, $10, $9, 3);
	`, rev1ID, entryID, trackID, channelID, now.Add(-10*time.Minute),
		rev2ID, now.Add(-5*time.Minute),
		rev3ID, now.Add(-1*time.Minute),
		snapName)
	assert.NoError(t, err)

	// Insert 1 revision in a different track/channel but with the most recent update
	altRevID := uuid.New()
	_, err = globalDB.Exec(`
		INSERT INTO revision (id, entry_id, snap_track_id, snap_channel_id, updated_at, sequence_number)
		VALUES ($1, $2, $3, $4, $5, 99);
	`, altRevID, entryID, altTrackID, altChannelID, now.Add(1*time.Minute))
	assert.NoError(t, err)

	// Call the method under test
	revision, errObj := globalRepo.GetLatestRevisionByTrackAndChannel("getLatestRevision", "latest", "stable")
	assert.Nil(t, errObj)
	assert.NotNil(t, revision)

	assert.Equal(t, int64(3), int64(*revision.SequenceNumber))
	assert.Equal(t, rev3ID, revision.ID)
}

func TestGetTrackById(t *testing.T) {
	trackID := uuid.New()

	_, err := globalDB.Exec(`
		INSERT INTO track (id, name, entry_id)
		VALUES ($1, 'latest', $2);
	`, trackID, mockUUID)
	assert.NoError(t, err, "failed to insert test track")

	track, cerr := globalRepo.GetTrackById(trackID)
	assert.Nil(t, cerr, "expected no error when retrieving an existing track")
	assert.NotNil(t, track, "expected track to be not nil")
	assert.Equal(t, "latest", track.Name, "expected track name to be 'latest'")

	nonExistingID := uuid.New()
	track, cerr = globalRepo.GetTrackById(nonExistingID)
	assert.NotNil(t, cerr, "expected error when retrieving a non-existing track")
	assert.Nil(t, track, "expected nil track for non-existing id")
	if cerr != nil {
		assert.Equal(t, cerror.ResourceNotFound, cerr.GetCode(), "expected error code to be ResourceNotFound")
	}
}

func TestGetChannelById(t *testing.T) {
	channelID := uuid.New()

	_, err := globalDB.Exec(`
		INSERT INTO channel (id, name, snap_track_id, entry_id)
		VALUES ($1, 'stable', $2, $3);
	`, channelID, mockUUID, mockUUID)
	assert.NoError(t, err, "failed to insert test channel")

	// Test retrieving the inserted channel.
	channel, cerr := globalRepo.GetChannelById(channelID)
	assert.Nil(t, cerr, "expected no error when retrieving an existing channel")
	assert.NotNil(t, channel, "expected channel to be not nil")
	assert.Equal(t, "stable", channel.Name, "expected channel name to be 'stable'")

	// Test retrieving a non-existing channel.
	nonExistingID := uuid.New()
	channel, cerr = globalRepo.GetChannelById(nonExistingID)
	assert.NotNil(t, cerr, "expected error when retrieving a non-existing channel")
	assert.Nil(t, channel, "expected nil channel for non-existing id")
	if cerr != nil {
		assert.Equal(t, cerror.ResourceNotFound, cerr.GetCode(), "expected error code to be ResourceNotFound")
	}
}
