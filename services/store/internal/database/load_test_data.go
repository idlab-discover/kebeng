package database

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/google/uuid"
	"github.com/idlab-discover/kebeng/services/store/internal/models"
	"github.com/idlab-discover/kebeng/services/store/internal/repository"
	"github.com/sirupsen/logrus"
)

type TestData struct {
	Entries   []models.SnapEntry    `json:"snap_entries"`
	Tracks    []models.SnapTrack    `json:"snap_tracks"`
	Channels  []models.SnapChannel  `json:"snap_channels"`
	Branches  []models.SnapBranch   `json:"snap_branches"`
	Revisions []models.SnapRevision `json:"snap_revisions"`
	Comments  []models.SnapComment  `json:"snap_comments"`
}

// TODO: add channel and track and then remove computing path so that
// the path can be reconstructed from the database
func LoadTestData(filePath string, repo repository.ISnapsRepository) error {
	logrus.Info("Inserting test data")

	// Check if file exists and read its content.
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		logrus.Warnf("Test data file does not exist: %s", filePath)
		return nil
	}
	file, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to open test data file: %v", err)
	}
	if len(file) == 0 {
		logrus.Info("Test data file is empty")
		return nil
	}

	// Unmarshal JSON test data into our TestData struct.
	testData := &TestData{}
	if err = json.Unmarshal(file, testData); err != nil {
		return fmt.Errorf("failed to unmarshal test data: %v", err)
	}

	idMap := make(map[uuid.UUID]uuid.UUID)

	// --- Insert Entries ---
	for _, entry := range testData.Entries {
		isPrivate := false
		if entry.Private != nil {
			isPrivate = *entry.Private
		}
		// Use RegisterSnap to insert the snap entry.
		// (Note: you may later update this to include the account id.)
		registeredSnap, cerr := repo.RegisterSnap(entry.Name, isPrivate, *entry.Store, entry.AccountID)
		if cerr != nil {
			return fmt.Errorf("failed to register snap (%s): %v", entry.Name, cerr)
		}
		logrus.Infof("Registered SnapEntry: %+v", registeredSnap)
		// Update mapping: original ID -> new generated ID.
		idMap[entry.ID] = registeredSnap.ID
	}

	for _, track := range testData.Tracks {
		// Use RegisterTrack to insert the snap track.
		newEntryID, ok := idMap[track.SnapEntryID]
		if !ok {
			logrus.Warnf("Checking for entryId: %s, Snap ID mapping: %+v", track.SnapEntryID, idMap)
			return fmt.Errorf("no registered snap entry found for track (%s)", track.ID)
		}
		registeredTrack, cerr := repo.AddTrack(newEntryID, track.Name)
		if cerr != nil {
			return fmt.Errorf("failed to register track (%s): %v", track.Name, cerr)
		}
		idMap[track.ID] = registeredTrack.ID
	}

	for _, channel := range testData.Channels {
		newEntryID, ok := idMap[channel.SnapEntryID]
		if !ok {
			logrus.Warnf("Checking for entryId: %s, Snap ID mapping: %+v", channel.SnapEntryID, idMap)
			return fmt.Errorf("no registered snap entry found for channel (%s)", channel.ID)
		}
		newTrackId, ok2 := idMap[channel.SnapTrackID]
		if !ok2 {
			logrus.Warnf("Checking for trackId: %s, Snap ID mapping: %+v", channel.SnapTrackID, idMap)
			return fmt.Errorf("no registered snap track found for channel (%s)", channel.ID)
		}

		registeredChannel, cerr := repo.AddChannel(newEntryID, newTrackId, channel.Name)
		if cerr != nil {
			return fmt.Errorf("failed to register channel (%s): %v", channel.Name, cerr)
		}
		logrus.Infof("Registered SnapChannel: %+v", channel)
		idMap[channel.ID] = registeredChannel.ID
	}

	for _, rev := range testData.Revisions {
		newEntryID, ok := idMap[rev.SnapEntryID]
		if !ok {
			logrus.Warnf("Checking for entryId: %s, Snap ID mapping: %+v", rev.SnapEntryID, idMap)
			return fmt.Errorf("no registered snap entry found for revision (%s)", rev.ID)
		}
		newTrackId, ok2 := idMap[rev.SnapTrackID]
		if !ok2 {
			logrus.Warnf("Checking for trackId: %s, Snap ID mapping: %+v", rev.SnapTrackID, idMap)
			return fmt.Errorf("no registered snap track found for revision (%s)", rev.ID)
		}
		newChannelId, ok3 := idMap[rev.SnapChannelID]
		if !ok3 {
			logrus.Warnf("Checking for channelId: %s, Snap ID mapping: %+v", rev.SnapChannelID, idMap)
			return fmt.Errorf("no registered snap channel found for revision (%s)", rev.ID)
		}

		var size uint64 = 0
		if rev.Size != nil {
			size = *rev.Size
		}

		registeredRevision, cerr := repo.AddRevision(newEntryID, newTrackId, newChannelId, size, *rev.SequenceNumber)
		if cerr != nil {
			return fmt.Errorf("failed to add revision for snap (%s): %v", newEntryID, cerr)
		}
		idMap[rev.ID] = registeredRevision.ID
	}

	return nil
}
