package repositories

import (
	"fmt"
	"slices"
	"strings"

	"github.com/idlab-discover/kebeng/services/store/internal/snap"
	"github.com/lib/pq"
	"github.com/sirupsen/logrus"

	"github.com/google/uuid"

	"github.com/idlab-discover/kebeng/services/store/internal/models"
	"github.com/jmoiron/sqlx"

	cerror "github.com/idlab-discover/kebeng/common/cerror"
)

type ISnapsRepository interface {
	// CREATE
	AddRevision(snapEntry models.SnapEntry, size uint64) (*models.SnapRevision, *cerror.CustomError)
	AddSnap(name string, size uint64, accountId uuid.UUID) (*models.SnapEntry, *cerror.CustomError)
	AddUpload(snapName string, upDownId string, size uint, channels []string) (*models.SnapUpload, *cerror.CustomError)
	RegisterSnap(snapName string, isPrivate bool) (*models.SnapEntry, *cerror.CustomError)

	// READ
	GetAllSnapEntries() (*[]models.SnapEntry, *cerror.CustomError)
	GetChannelsByTrackId(trackId uuid.UUID) ([]*models.SnapChannel, *cerror.CustomError)
	GetCommentsByEntryId(entryId uuid.UUID) ([]*models.SnapComment, *cerror.CustomError)
	GetEntriesByAccountId(accountId uuid.UUID, preloadAssociations []string) ([]*models.SnapEntry, *cerror.CustomError)
	GetEntryById(id uuid.UUID, preloadAssociations []string) (*models.SnapEntry, *cerror.CustomError)
	GetEntryByName(name string, preloadAssociations []string) (*models.SnapEntry, *cerror.CustomError)
	GetRevisionByChannel(channel string, snapName string) (*models.SnapRevision, *cerror.CustomError)
	GetRevisionByEntryId(entryId uuid.UUID) (*models.SnapRevision, *cerror.CustomError)
	GetRevisionById(id string) (*models.SnapRevision, *cerror.CustomError)
	GetRevisionByNameAndSequence(name string, sequence uint) (*models.SnapRevision, *cerror.CustomError)
	GetRevisionBySHA(SHA3_384 string, encoded bool) (*models.SnapRevision, *cerror.CustomError)
	GetSections() (*[]string, *cerror.CustomError)
	GetTracksBySnapId(snapId uuid.UUID) ([]*models.SnapTrack, *cerror.CustomError)
	GetUploadByUpDownId(upDownId string) (*models.SnapUpload, *cerror.CustomError)
	GetUploadsByEntryId(d uuid.UUID) ([]*models.SnapUpload, *cerror.CustomError)

	// UPDATE
	ReleaseSnap(channels []string, snapEntryId uuid.UUID, revisionId uuid.UUID) *cerror.CustomError
	SetChannelRevision(trackName string, channelName string, revisionId uint, snapId uuid.UUID) (*models.SnapTrack, *cerror.CustomError)
	UpdateRevision(revision *models.SnapRevision, revisionBytes *[]byte) (*models.SnapRevision, *cerror.CustomError)
}

type SnapsRepository struct {
	db *sqlx.DB
}

func NewSnapsRepository(db *sqlx.DB) *SnapsRepository {
	return &SnapsRepository{db: db}
}

// ============ CREATE =============

func (sp *SnapsRepository) AddRevision(snapEntry *models.SnapEntry, size uint64) (*models.SnapRevision, *cerror.CustomError) {
	// TODO: fix the need for an empty revision
	// TODO: add build_assertion_filename if an assertion exists -> doesn't get checked in official snap store either
	snapRevision := models.SnapRevision{
		SnapName:    &snapEntry.Name,
		SnapEntryID: snapEntry.ID,
		SHA3_384:    nil,
		Size:        &size,
	}

	query := `
		INSERT INTO snap_revisions (snap_name, snap_entry_id, sha3_384, size)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`
	err := sp.db.Get(&snapRevision.ID, query, snapRevision.SnapName, snapRevision.SnapEntryID, snapRevision.SHA3_384, snapRevision.Size)
	if err != nil {
		logrus.Error(err)
		return nil, cerror.ConvertError(err)
	}

	return &snapRevision, nil
}

// Not sure if this is needed
// Used when a new snap gets uploaded for the first time (=registering a snap)

