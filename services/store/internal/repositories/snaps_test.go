package repositories_test

import (
	"io"
	"os"
	"testing"

	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3" // SQLite driver
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"

	_ "github.com/golang-migrate/migrate/v4/source/file" // needed for file source

	cerror "github.com/idlab-discover/kebeng/common/cerror"
	storeDB "github.com/idlab-discover/kebeng/services/store/internal"
	"github.com/idlab-discover/kebeng/services/store/internal/config"
	"github.com/idlab-discover/kebeng/services/store/internal/models"
	"github.com/idlab-discover/kebeng/services/store/internal/repositories"
)

var (
	globalRepo *repositories.SnapsRepository
	globalDB   *sqlx.DB
	cleanupDB  func()
	mockUUID   = uuid.New()
)

func setupGlobalTestDB() (*repositories.SnapsRepository, *sqlx.DB, func()) {
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

	repo := repositories.NewSnapsRepository(db)

	_, err2 := repo.RegisterSnap("test", false)
	if err2 != nil {
		logrus.Fatalf("failed to register existing snap: %v", err2)
	}

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

func TestAddRevision(t *testing.T) {
	tests := []struct {
		name              string
		snapEntry         models.SnapEntry
		size              uint64
		expectError       bool
		expectedErrorCode string
	}{
		{
			name: "Success adding revision",
			snapEntry: models.SnapEntry{
				ID:   mockUUID,
				Name: "mock-snap",
			},
			size:        999,
			expectError: false,
		},
		{
			name: "Fail adding revision for non-existing snap entry",
			snapEntry: models.SnapEntry{
				ID:   uuid.New(),
				Name: "nonexistent",
			},
			size:              999,
			expectError:       true,
			expectedErrorCode: cerror.ResourceNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := globalRepo.AddRevision(&tt.snapEntry, tt.size)
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

// Not sure if this is needed
// func TestAddSnap(t *testing.T) {
// 	tests := []struct {
// 		name              string
// 		snapName          string
// 		size              uint64
// 		accountId         uuid.UUID
// 		expectError       bool
// 		expectedErrorCode string
// 	}{
// 		{
// 			name:      "Success adding snap",
// 			snapName:  "test-1",
// 			size:      999,
// 			accountId: uuid.New(),
// 		},
// 		{
// 			name:              "Fail adding snap",
// 			snapName:          "mock-snap",
// 			size:              999,
// 			expectError:       true,
// 			expectedErrorCode: cerror.AlreadyRegistered,
// 		},
// 	}

// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			_, err := globalRepo.AddSnap(tt.snapName, tt.size, uuid.New())
// 			if tt.expectError {
// 				assert.NotNil(t, err)
// 				assert.Equal(t, tt.expectedErrorCode, err.GetCode())
// 			} else {
// 				assert.Nil(t, err)
// 			}
// 		})
// 	}
// }

func AddUpload(t *testing.T) {
	tests := []struct {
		name              string
		snapName          string
		upDownId          string
		fileSize          uint
		channels          []string
		expectError       bool
		expectedErrorCode string
	}{
		{
			name:              "Success adding upload",
			snapName:          "mock-snap",
			upDownId:          "test-up-down-id",
			fileSize:          999,
			channels:          []string{"latest", "beta"},
			expectError:       false,
			expectedErrorCode: "",
		},
		{
			name:              "Fail adding upload for non-existing snap entry",
			snapName:          "nonexistent",
			upDownId:          "test-up-down-id",
			fileSize:          999,
			channels:          []string{"latest"},
			expectError:       true,
			expectedErrorCode: cerror.ResourceNotFound,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := globalRepo.AddUpload(tt.snapName, tt.upDownId, tt.fileSize, tt.channels)
			if tt.expectError {
				assert.NotNil(t, err)
				if err != nil {
					assert.Equal(t, tt.expectedErrorCode, err.GetCode())
				}
			} else {
				assert.Nil(t, err)
				assert.NotNil(t, resp.ID)
			}
		})
	}
}

func TestRegisterSnap(t *testing.T) {
	tests := []struct {
		name              string
		entryName         string
		entryPrivate      bool
		expectError       bool
		expectedErrorCode string
	}{
		{
			name:              "Success adding snap",
			entryName:         "test-1",
			entryPrivate:      false,
			expectError:       false,
			expectedErrorCode: "",
		},
		{
			name:              "Fail adding snap",
			entryName:         "mock-snap",
			entryPrivate:      false,
			expectError:       true,
			expectedErrorCode: cerror.AlreadyRegistered,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry, err := globalRepo.RegisterSnap(tt.entryName, tt.entryPrivate)
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

func TestAddUpload(t *testing.T) {
	tests := []struct {
		name              string
		snapName          string
		upDownId          string
		fileSize          uint
		channels          []string
		expectError       bool
		expectedErrorCode string
	}{
		{
			name:              "Success adding upload",
			snapName:          "mock-snap",
			upDownId:          "test-up-down-id",
			fileSize:          999,
			channels:          []string{"latest", "beta"},
			expectError:       false,
			expectedErrorCode: "",
		},
		{
			name:              "Fail adding upload for non-existing snap entry",
			snapName:          "nonexistent",
			upDownId:          "test-up-down-id",
			fileSize:          999,
			channels:          []string{"latest"},
			expectError:       true,
			expectedErrorCode: cerror.ResourceNotFound,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := globalRepo.AddUpload(tt.snapName, tt.upDownId, tt.fileSize, tt.channels)
			if tt.expectError {
				assert.NotNil(t, err)
				if err != nil {
					assert.Equal(t, tt.expectedErrorCode, err.GetCode())
				}
			} else {
				assert.Nil(t, err)
				assert.NotNil(t, resp.ID)
			}
		})
	}
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

// func TestGetEntryByName(t *testing.T) {
// 	_, err := globalRepo.RegisterSnap("test-1", false)
// 	tests := []struct {
// 		name              string
// 		entryName         string
// 		entryPrivate      bool
// 		expectError       bool
// 		expectedErrorCode string
// 	}{
// 		{
// 			name:              "Success getting entry by name",
// 			entryName:         "test-1",
// 			entryPrivate:      false,
// 			expectError:       false,
// 			expectedErrorCode: "",
// 		},
// 		{
// 			name:              "Fail getting entry by name",
// 			entryName:         "nonexistent",
// 			entryPrivate:      false,
// 			expectError:       true,
// 			expectedErrorCode: cerror.ResourceNotFound,
// 		},
// 	}

// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			entry, err := globalRepo.GetEntryByName(tt.entryName, nil)
// 			if tt.expectError {
// 				assert.Equal(t, tt.expectedErrorCode, err.GetCode())
// 			} else {
// 				assert.Nil(t, err)
// 				assert.NotNil(t, entry.ID)
// 				assert.Equal(t, tt.entryName, entry.Name)
// 			}
// 		})
// 	}
// }

// func TestGetRevisionByChannel(t *testing.T) {
// 	_, err := globalRepo.AddSnap("test-2", 999, uuid.New())
// 	if err != nil {
// 		logrus.Errorf("failed to add snap: %s", err)
// 	}
// 	tests := []struct {
// 		name              string
// 		entryName         string
// 		channel           string
// 		expectError       bool
// 		expectedErrorCode string
// 	}{
// 		{
// 			name:              "Success getting revision by channel",
// 			entryName:         "test-2",
// 			channel:           "latest",
// 			expectError:       false,
// 			expectedErrorCode: "",
// 		},
// 		{
// 			name:              "Fail getting revision by channel",
// 			entryName:         "test",
// 			channel:           "nonexistent",
// 			expectError:       true,
// 			expectedErrorCode: cerror.ResourceNotFound,
// 		},
// 	}

// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			revision, err := globalRepo.GetRevisionByChannel(tt.channel, tt.entryName)
// 			if tt.expectError {
// 				assert.Equal(t, err.GetCode(), tt.expectedErrorCode)
// 			} else {
// 				assert.Nil(t, err)
// 				assert.NotNil(t, revision.ID)
// 				assert.Equal(t, tt.channel, revision)
// 			}
// 		})
// 	}
// }

func mockData(db *sqlx.DB) {
	// Mock snap entry
	_, err := db.Exec(`
		INSERT INTO public.snap_entries (id, private, name, type, confinement, status, price, store, icon_url)
		VALUES ($1, false, 'mock-snap', 'application', 'strict', 'active', 0.0, 'mock-store', 'http://mock-icon-url.com');
	`, mockUUID)
	if err != nil {
		logrus.Fatalf("failed to insert mock data for snap entry: %v", err)
	}

	// Mock snap revision
	_, err = db.Exec(`
		INSERT INTO public.snap_revisions (id, snap_name, snap_entry_id, build_assertion_filename, sha3_384, size, sequence_number, architectures, status, version)
		VALUES ($1, 'mock-snap', $2, 'mock-build-assertion', 'mock-sha3-384', 999, 1, ARRAY['mock-arch'], 'active', '1.0.0');
	`, mockUUID, mockUUID)
	if err != nil {
		logrus.Fatalf("failed to insert mock data for snap revision: %v", err)
	}

	// Mock snap track
	_, err = db.Exec(`
		INSERT INTO public.snap_tracks (id, name, snap_entry_id)
		VALUES ($1, 'latest', $2);
	`, mockUUID, mockUUID)
	if err != nil {
		logrus.Fatalf("failed to insert mock data for snap track: %v", err)
	}

	// Mock snap channel
	_, err = db.Exec(`
		INSERT INTO public.snap_channels (id, name, snap_track_id, snap_entry_id, revision_id)
		VALUES ($1, 'latest', $2, $3, $4);
	`, mockUUID, mockUUID, mockUUID, mockUUID)
	if err != nil {
		logrus.Fatalf("failed to insert mock data for snap channel: %v", err)
	}

	// Mock snap comment
	_, err = db.Exec(`
		INSERT INTO public.snap_comments (id, snap_entry_id, author_id, reason, comment)
		VALUES ($1, $2, $3, 'mock-reason', 'mock-comment');
	`, mockUUID, mockUUID, mockUUID)
	if err != nil {
		logrus.Fatalf("failed to insert mock data for snap comment: %v", err)
	}
}
