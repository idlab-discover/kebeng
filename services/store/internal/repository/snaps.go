package repository

import (
	"fmt"
	"slices"
	"strings"

	"github.com/idlab-discover/kebeng/services/store/internal/snap"
	"github.com/sirupsen/logrus"

	"github.com/google/uuid"

	"github.com/idlab-discover/kebeng/services/store/internal/models"
	"github.com/jmoiron/sqlx"

	cerror "github.com/idlab-discover/kebeng/common/cerror"
)

type ISnapsRepository interface {
	// CREATE
	AddChannel(snapEntryId uuid.UUID, snapTrackId uuid.UUID, channelName string) (*models.SnapChannel, *cerror.CustomError)
	AddDefaultChannels(snapEntryId uuid.UUID, snapTrackId uuid.UUID) *cerror.CustomError
	AddRevision(entryId uuid.UUID, trackId uuid.UUID, channelId uuid.UUID, size uint64, sequenceNumber uint) (*models.SnapRevision, *cerror.CustomError)
	AddTrack(entryId uuid.UUID, trackName string) (*models.SnapTrack, *cerror.CustomError)
	AddUpload(snapName string, entrId uuid.UUID, status string, accountId uuid.UUID) (*models.SnapUpload, *cerror.CustomError)
	RegisterSnap(snapName string, isPrivate bool, store string, accountId uuid.UUID) (*models.SnapEntry, *cerror.CustomError)

	// READ
	GetAllSnapEntries() (*[]models.SnapEntry, *cerror.CustomError)
	GetChannelsByTrackId(trackId uuid.UUID) ([]*models.SnapChannel, *cerror.CustomError)
	GetCommentsByEntryId(entryId uuid.UUID) ([]*models.SnapComment, *cerror.CustomError)
	GetEntriesByAccountId(accountId uuid.UUID, preloadAssociations []string) ([]*models.SnapEntry, *cerror.CustomError)
	GetEntryById(id uuid.UUID, preloadAssociations []string) (*models.SnapEntry, *cerror.CustomError)
	GetEntryByName(name string, preloadAssociations []string) (*models.SnapEntry, *cerror.CustomError)
	GetRevisionsByEntryId(entryId uuid.UUID) ([]*models.SnapRevision, *cerror.CustomError)
	GetRevisionById(id uuid.UUID) (*models.SnapRevision, *cerror.CustomError)
	GetRevisionByNameAndSequence(name string, sequence uint) (*models.SnapRevision, *cerror.CustomError)
	GetRevisionBySHA(SHA3_384 string, encoded bool) (*models.SnapRevision, *cerror.CustomError)
	GetSections() (*[]string, *cerror.CustomError)
	GetTracksBySnapId(snapId uuid.UUID) ([]*models.SnapTrack, *cerror.CustomError)
	GetTrackById(id uuid.UUID) (*models.SnapTrack, *cerror.CustomError)
	GetChannelById(id uuid.UUID) (*models.SnapChannel, *cerror.CustomError)
	GetLatestRevision(snapName string, track string, channel string) (*models.SnapRevision, *cerror.CustomError)

	// UPDATE
	ReleaseSnap(channels []string, snapEntryId uuid.UUID, revisionId uuid.UUID) *cerror.CustomError
	UpdateRevision(revision *models.SnapRevision, revisionBytes *[]byte) (*models.SnapRevision, *cerror.CustomError)
}

type SnapsRepository struct {
	db *sqlx.DB
}

func NewSnapsRepository(db *sqlx.DB) ISnapsRepository {
	return &SnapsRepository{db: db}
}

// ============ CREATE =============

func (sp *SnapsRepository) AddChannel(snapEntryId uuid.UUID, snapTrackId uuid.UUID, channelName string) (*models.SnapChannel, *cerror.CustomError) {
	channel := models.SnapChannel{
		Name:        channelName,
		SnapEntryID: snapEntryId,
		SnapTrackID: snapTrackId,
	}
	query := `
		INSERT INTO channel (name, entry_id, snap_track_id)
		VALUES ($1, $2, $3)
		RETURNING id
	`
	err := sp.db.Get(&channel.ID, query, channel.Name, channel.SnapEntryID, channel.SnapTrackID)
	if err != nil {
		logrus.Error(err)
		return nil, cerror.ConvertError(err)
	}

	return &channel, nil
}