// AddSnap registers a new snap with the given name, size, and accountId.
// It ensures the snap does not already exist, creates a new SnapEntry,
// adds an initial upload, and sets up default tracks and channels.
// func (sp *SnapsRepository) AddSnap(name string, size uint64, accountId uuid.UUID) (*models.SnapEntry, *cerror.CustomError) {
// 	existingSnap, err := sp.GetEntryByName(name, nil)
// 	if err != nil {
// 		// Already logged in GetEntryByName
// 		if err.GetCode() != cerror.ResourceNotFound {
// 			return nil, err
// 		}
// 	}

// 	// if the snap already exists, return an *cerror.CustomError
// 	if existingSnap != nil {
// 		return nil, cerror.NewCustomError(cerror.AlreadyRegistered, fmt.Sprintf("snap with name '%s' already exists", name))
// 	}

// 	// when registering a snap, not finding one is what you want
// 	var newSnapEntry models.SnapEntry
// 	newSnapEntry.Name = name
// 	newSnapEntry.AccountID = accountId
// 	typeStr := "app"
// 	newSnapEntry.Type = &typeStr
// 	//newSnapEntry.Confinement = "strict"
// 	//newSnapEntry.Base = "core18" // default base

// 	// snap_entries table contains snaps with unique names (doesn't keep track of revisions or channels)
// 	query := `
// 		INSERT INTO snap_entries (name, account_id, type)
// 		VALUES ($1, $2, $3)
// 		RETURNING id
// 	`
// 	err2 := sp.db.Get(&newSnapEntry.ID, query, name, accountId, "app")
// 	if err2 != nil {
// 		logrus.Error(err2)
// 		return nil, cerror.ConvertError(err2)
// 	}
//
// 	// snap_uploads table contains channels where the snap is uploaded
// 	sp.AddUpload(name, newSnapEntry.ID.String(), uint(size), []string{"latest/stable"})

// 	// For now when we register a snap we are going to create the default tracks/channels
// 	track := models.SnapTrack{
// 		Name:        "latest", // first upload of a snap is always the current latest
// 		SnapEntryID: newSnapEntry.ID,
// 	}
// 	query = `
// 		INSERT INTO snap_tracks (name, snap_entry_id)
// 		VALUES ($1, $2)
// 		RETURNING id
// 	`
// 	err2 = sp.db.Get(&track.ID, query, track.Name, newSnapEntry.ID)
// 	if err2 != nil {
// 		logrus.Error(err2)
// 		return nil, cerror.ConvertError(err2)
// 	}

// 	newRevision, _ := sp.AddRevision(&newSnapEntry, size)

// 	sp.addChannels(newSnapEntry, *newRevision, track.ID)

// 	return &newSnapEntry, nil

// }

func (sp *SnapsRepository) AddUpload(snapName string, upDownId string, fileSize uint, channels []string) (*models.SnapUpload, *cerror.CustomError) {
	var snap models.SnapEntry
	query := `
		SELECT id
		FROM snap_entries
		WHERE name = $1
	`
	err := sp.db.Get(&snap.ID, query, snapName)
	if err != nil {
		logrus.Error(err)
		return nil, cerror.ConvertError(err, fmt.Sprintf("resource not found: snap with name = '%s'", snapName))
	}

	snapUpload := models.SnapUpload{
		UpDownID:    upDownId,
		Filesize:    fileSize,
		SnapEntryID: snap.ID,
	}

	logrus.Infof("Uploading: %s", snapName)

	// TODO: fix lazy; this should be converted to a table so that the channels can be stored separately or maybe redis
	if len(channels) > 0 {
		snapUpload.Channels = pq.StringArray(channels)
	}

	query = `
		INSERT INTO snap_uploads (up_down_id, filesize, channels, snap_entry_id)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`
	err = sp.db.Get(&snapUpload.ID, query, upDownId, fileSize, snapUpload.Channels, snap.ID)
	if err != nil {
		logrus.Error(err)
		return nil, cerror.ConvertError(err, fmt.Sprintf("resource not found: snap with id = '%s'", snap.ID.String()))
	}

	return &snapUpload, nil
}

