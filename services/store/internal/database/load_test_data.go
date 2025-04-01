package database

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/google/uuid"
	"github.com/idlab-discover/kebeng/services/store/internal/models"
	"github.com/idlab-discover/kebeng/services/store/internal/repositories"
	"github.com/sirupsen/logrus"
)

type TestData struct {
	SnapEntries   []models.SnapEntry    `json:"snap_entries"`
	SnapTracks    []models.SnapTrack    `json:"snap_tracks"`
	SnapChannels  []models.SnapChannel  `json:"snap_channels"`
	SnapBranches  []models.SnapBranch   `json:"snap_branches"`
	SnapRevisions []models.SnapRevision `json:"snap_revisions"`
	SnapComments  []models.SnapComment  `json:"snap_comments"`
}

// TODO: add channel and track and then remove computing path so that
// the path can be reconstructed from the database
func LoadTestData(filePath string, repo repositories.ISnapsRepository) ([]string, error) {
	logrus.Info("Inserting test data")

	// Check if file exists and read its content.
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		logrus.Warnf("Test data file does not exist: %s", filePath)
		return nil, nil
	}
	file, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open test data file: %v", err)
	}
	if len(file) == 0 {
		logrus.Info("Test data file is empty")
		return nil, nil
	}

	// Unmarshal JSON test data into our TestData struct.
	testData := &TestData{}
	if err = json.Unmarshal(file, testData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal test data: %v", err)
	}

	// Mapping from the original test entry IDs to the new IDs returned from the repository.
	snapIDMap := make(map[uuid.UUID]uuid.UUID)

	// --- Insert SnapEntries ---
	for _, entry := range testData.SnapEntries {
		isPrivate := false
		if entry.Private != nil {
			isPrivate = *entry.Private
		}
		// Use RegisterSnap to insert the snap entry.
		// (Note: you may later update this to include the account id.)
		registeredSnap, cerr := repo.RegisterSnap(entry.Name, isPrivate)
		if cerr != nil {
			return nil, fmt.Errorf("failed to register snap (%s): %v", entry.Name, cerr)
		}
		logrus.Infof("Registered SnapEntry: %+v", registeredSnap)
		// Update mapping: original ID -> new generated ID.
		snapIDMap[entry.ID] = registeredSnap.ID
	}

	logrus.Infof("Snap ID mapping: %+v", snapIDMap)

	// --- Insert SnapRevisions ---
	// For each revision, update its SnapEntryID using the mapping.
	for _, rev := range testData.SnapRevisions {
		newEntryID, ok := snapIDMap[rev.SnapEntryID]
		if !ok {
			logrus.Warnf("Checking for entryId: %s, Snap ID mapping: %+v", rev.SnapEntryID, snapIDMap)
			return nil, fmt.Errorf("no registered snap entry found for revision (%s)", rev.ID)
		}
		// Retrieve the snap entry so that we can call AddRevision.
		snapEntry, cerr := repo.GetEntryById(newEntryID, nil)
		if cerr != nil {
			return nil, fmt.Errorf("failed to get snap entry for revision (%s): %v", rev.ID, cerr)
		}

		// Use the size from the revision, or 0 if nil.
		var size uint64 = 0
		if rev.Size != nil {
			size = *rev.Size
		}

		_, cerr = repo.AddRevision(rev.SnapEntryID, rev.SnapTrackID, rev.SnapChannelID, size)
		if cerr != nil {
			return nil, fmt.Errorf("failed to add revision for snap (%s): %v", snapEntry.ID, cerr)
		}
		logrus.Infof("Inserted Revision for snap entry: %s", snapEntry.ID)
	}

	// --- Insert SnapTracks, SnapChannels, SnapBranches, and SnapComments ---
	// (Not implemented; we assume the test data for these is present for computing file paths.)

	// Now compute file paths for each revision.
	// For each revision in testData.SnapRevisions, find the corresponding track and channel.
	var snapPaths []string
	for _, rev := range testData.SnapRevisions {
		// Use the original entry_id from test data (this should match one of the testData.snap_entries)
		origEntryID := rev.SnapEntryID

		// Look up the snap entry in testData.
		var snapEntry *models.SnapEntry
		for _, e := range testData.SnapEntries {
			if e.ID == origEntryID {
				snapEntry = &e
				break
			}
		}
		if snapEntry == nil {
			return nil, fmt.Errorf("no snap entry found in test data for revision (%s)", rev.ID)
		}

		// Find a track for this entry.
		var trackName string
		for _, t := range testData.SnapTracks {
			if t.SnapEntryID == origEntryID {
				trackName = t.Name
				break
			}
		}
		if trackName == "" {
			return nil, fmt.Errorf("no snap track found for entry (%s)", origEntryID)
		}

		// Find a channel for this entry and revision.
		var channelName string
		for _, c := range testData.SnapChannels {
			// Assume the channel is for this snap entry and its track_id matches the tracks's id.
			if c.SnapEntryID == origEntryID && c.SnapTrackID == rev.SnapTrackID {
				channelName = c.Name
				break
			}
		}
		if channelName == "" {
			return nil, fmt.Errorf("no snap channel found for entry (%s) and revision (%s)", origEntryID, rev.ID)
		}

		// Build the path using the format:
		// snaps/<snap_entry_name>/<track>/<channel>/<snap_revision.snap_name>_<sequence_number>.snap
		// We assume that rev.SnapName is set (e.g. "test.snap") and rev.SequenceNumber is provided.
		if rev.SequenceNumber == nil {
			return nil, fmt.Errorf("revision (%s) missing sequence_number", rev.ID)
		}
		baseName := snapEntry.Name
		path := fmt.Sprintf("%s/%s/%s/%s_%d.snap",
			snapEntry.Name,
			trackName,
			channelName,
			baseName,
			*rev.SequenceNumber,
		)
		snapPaths = append(snapPaths, path)
		logrus.Infof("Computed snap path: %s", path)
	}

	return snapPaths, nil
}
