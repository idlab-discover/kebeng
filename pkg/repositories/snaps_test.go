package repositories

import (
	"testing"

	"github.com/google/uuid"
	"github.com/idlab-discover/kebeng/pkg/models"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestAddSnap(t *testing.T) {
	// Setup in-memory database
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	// Migrate the schema
	err = db.AutoMigrate(&models.SnapEntry{}, &models.SnapTrack{}, &models.SnapRisk{}, &models.SnapUpload{}, &models.SnapRevision{})
	if err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	repo := NewSnapsRepository(db)

	t.Run("Add new snap successfully", func(t *testing.T) {
		name := "test-snap"
		size := uint64(1024)
		accountId := uint(1)

		snapEntry, err := repo.AddSnap(name, size, accountId)
		assert.NoError(t, err)
		assert.NotNil(t, snapEntry)
		assert.Equal(t, name, snapEntry.Name)
		assert.Equal(t, accountId, snapEntry.AccountID)
	})

	t.Run("Add snap that already exists", func(t *testing.T) {
		name := "existing-snap"
		size := uint64(2048)
		accountId := uint(2)

		// Add the snap first
		_, err := repo.AddSnap(name, size, accountId)
		assert.NoError(t, err)

		// Try to add the same snap again
		_, err = repo.AddSnap(name, size, accountId)
		assert.Error(t, err)
		assert.Equal(t, "snap with name=existing-snap already exists", err.Error())
	})
}

func TestGetSnap(t *testing.T) {
	// Setup in-memory database
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	// Migrate the schema
	err = db.AutoMigrate(&models.SnapEntry{}, &models.SnapTrack{}, &models.SnapRisk{}, &models.SnapUpload{}, &models.SnapRevision{})
	if err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	repo := NewSnapsRepository(db)

	t.Run("Get snap by name", func(t *testing.T) {
		name := "test-snap"
		size := uint64(1024)
		accountId := uint(1)

		_, err := repo.AddSnap(name, size, accountId)
		assert.NoError(t, err)

		snapEntry, err := repo.GetSnap(name, false)
		assert.NoError(t, err)
		assert.NotNil(t, snapEntry)
		assert.Equal(t, name, snapEntry.Name)
	})

	t.Run("Get snap by name that does not exist", func(t *testing.T) {
		snapEntry, err := repo.GetSnap("non-existent-snap", false)
		assert.NoError(t, err)
		assert.Nil(t, snapEntry)
	})
}

func TestGetSnapById(t *testing.T) {
	// Setup in-memory database
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	// Migrate the schema
	err = db.AutoMigrate(&models.SnapEntry{}, &models.SnapTrack{}, &models.SnapRisk{}, &models.SnapUpload{}, &models.SnapRevision{})
	if err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	repo := NewSnapsRepository(db)

	t.Run("Get snap by ID", func(t *testing.T) {
		name := "test-snap"
		size := uint64(1024)
		accountId := uint(1)

		snapEntry, err := repo.AddSnap(name, size, accountId)
		assert.NoError(t, err)

		fetchedSnap, err := repo.GetSnapById(snapEntry.ID, false)
		assert.NoError(t, err)
		assert.NotNil(t, fetchedSnap)
		assert.Equal(t, snapEntry.ID, fetchedSnap.ID)
	})

	t.Run("Get snap by ID that does not exist", func(t *testing.T) {
		invalidUUID := uuid.New()
		snapEntry, err := repo.GetSnapById(invalidUUID, false)
		assert.NoError(t, err)
		assert.Nil(t, snapEntry)
	})
}

func TestGetSnapByStoreId(t *testing.T) {
	// Setup in-memory database
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	// Migrate the schema
	err = db.AutoMigrate(&models.SnapEntry{}, &models.SnapTrack{}, &models.SnapRisk{}, &models.SnapUpload{}, &models.SnapRevision{})
	if err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	repo := NewSnapsRepository(db)

	t.Run("Get snap by store ID", func(t *testing.T) {
		name := "test-snap"
		size := uint64(1024)
		accountId := uint(1)

		snapEntry, err := repo.AddSnap(name, size, accountId)
		assert.NoError(t, err)

		fetchedSnap, err := repo.GetSnapId(snapEntry.ID, false)
		assert.NoError(t, err)
		assert.NotNil(t, fetchedSnap)
		assert.Equal(t, snapEntry.ID, fetchedSnap.ID)
	})

	t.Run("Get snap by store ID that does not exist", func(t *testing.T) {
		invalidUUID := uuid.New()
		snapEntry, err := repo.GetSnapById(invalidUUID, false)
		assert.NoError(t, err)
		assert.Nil(t, snapEntry)
	})
}

func TestGetRevisionByChannel(t *testing.T) {
	// Setup in-memory database
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	// Migrate the schema
	err = db.AutoMigrate(&models.SnapEntry{}, &models.SnapTrack{}, &models.SnapRisk{}, &models.SnapUpload{}, &models.SnapRevision{}, &models.SnapBranch{})
	if err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	repo := NewSnapsRepository(db)

	t.Run("Get revision by channel", func(t *testing.T) {
		name := "test-snap"
		size := uint64(1024)
		accountId := uint(1)

		_, err := repo.AddSnap(name, size, accountId)
		assert.NoError(t, err)

		revision, err := repo.GetRevisionByChannel("latest/stable", name)
		assert.NoError(t, err)
		assert.NotNil(t, revision)
	})

	t.Run("Get revision by channel that does not exist", func(t *testing.T) {
		revision, err := repo.GetRevisionByChannel("non-existent-channel", "non-existent-snap")
		assert.Error(t, err)
		assert.Nil(t, revision)
	})
}
func TestGetTracks(t *testing.T) {
	// Setup in-memory database
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	// Migrate the schema
	err = db.AutoMigrate(&models.SnapEntry{}, &models.SnapTrack{}, &models.SnapRisk{}, &models.SnapUpload{}, &models.SnapRevision{})
	if err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	repo := NewSnapsRepository(db)

	t.Run("Get tracks for existing snap", func(t *testing.T) {
		name := "test-snap"
		size := uint64(1024)
		accountId := uint(1)

		snapEntry, err := repo.AddSnap(name, size, accountId)
		assert.NoError(t, err)

		tracks, err := repo.GetTracks(snapEntry.ID)
		assert.NoError(t, err)
		assert.NotNil(t, tracks)
		assert.Len(t, *tracks, 1) // Default track "latest" should be created
		assert.Equal(t, "latest", (*tracks)[0].Name)
	})

	t.Run("Get tracks for non-existent snap", func(t *testing.T) {
		tracks, err := repo.GetTracks(uuid.New())
		assert.Error(t, err)
		assert.Nil(t, tracks)
		assert.Equal(t, "unknown error encountered", err.Error())
	})
}
func TestGetRisks(t *testing.T) {
	// Setup in-memory database
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	// Migrate the schema
	err = db.AutoMigrate(&models.SnapEntry{}, &models.SnapTrack{}, &models.SnapRisk{}, &models.SnapUpload{}, &models.SnapRevision{})
	if err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	repo := NewSnapsRepository(db)

	t.Run("Get risks for existing track", func(t *testing.T) {
		name := "test-snap"
		size := uint64(1024)
		accountId := uint(1)

		snapEntry, err := repo.AddSnap(name, size, accountId)
		assert.NoError(t, err)

		tracks, err := repo.GetTracks(snapEntry.ID)
		assert.NoError(t, err)
		assert.NotNil(t, tracks)
		assert.Len(t, *tracks, 1)

		trackId := (*tracks)[0].ID
		risks, err := repo.GetRisks(trackId)
		assert.NoError(t, err)
		assert.NotNil(t, risks)
		assert.Len(t, *risks, 4) // Default risks "stable", "candidate", "beta", "edge" should be created
	})

	t.Run("Get risks for non-existent track", func(t *testing.T) {
		risks, err := repo.GetRisks(999)
		assert.Error(t, err)
		assert.Nil(t, risks)
		assert.Equal(t, "unknown error encountered", err.Error())
	})
}
func TestGetRevision(t *testing.T) {
	// Setup in-memory database
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	// Migrate the schema
	err = db.AutoMigrate(&models.SnapEntry{}, &models.SnapTrack{}, &models.SnapRisk{}, &models.SnapUpload{}, &models.SnapRevision{})
	if err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	repo := NewSnapsRepository(db)

	t.Run("Get revision by ID", func(t *testing.T) {
		name := "test-snap"
		size := uint64(1024)
		accountId := uint(1)

		snapEntry, err := repo.AddSnap(name, size, accountId)
		assert.NoError(t, err)

		revision := repo.addRevision(*snapEntry, size)

		fetchedRevision, err := repo.GetRevision(revision.ID)
		assert.NoError(t, err)
		assert.NotNil(t, fetchedRevision)
		assert.Equal(t, revision.ID, fetchedRevision.ID)
	})

	t.Run("Get revision by ID that does not exist", func(t *testing.T) {
		revision, err := repo.GetRevision(999)
		assert.Error(t, err)
		assert.Nil(t, revision)
		assert.Equal(t, "unknown error encountered", err.Error())
	})
}
func TestSetChannelRevision(t *testing.T) {
	// Setup in-memory database
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	// Migrate the schema
	err = db.AutoMigrate(&models.SnapEntry{}, &models.SnapTrack{}, &models.SnapRisk{}, &models.SnapUpload{}, &models.SnapRevision{})
	if err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	repo := NewSnapsRepository(db)

	t.Run("Set channel revision successfully", func(t *testing.T) {
		name := "test-snap"
		size := uint64(1024)
		accountId := uint(1)

		snapEntry, err := repo.AddSnap(name, size, accountId)
		assert.NoError(t, err)

		trackName := "latest"
		riskName := "stable"
		revision := repo.addRevision(*snapEntry, size)

		track, err := repo.SetChannelRevision(trackName, riskName, revision.ID, snapEntry.ID)
		assert.NoError(t, err)
		assert.NotNil(t, track)
		assert.Equal(t, trackName, track.Name)
	})

	t.Run("Set channel revision with non-existent track", func(t *testing.T) {
		name := "test-snap-2"
		size := uint64(1024)
		accountId := uint(1)

		snapEntry, err := repo.AddSnap(name, size, accountId)
		assert.NoError(t, err)

		trackName := "non-existent-track"
		riskName := "stable"
		revision := repo.addRevision(*snapEntry, size)

		track, err := repo.SetChannelRevision(trackName, riskName, revision.ID, snapEntry.ID)
		assert.Error(t, err)
		assert.Nil(t, track)
		assert.Equal(t, "track does not exist for snap", err.Error())
	})

	t.Run("Set channel revision with non-existent risk", func(t *testing.T) {
		name := "test-snap-3"
		size := uint64(1024)
		accountId := uint(1)

		snapEntry, err := repo.AddSnap(name, size, accountId)
		assert.NoError(t, err)

		trackName := "latest"
		riskName := "non-existent-risk"
		revision := repo.addRevision(*snapEntry, size)

		track, err := repo.SetChannelRevision(trackName, riskName, revision.ID, snapEntry.ID)
		assert.Error(t, err)
		assert.Nil(t, track)
		assert.Equal(t, "risk does not exist for track", err.Error())
	})

	t.Run("Set channel revision with non-existent revision", func(t *testing.T) {
		name := "test-snap-4"
		size := uint64(1024)
		accountId := uint(1)

		snapEntry, err := repo.AddSnap(name, size, accountId)
		assert.NoError(t, err)

		trackName := "latest"
		riskName := "stable"
		nonExistentRevisionId := uint(999)

		track, err := repo.SetChannelRevision(trackName, riskName, nonExistentRevisionId, snapEntry.ID)
		assert.Error(t, err)
		assert.Nil(t, track)
		assert.Equal(t, "unknown error encountered", err.Error())
	})
}
func TestGetRevisionBySHA(t *testing.T) {
	// Setup in-memory database
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	// Migrate the schema
	err = db.AutoMigrate(&models.SnapEntry{}, &models.SnapTrack{}, &models.SnapRisk{}, &models.SnapUpload{}, &models.SnapRevision{})
	if err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	repo := NewSnapsRepository(db)

	t.Run("Get revision by SHA3_384", func(t *testing.T) {
		name := "test-snap"
		size := uint64(1024)
		accountId := uint(1)
		SHA3_384 := "test-sha3-384"

		snapEntry, err := repo.AddSnap(name, size, accountId)
		assert.NoError(t, err)

		revision := repo.addRevision(*snapEntry, size)
		revision.SHA3_384 = SHA3_384
		db.Save(&revision)

		fetchedRevision, err := repo.GetRevisionBySHA(SHA3_384, false)
		assert.NoError(t, err)
		assert.NotNil(t, fetchedRevision)
		assert.Equal(t, SHA3_384, fetchedRevision.SHA3_384)
	})

	t.Run("Get revision by encoded SHA3_384", func(t *testing.T) {
		name := "test-snap-encoded"
		size := uint64(1024)
		accountId := uint(1)
		SHA3384Encoded := "test-sha3-384-encoded"

		snapEntry, err := repo.AddSnap(name, size, accountId)
		assert.NoError(t, err)

		revision := repo.addRevision(*snapEntry, size)
		revision.SHA3384Encoded = SHA3384Encoded
		db.Save(&revision)

		fetchedRevision, err := repo.GetRevisionBySHA(SHA3384Encoded, true)
		assert.NoError(t, err)
		assert.NotNil(t, fetchedRevision)
		assert.Equal(t, SHA3384Encoded, fetchedRevision.SHA3384Encoded)
	})

	t.Run("Get revision by non-existent SHA3_384", func(t *testing.T) {
		SHA3_384 := "non-existent-sha3-384"

		fetchedRevision, err := repo.GetRevisionBySHA(SHA3_384, false)
		assert.NoError(t, err)
		assert.Nil(t, fetchedRevision)
	})

	t.Run("Get revision by non-existent encoded SHA3_384", func(t *testing.T) {
		SHA3384Encoded := "non-existent-sha3-384-encoded"

		fetchedRevision, err := repo.GetRevisionBySHA(SHA3384Encoded, true)
		assert.NoError(t, err)
		assert.Nil(t, fetchedRevision)
	})
}
func TestGetUpload(t *testing.T) {
	// Setup in-memory database
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	// Migrate the schema
	err = db.AutoMigrate(&models.SnapEntry{}, &models.SnapTrack{}, &models.SnapRisk{}, &models.SnapUpload{}, &models.SnapRevision{})
	if err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	repo := NewSnapsRepository(db)

	t.Run("Get upload by upDownId", func(t *testing.T) {
		name := "test-snap"
		size := uint64(1024)
		accountId := uint(1)
		upDownId := "test-updown-id"

		_, err := repo.AddSnap(name, size, accountId)
		assert.NoError(t, err)

		_, err = repo.AddUpload(name, upDownId, uint(size), []string{"latest/stable"})
		assert.NoError(t, err)

		snapUpload, err := repo.GetUpload(upDownId)
		assert.NoError(t, err)
		assert.NotNil(t, snapUpload)
		assert.Equal(t, upDownId, snapUpload.UpDownID)
	})

	t.Run("Get upload by non-existent upDownId", func(t *testing.T) {
		snapUpload, err := repo.GetUpload("non-existent-updown-id")
		assert.Error(t, err)
		assert.Nil(t, snapUpload)
		assert.Equal(t, "not found", err.Error())
	})
}

func TestUpdateRevision(t *testing.T) {
	// Setup in-memory database
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	// Migrate the schema
	err = db.AutoMigrate(&models.SnapEntry{}, &models.SnapTrack{}, &models.SnapRisk{}, &models.SnapUpload{}, &models.SnapRevision{})
	if err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	// repo := NewSnapsRepository(db)

	// t.Run("Update revision successfully", func(t *testing.T) {
	// 	name := "test-snap"
	// 	size := uint64(1024)
	// 	accountId := uint(1)

	// 	snapEntry, err := repo.AddSnap(name, size, accountId)
	// 	assert.NoError(t, err)

	// 	revision := repo.addRevision(*snapEntry, size)
	// 	revisionBytes := []byte("test-revision-bytes")

	// 	updatedRevision, err := repo.UpdateRevision(revision, &revisionBytes)
	// 	assert.NoError(t, err)
	// 	assert.NotNil(t, updatedRevision)
	// 	assert.Equal(t, revision.ID, updatedRevision.ID)
	// })

	// t.Run("Update revision with error", func(t *testing.T) {
	// 	revision := models.SnapRevision{}
	// 	revisionBytes := []byte("test-revision-bytes")

	// 	updatedRevision, err := repo.UpdateRevision(&revision, &revisionBytes)
	// 	assert.Error(t, err)
	// 	assert.Nil(t, updatedRevision)
	// })
}