func (sp *SnapsRepository) RegisterSnap(snapName string, isPrivate bool) (*models.SnapEntry, *cerror.CustomError) {
	snapEntry := models.SnapEntry{
		Name:    snapName,
		Private: &isPrivate,
	}

	query := `
		INSERT INTO snap_entries (name, private)
		VALUES ($1, $2)
		RETURNING id
	`
	err := sp.db.Get(&snapEntry.ID, query, snapName, isPrivate)
	if err != nil {
		logrus.Error(err)
		return nil, cerror.ConvertError(err)
	}

	return &snapEntry, nil
}

// ============ READ =============

func (sp *SnapsRepository) GetAllSnapEntries() (*[]models.SnapEntry, *cerror.CustomError) {
	var snaps []models.SnapEntry
	query := `
		SELECT *
		FROM snap_entries
	`
	err := sp.db.Select(&snaps, query)
	if err != nil {
		logrus.Error(err)
		return nil, cerror.ConvertError(err)
	}

	if len(snaps) == 0 {
		return nil, cerror.NewCustomError(cerror.ResourceNotFound, "resource not found: no snaps")
	}

	return &snaps, nil
}

func (sp *SnapsRepository) GetChannelsByTrackId(trackId uuid.UUID) ([]*models.SnapChannel, *cerror.CustomError) {
	var channels []*models.SnapChannel
	query := `
		SELECT *
		FROM snap_channels
		WHERE snap_track_id = $1
	`
	err := sp.db.Select(&channels, query, trackId)
	if err != nil {
		logrus.Error(err)
		return nil, cerror.ConvertError(err, fmt.Sprintf("resource not found: channels for track with id = '%s'", trackId.String()))
	}

	// manual check for empty result because db.Select doesn't return an error for empty results
	if len(channels) == 0 {
		return nil, cerror.NewCustomError(cerror.ResourceNotFound, fmt.Sprintf("resource not found: channels for track with id = '%s'", trackId.String()))
	}

	return channels, nil
}

func (sp *SnapsRepository) GetCommentsByEntryId(entryId uuid.UUID) ([]*models.SnapComment, *cerror.CustomError) {
	var comments []*models.SnapComment
	query := `
			SELECT *
			FROM snap_comments
			WHERE snap_entry_id = $1
		`
	err := sp.db.Select(&comments, query, entryId)
	if err != nil {
		logrus.Error(err)
		return nil, cerror.ConvertError(err, fmt.Sprintf("resource not found: comments for snap with id = '%s'", entryId.String()))
	}

	// manual check for empty result because db.Select doesn't return an error for empty results
	if len(comments) == 0 {
		return nil, cerror.NewCustomError(cerror.ResourceNotFound, fmt.Sprintf("resource not found: comments for snap with id = '%s'", entryId.String()))
	}

	return comments, nil
}

func (sp *SnapsRepository) GetEntriesByAccountId(accountId uuid.UUID, preloadAssociations []string) ([]*models.SnapEntry, *cerror.CustomError) {
	var entries []*models.SnapEntry
	query := `
            SELECT * 
            FROM snap_entries 
            WHERE account_id = $1
        `
	err := sp.db.Select(&entries, query, accountId)
	if err != nil {
		logrus.Error(err)
		return nil, cerror.ConvertError(err, fmt.Sprintf("resource not found: snaps for account with id = '%s'", accountId.String()))
	}

	// manual check for empty result because db.Select doesn't return an error for empty results
	if len(entries) == 0 {
		return nil, cerror.NewCustomError(cerror.ResourceNotFound, fmt.Sprintf("resource not found: snaps for account with id = '%s'", accountId.String()))
	}

	for _, entry := range entries {

		cerr := sp.getPreloadAssociations(entry, &preloadAssociations)
		if cerr != nil {
			logrus.Error("failed to preload associations")
			return nil, cerr
		}
	}

	return entries, nil
}

func (sp *SnapsRepository) GetEntryById(Id uuid.UUID, preloadAssociations []string) (*models.SnapEntry, *cerror.CustomError) {
	var snapEntry models.SnapEntry
	query := `
		SELECT *
		FROM snap_entries
		WHERE id = $1
	`
	err := sp.db.Get(&snapEntry, query, Id)
	if err != nil {
		logrus.Error(err)
		return nil, cerror.ConvertError(err, fmt.Sprintf("resource not found: snap with id = '%s'", Id.String()))
	}

	cerr := sp.getPreloadAssociations(&snapEntry, &preloadAssociations)
	if cerr != nil {
		logrus.Error("failed to preload associations")
		return nil, cerr
	}

	return &snapEntry, nil
}