func (sp *SnapsRepository) AddDefaultChannels(snapEntryId uuid.UUID, snapTrackId uuid.UUID) *cerror.CustomError {
	channels := []string{"stable", "candidate", "beta", "edge"}

	for _, channel := range channels {
		_, err := sp.AddChannel(snapEntryId, snapTrackId, channel)
		if err != nil {
			return err
		}
	}

	return nil
}

func (sp *SnapsRepository) AddRevision(entryId uuid.UUID, trackId uuid.UUID, channelId uuid.UUID, size uint64, sequenceNumber uint) (*models.SnapRevision, *cerror.CustomError) {
	// TODO: fix the need for an empty revision
	// TODO: add build_assertion_filename if an assertion exists -> doesn't get checked in official snap store either
	snapRevision := models.SnapRevision{
		SnapEntryID:    entryId,
		SnapTrackID:    trackId,
		SnapChannelID:  channelId,
		SHA3_384:       nil, // TODO: calculate sha3_384 in logic and at it to the parameters
		Size:           &size,
		SequenceNumber: &sequenceNumber,
	}
	query := `
		INSERT INTO revision (entry_id, snap_track_id, snap_channel_id, sha3_384, size, sequence_number)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`
	err := sp.db.Get(&snapRevision.ID, query, snapRevision.SnapEntryID, snapRevision.SnapTrackID, snapRevision.SnapChannelID, snapRevision.SHA3_384, snapRevision.Size, snapRevision.SequenceNumber)
	if err != nil {
		logrus.Error(err)
		return nil, cerror.ConvertError(err)
	}

	return &snapRevision, nil
}

func (sp *SnapsRepository) AddTrack(entryId uuid.UUID, trackName string) (*models.SnapTrack, *cerror.CustomError) {
	track := models.SnapTrack{
		Name:        trackName,
		SnapEntryID: entryId,
	}
	// If track already exists, this simply returns it
	query := `
		INSERT INTO track (name, entry_id)
		VALUES ($1, $2)
		RETURNING id
	`
	err := sp.db.Get(&track.ID, query, track.Name, track.SnapEntryID)
	if err != nil {
		logrus.Error(err)
		return nil, cerror.ConvertError(err)
	}

	return &track, nil

}

func (sp *SnapsRepository) AddUpload(snapName string, entryId uuid.UUID, status string, accountId uuid.UUID) (*models.SnapUpload, *cerror.CustomError) {
	upload := models.SnapUpload{
		SnapName:  snapName,
		EntryID:   entryId,
		Status:    status,
		AccountID: accountId,
	}
	query := `
		INSERT INTO upload (snap_name, entry_id, status, account_id)
		VALUES ($1, $2, $3, $4)
		RETURNING id, status_details_url
	`
	err := sp.db.Get(&upload, query, upload.SnapName, upload.EntryID, upload.Status, upload.AccountID)
	if err != nil {
		logrus.Error(err)
		return nil, cerror.ConvertError(err)
	}

	return &upload, nil
}