func (sp *SnapsRepository) GetEntryByName(name string, preloadAssociations []string) (*models.SnapEntry, *cerror.CustomError) {
	if preloadAssociations == nil {
		preloadAssociations = []string{}
	}

	var snapEntry models.SnapEntry
	query := `
		SELECT *
		FROM snap_entries
		WHERE name = $1
	`
	err := sp.db.Get(&snapEntry, query, name)
	if err != nil {
		logrus.Errorf("FUNCTION GetEntryByName: %s", err)
		return nil, cerror.ConvertError(err, fmt.Sprintf("resource not found: snap with name = '%s'", name))
	}

	cerr := sp.getPreloadAssociations(&snapEntry, &preloadAssociations)
	if cerr != nil {
		logrus.Error("failed to preload associations")
		return nil, cerr
	}

	return &snapEntry, nil
}

func (sp *SnapsRepository) GetRevisionByChannel(channel string, snapName string) (*models.SnapRevision, *cerror.CustomError) {
	snapEntry, err1 := sp.GetEntryByName(snapName, nil)
	if err1 != nil {
		logrus.Error(err1.GetMessage())
		return nil, err1
	}

	channelParts := strings.Split(channel, "/")
	var track string
	var channelname string
	if len(channelParts) == 1 {
		if channelParts[0] == "beta" || channelParts[0] == "edge" || channelParts[0] == "stable" || channelParts[0] == "candidate" {
			track = "latest"
			channelname = channelParts[0]
		} else {
			track = channelParts[0]
			channelname = "stable"
		}
	} else if len(channelParts) == 2 {
		track = channelParts[0]
		channelname = channelParts[1]
	} else {
		return nil, cerror.NewCustomError(cerror.NotImplemented, "branches not yet supported for channels")
	}

	var snapTrack models.SnapTrack
	query := `
			SELECT id
			FROM snap_tracks
			WHERE snap_entry_id = $1 AND name = $2
		`
	err := sp.db.Get(&snapTrack, query, snapEntry.ID, track)
	if err != nil {
		logrus.Error(err)
		return nil, cerror.ConvertError(err, fmt.Sprintf("resource not found: track '%s' for snap with name = '%s'", track, snapName))
	}

	var snapChannel models.SnapChannel
	query = `
			SELECT revision_id
			FROM snap_channels
			WHERE snap_entry_id = $1 AND snap_track_id = $2 AND name = $3
		`
	err = sp.db.Get(&snapChannel, query, snapEntry.ID, snapTrack.ID, channelname)
	if err != nil {
		logrus.Error(err)
		return nil, cerror.ConvertError(err, fmt.Sprintf("resource not found: channel '%s' for snap with name = '%s'", channel, snapName))
	}

	var snapRevision models.SnapRevision
	query = `
			SELECT *
			FROM snap_revisions
			WHERE id = $1
		`
	err = sp.db.Get(&snapRevision, query, snapChannel.RevisionID)
	if err != nil {
		logrus.Error(err)
		return nil, cerror.ConvertError(err, fmt.Sprintf("resource not found: revision with id = '%s'", snapChannel.RevisionID))
	}

	return &snapRevision, nil
}

func (sp *SnapsRepository) GetRevisionsByEntryId(entryId uuid.UUID) ([]*models.SnapRevision, *cerror.CustomError) {
	var revisions []*models.SnapRevision
	query := `
			SELECT *
			FROM snap_revisions
			WHERE snap_entry_id = $1
		`
	err := sp.db.Select(&revisions, query, entryId)
	if err != nil {
		logrus.Error(err)
		return nil, cerror.ConvertError(err, fmt.Sprintf("resource not found: revisions for snap with id = '%s'", entryId.String()))
	}

	// manual check for empty result because db.Select doesn't return an error for empty results
	if len(revisions) == 0 {
		return nil, cerror.NewCustomError(cerror.ResourceNotFound, fmt.Sprintf("resource not found: revisions for snap with id = '%s'", entryId.String()))
	}

	return revisions, nil
}

func (sp *SnapsRepository) GetRevisionById(id uuid.UUID) (*models.SnapRevision, *cerror.CustomError) {
	var revision models.SnapRevision
	query := `
		SELECT *
		FROM snap_revisions
		WHERE id = $1
	`
	err := sp.db.Get(&revision, query, id)
	if err != nil {
		logrus.Error(err)
		return nil, cerror.ConvertError(err, fmt.Sprintf("resource not found: revision with id = '%s'", id))
	}

	return &revision, nil
}

func (sp *SnapsRepository) GetRevisionByNameAndSequence(name string, sequence uint) (*models.SnapRevision, *cerror.CustomError) {
	var entry models.SnapEntry
	query := `
		SELECT id
		FROM snap_entries
		WHERE name = $1
	`
	err := sp.db.Get(&entry, query, name)
	if err != nil {
		logrus.Error(err)
		return nil, cerror.ConvertError(err, fmt.Sprintf("resource not found: snap with name = '%s' while searching for revision", name))
	}

	var revision models.SnapRevision
	query = `
		SELECT *
		FROM snap_revisions
		WHERE snap_entry_id = $1 AND sequence_number = $2
	`
	err = sp.db.Get(&revision, query, entry.ID, sequence)
	if err != nil {
		logrus.Error(err)
		return nil, cerror.ConvertError(err, fmt.Sprintf("resource not found: revision with sequence = '%d' for snap with name = '%s'", sequence, name))
	}

	return &revision, nil
}

func (sp *SnapsRepository) GetRevisionBySHA(SHA3_384 string, encoded bool) (*models.SnapRevision, *cerror.CustomError) {
	var revision models.SnapRevision

	if encoded {
		logrus.Tracef("Getting snap revision by encoded sha3_384: %s", SHA3_384)
		query := `
			SELECT *
			FROM snap_revisions
			WHERE sha3_384_encoded = $1
		`
		err := sp.db.Get(&revision, query, SHA3_384)
		if err != nil {
			logrus.Error(err)
			return nil, cerror.ConvertError(err)
		}
	} else {
		logrus.Tracef("Getting snap revision by sha3_384: %s", SHA3_384)
		query := `
			SELECT *
			FROM snap_revisions
			WHERE sha3_384 = $1
		`
		err := sp.db.Get(&revision, query, SHA3_384)
		if err != nil {
			logrus.Error(err)
			return nil, cerror.ConvertError(err)
		}
	}

	return &revision, nil
}

// QUESTION: Think this is for browsing categories? Not sure
func (sp *SnapsRepository) GetSections() (*[]string, *cerror.CustomError) {
	// TODO: add these to the database for real
	sections := []string{
		"general",
	}

	return &sections, nil
}

func (sp *SnapsRepository) GetTracksBySnapId(snapId uuid.UUID) ([]*models.SnapTrack, *cerror.CustomError) {
	var tracks []*models.SnapTrack

	query := `
		SELECT *
		FROM snap_tracks
		WHERE snap_entry_id = $1
	`
	err := sp.db.Select(&tracks, query, snapId)
	if err != nil {
		logrus.Error(err)
		return nil, cerror.ConvertError(err, fmt.Sprintf("resource not found: tracks for snap with id = '%s'", snapId.String()))
	}

	if len(tracks) == 0 {
		return nil, cerror.NewCustomError(cerror.ResourceNotFound, fmt.Sprintf("resource not found: tracks for snap with id = '%s'", snapId.String()))
	}

	return tracks, nil
}

// QUESTION: still not sure what UpDownId is?
func (sp *SnapsRepository) GetUploadByUpDownId(upDownId string) (*models.SnapUpload, *cerror.CustomError) {
	var snapUpload models.SnapUpload
	query := `
		SELECT *
		FROM snap_uploads
		WHERE up_down_id = $1
	`
	err := sp.db.Get(&snapUpload, query, upDownId)
	if err != nil {
		logrus.Error(err)
		return nil, cerror.ConvertError(err, fmt.Sprintf("resource not found: upload with up_down_id = '%s'", upDownId))
	}

	return &snapUpload, nil
}