// QUESTION: maybe we can just internaly call this AddEntry -> clearer name?
// QUESTION: right now an snap entry is bound to an account. Wouldn't it be better to bound snap revisions to an account?
func (sp *SnapsRepository) RegisterSnap(snapName string, isPrivate bool, storeName string, accountId uuid.UUID) (*models.SnapEntry, *cerror.CustomError) {
	snapEntry := models.SnapEntry{
		Name:      snapName,
		Private:   &isPrivate,
		Store:     &storeName,
		AccountID: accountId,
	}

	query := `
		INSERT INTO entry (name, private, store, account_id)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`
	err := sp.db.Get(&snapEntry.ID, query, snapName, isPrivate, storeName, accountId)
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
		FROM entry
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
		FROM channel
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
			FROM comment
			WHERE entry_id = $1
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
            FROM entry 
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
		FROM entry
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
		FROM entry
		WHERE name = $1
	`
	err := sp.db.Get(&snapEntry, query, name)
	if err != nil {
		logrus.Warnf("FUNCTION GetEntryByName while searching for snap with name '%s': %s", name, err)
		return nil, cerror.ConvertError(err, fmt.Sprintf("entry not found: snap with name = '%s'", name))
	}

	cerr := sp.getPreloadAssociations(&snapEntry, &preloadAssociations)
	if cerr != nil {
		logrus.Error("failed to preload associations")
		return nil, cerr
	}

	return &snapEntry, nil
}

func (sp *SnapsRepository) GetRevisionsByEntryId(entryId uuid.UUID) ([]*models.SnapRevision, *cerror.CustomError) {
	var revisions []*models.SnapRevision
	query := `
			SELECT *
			FROM revision
			WHERE entry_id = $1
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
		FROM revision
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
		FROM entry
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
		FROM revision
		WHERE entry_id = $1 AND sequence_number = $2
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
			FROM revision
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
			FROM revision
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
		FROM track
		WHERE entry_id = $1
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

func (sp *SnapsRepository) GetLatestRevision(snapName string, track string, channel string) (*models.SnapRevision, *cerror.CustomError) {
	var revision models.SnapRevision
	// NOTE: this query might be quite expensive if there are a lot of revisions
	// NOTE split up to make it more efficient perhaps
	query := `
		SELECT r.*
		FROM entry e
		JOIN track t ON t.entry_id = e.id
		JOIN channel c ON c.entry_id = e.id AND c.snap_track_id = t.id
		JOIN revision r ON r.entry_id = e.id 
					   AND r.snap_track_id = t.id 
					   AND r.snap_channel_id = c.id
		WHERE e.name = $1
		  AND t.name = $2
		  AND c.name = $3
		ORDER BY r.updated_at DESC
		LIMIT 1;
	`
	err := sp.db.Get(&revision, query, snapName, track, channel)
	if err != nil {
		logrus.Error(err)
		return nil, cerror.ConvertError(err, fmt.Sprintf("resource not found: latest revision for snap with name = '%s' and track = '%s' and channel = '%s'", snapName, track, channel))
	}

	return &revision, nil
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
			FROM track
			WHERE entry_id = $1 AND name = $2
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
			FROM channel
			WHERE entry_id = $1 AND name = $2 AND snap_track_id = $3
		`
		err = sp.db.Get(&channel, query, snapEntryId, channelForRelease, track.ID)
		if err != nil {
			logrus.Error(err)
			return cerror.ConvertError(err, fmt.Sprintf("resource not found: channel '%s' for snap with id = '%s'", channelForRelease, snapEntryId.String()))
		}

		var revision models.SnapRevision
		query = `
			SELECT id
			FROM revision
			WHERE id = $1
		`
		err = sp.db.Get(&revision, query, revisionId)
		if err != nil {
			logrus.Error(err)
			return cerror.ConvertError(err, fmt.Sprintf("resource not found: revision with id = '%s'", revisionId.String()))
		}

		query = `
			UPDATE channel
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

// QUESTION: not sure what revisionBytes is for?
func (sp *SnapsRepository) UpdateRevision(revision *models.SnapRevision, revisionBytes *[]byte) (*models.SnapRevision, *cerror.CustomError) {
	var newRevision models.SnapRevision
	query := `
		UPDATE revision
		SET  sha3_384 = $1, sha3_384_encoded = $2, size = $3, sequence_number = $4, architectures = $5, status = $6, version = $7
		WHERE id = $8
		RETURNING *
	`
	err := sp.db.Get(&newRevision, query, revision.SHA3_384, revision.SHA3_384_Encoded, revision.Size, revision.SequenceNumber, revision.Architectures, revision.Status, revision.Version, revision.ID)
	if err != nil {
		logrus.Error(err)
		return nil, cerror.ConvertError(err, fmt.Sprintf("resource not found: revision with id = '%s'", revision.ID.String()))
	}

	return &newRevision, nil
}

func (sp *SnapsRepository) GetTrackById(id uuid.UUID) (*models.SnapTrack, *cerror.CustomError) {
	var track models.SnapTrack
	query := `
		SELECT *
		FROM track
		WHERE id = $1
	`
	err := sp.db.Get(&track, query, id)
	if err != nil {
		logrus.Error(err)
		return nil, cerror.ConvertError(err, fmt.Sprintf("resource not found: track with id = '%s'", id.String()))
	}
	return &track, nil
}

func (sp *SnapsRepository) GetChannelById(id uuid.UUID) (*models.SnapChannel, *cerror.CustomError) {
	var channel models.SnapChannel
	query := `
		SELECT *
		FROM channel
		WHERE id = $1
	`
	err := sp.db.Get(&channel, query, id)
	if err != nil {
		logrus.Error(err)
		return nil, cerror.ConvertError(err, fmt.Sprintf("resource not found: channel with id = '%s'", id.String()))
	}
	return &channel, nil
}

// ============ PRIVATE =============
// ============ HELPER ==============

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
		FROM entry
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
		UPDATE entry
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

	}
	return nil
}