func (sp *SnapsRepository) GetUploadsByEntryId(d uuid.UUID) ([]*models.SnapUpload, *cerror.CustomError) {
	query := `
			SELECT *
			FROM snap_uploads
			WHERE snap_entry_id = $1
		`
	var uploads []*models.SnapUpload
	err := sp.db.Select(&uploads, query, d)
	if err != nil {
		logrus.Error(err)
		return nil, cerror.ConvertError(err, fmt.Sprintf("resource not found: uploads for snap with id = '%s'", d.String()))
	}

	// manual check for empty result because db.Select doesn't return an error for empty results
	if len(uploads) == 0 {
		return nil, cerror.NewCustomError(cerror.ResourceNotFound, fmt.Sprintf("resource not found: uploads for snap with id = '%s'", d.String()))
	}

	return uploads, nil
}

// ============ UPDATE =============

// ReleaseSnap releases a snap to the specified channels. It supports releasing to multiple
// channels at once. The channels can be specified in the following formats:
//   - "edge": Assumes the track is "latest" (interpreted as "latest/edge").
//   - "latest/edge": Specifies both the track and channel.
//   - "latest/edge/some_branch": Specifies track, channel, and branch (not supported).
//
// For each channel, the function performs the following steps:
//  1. Parses the channel string to extract the track and channel names.
//  2. Retrieves the corresponding track from the database.
//  3. Retrieves the corresponding channel from the database.
//  4. Validates the existence of the specified revision.
//  5. Updates the channel to point to the specified revision.
//
// Parameters:
//   - channels: A slice of strings representing the channels to release the snap to.
//   - snapEntryId: The UUID of the snap entry to be released.
//   - revisionId: The UUID of the revision to be released.
//
// Returns:
//   - A pointer to a CustomError if an error occurs, or nil if the operation is successful.
//
// Notes:
//   - Branches in channels (e.g., "latest/edge/some_branch") are not yet supported and will
//     result in a "NotImplemented" error.
func (sp *SnapsRepository) ReleaseSnap(channels []string, snapEntryId uuid.UUID, revisionId uuid.UUID) *cerror.CustomError {
	var trackForRelease string
	var channelForRelease string
	// It's possible to release a snap to multiple <tracks/channels> at once
	for _, cn := range channels {
		// It's possible this comes in the form:
		//   - single string values like "edge" where the track is assumed to be "latest" there is no branch -> ("latest/<channel>")
		//   - two values "latest/edge" where the channel is proceeded by the track -> ("<track>/<channel>")
		//   - three values "latest/edge/some_branch" -> ("<track>/<channel>/<branch>")
		parts := strings.Split(cn, "/")
		if len(parts) == 1 {
			channelForRelease = parts[0]
			trackForRelease = "latest"
		} else if len(parts) == 2 {
			trackForRelease = parts[0]
			channelForRelease = parts[1]
		} else if len(parts) == 3 {
			return cerror.NewCustomError(cerror.NotImplemented, "branches not yet supported for channels")
		} else {
			return cerror.NewCustomError(cerror.InvalidField, "invalid channel format")
		}

		// get the track for release
		var track models.SnapTrack
		query := `
			SELECT id
			FROM snap_tracks
			WHERE snap_entry_id = $1 AND name = $2
		`
		err := sp.db.Get(&track, query, snapEntryId, trackForRelease)
		if err != nil {
			logrus.Error(err)
			return cerror.ConvertError(err, fmt.Sprintf("resource not found: track '%s' for snap with id = '%s'", trackForRelease, snapEntryId.String()))
		}

		// get the channel for release
		var channel models.SnapChannel
		query = `
			SELECT *
			FROM snap_channels
			WHERE snap_entry_id = $1 AND name = $2 AND snap_track_id = $3
		`
		err = sp.db.Get(&channel, query, snapEntryId, channelForRelease, track.ID)
		if err != nil {
			logrus.Error(err)
			return cerror.ConvertError(err, fmt.Sprintf("resource not found: channel '%s' for snap with id = '%s'", channelForRelease, snapEntryId.String()))
		}

		var revision models.SnapRevision
		query = `
			SELECT id
			FROM snap_revisions
			WHERE id = $1
		`
		err = sp.db.Get(&revision, query, revisionId)
		if err != nil {
			logrus.Error(err)
			return cerror.ConvertError(err, fmt.Sprintf("resource not found: revision with id = '%s'", revisionId.String()))
		}

		query = `
			UPDATE snap_channels
			SET revision_id = $1
			WHERE id = $2
		`
		_, err = sp.db.Exec(query, revision.ID, channel.ID)
		if err != nil {
			logrus.Error(err)
			return cerror.ConvertError(err)
		}
	}

	return nil
}

// SetChannelRevision updates the revision of a specific channel within a track for a snap entry.
// It performs the following steps:
//  1. Retrieves the snap track by its name and snap entry ID.
//  2. Retrieves the snap channel by its name, snap entry ID, and track ID.
//  3. Validates the existence of the specified revision ID.
//  4. Updates the channel's revision ID in the database.
//
// Parameters:
//   - trackName: The name of the track to which the channel belongs.
//   - channelName: The name of the channel to update.
//   - revisionId: The UUID of the revision to set for the channel.
//   - snapEntryId: The UUID of the snap entry associated with the track and channel.
//
// Returns:
//   - *models.SnapTrack: The snap track associated with the updated channel.
//   - *cerror.CustomError: A custom error object if any operation fails.
//
// Errors:
//   - Returns an error if the track, channel, or revision does not exist.
//   - Returns an error if the database operation to update the channel fails.
func (sp *SnapsRepository) SetChannelRevision(trackName string, channelName string, revisionId uuid.UUID, snapEntryId uuid.UUID) (*models.SnapTrack, *cerror.CustomError) {
	// get the snap track by its name and snap entry id
	var track models.SnapTrack
	query := `
		SELECT id
		FROM snap_tracks
		WHERE snap_entry_id = $1 AND name = $2
	`
	err := sp.db.Get(&track, query, snapEntryId, trackName)
	if err != nil {
		logrus.Error(err)
		return nil, cerror.ConvertError(err, fmt.Sprintf("resource not found: track '%s' for snap with id = '%s'", trackName, snapEntryId.String()))
	}

	// get the snap channel inside the track by its name, snap entry id, and track id
	var channel models.SnapChannel
	query = `
		SELECT *
		FROM snap_channels
		WHERE snap_entry_id = $1 AND name = $2 AND snap_track_id = $3
	`
	err = sp.db.Get(&channel, query, snapEntryId, channelName, track.ID)
	if err != nil {
		logrus.Error(err)
		return nil, cerror.ConvertError(err, fmt.Sprintf("resource not found: channel '%s' for snap with id = '%s'", channelName, snapEntryId.String()))
	}

	// FIX: this only gets revision to check if it exists -> maybe we could just skip this and let the db throw an error when it tries updating the channel
	var revision models.SnapRevision
	query = `
		SELECT *
		FROM snap_revisions
		WHERE id = $1
	`
	err = sp.db.Get(&revision, query, revisionId)
	if err != nil {
		logrus.Error(err)
		return nil, cerror.ConvertError(err, fmt.Sprintf("resource not found: revision with id = '%s'", revisionId.String()))
	}

	// update the channel's revision id
	query = `
		UPDATE snap_channels
		SET revision_id = $1
		WHERE id = $2
	`
	_, err = sp.db.Exec(query, revision.ID, channel.ID)
	if err != nil {
		logrus.Error(err)
		return nil, cerror.ConvertError(err, fmt.Sprintf("resource not found: channel '%s' for snap with id = '%s'", channelName, snapEntryId.String()))
	}

	return &track, nil
}

func (sp *SnapsRepository) UpdateRevision(revision *models.SnapRevision, revisionBytes *[]byte) (*models.SnapRevision, *cerror.CustomError) {
	var newRevision models.SnapRevision
	query := `
		UPDATE snap_revisions
		SET snap_name = $1, sha3_384 = $2, sha3_384_encoded = $3, size = $4, sequence_number = $5, architectures = $6, status = $7, version = $8, since = $9
		WHERE id = $10
		RETURNING *
	`
	err := sp.db.Get(&newRevision, query, revision.SnapName, revision.SHA3_384, revision.SHA3_384_Encoded, revision.Size, revision.SequenceNumber, revision.Architectures, revision.Status, revision.Version, revision.Since, revision.ID)
	if err != nil {
		logrus.Error(err)
		return nil, cerror.ConvertError(err, fmt.Sprintf("resource not found: revision with id = '%s'", revision.ID.String()))
	}

	return &newRevision, nil
}

// ============ PRIVATE =============
// ============ HELPER =============

func (sp *SnapsRepository) addChannels(snapEntry models.SnapEntry, snapRevision models.SnapRevision, trackId uuid.UUID) *cerror.CustomError {
	// TODO: fix me
	channels := []string{"stable", "candidate", "beta", "edge"}

	for _, channel := range channels {
		var snapChannel models.SnapChannel
		snapChannel.SnapEntryID = snapEntry.ID
		snapChannel.SnapTrackID = trackId
		snapChannel.Name = channel
		snapChannel.RevisionID = snapRevision.ID

		query := `
			INSERT INTO snap_channels (name, snap_entry_id, snap_track_id, revision_id)
			VALUES ($1, $2, $3, $4)
			RETURNING id
		`
		err := sp.db.Get(&snapChannel.ID, query, snapChannel.Name, snapChannel.SnapEntryID, snapChannel.SnapTrackID, snapChannel.RevisionID)
		if err != nil {
			logrus.Error(err)
			return cerror.ConvertError(err)
		}
	}

	return nil
}

func (sp *SnapsRepository) addDefaultChannels(newSnapEntry models.SnapEntry, newRevision models.SnapRevision, trackId uuid.UUID) *cerror.CustomError {
	return sp.addChannels(newSnapEntry, newRevision, trackId)
}

func (sp *SnapsRepository) updateMeta(metaBytes *[]byte) *cerror.CustomError {
	snapMeta, err2 := snap.GetSnapMetaFromBytes(*metaBytes, "/tmp")
	if err2 != nil {
		logrus.Error(err2)
		return cerror.NewCustomError(err2.Error(), "failed to get snap metadata from bytes")
	}
	logrus.Tracef("snapMeta: %+v", snapMeta)
	var snapEntry models.SnapEntry
	query := `
		SELECT *
		FROM snap_entries
		WHERE name = $1
	`
	err := sp.db.Get(&snapEntry, query, snapMeta.Name)
	if err != nil {
		logrus.Error(err)
		return cerror.ConvertError(err, fmt.Sprintf("resource not found: snap with name = '%s'", snapMeta.Name))
	}

	snapEntry.Type = &snapMeta.Type
	if snapMeta.Type == "" {
		defaultType := "app"
		snapEntry.Type = &defaultType
	}
	if snapMeta.Type != "" {
		snapEntry.Type = &snapMeta.Type
	} else {
		logrus.Warnf("Snap %s had an empty type from its metadata, using default 'app'", snapEntry.Name)
	}
	if snapMeta.Confinement != "" {
		snapEntry.Confinement = &snapMeta.Confinement
	} else {
		snapEntry.Confinement = nil
	}
	if snapMeta.Base != "" {
		snapEntry.Base = &snapMeta.Base
	} else {
		snapEntry.Base = nil
	}

	query = `
		UPDATE snap_entries
		SET type = $1, confinement = $2, base = $3
		WHERE id = $4
	`
	_, err = sp.db.Exec(query, snapEntry.Type, snapEntry.Confinement, snapEntry.Base, snapEntry.ID)
	if err != nil {
		logrus.Error(err)
		return cerror.ConvertError(err)
	}

	return nil
}

func (sp *SnapsRepository) getPreloadAssociations(entry *models.SnapEntry, preloadAssociations *[]string) *cerror.CustomError {
	all := slices.Contains(*preloadAssociations, models.ALL)

	switch {
	case all || slices.Contains(*preloadAssociations, models.COMMENT):
		resp, err := sp.GetCommentsByEntryId(entry.ID)
		if err != nil {
			// Already logged in GetCommentsByEntryId
			return err
		}
		entry.LatestComments = resp
		fallthrough

	case all || slices.Contains(*preloadAssociations, models.REVISION) || all:
		resp, err := sp.GetRevisionsByEntryId(entry.ID)
		if err != nil {
			// Already logged in GetRevisionsByEntryId
			return err
		}
		entry.Revisions = resp
		fallthrough

	case all || slices.Contains(*preloadAssociations, models.UPLOAD):
		resp, err := sp.GetUploadsByEntryId(entry.ID)
		if err != nil {
			// Already logged in GetUploadsByEntryId
			return err
		}
		entry.Uploads = resp
	}
	return nil
}
