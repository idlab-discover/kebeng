package repositories

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
	GetEntryByName(name string, preloadAssociations []string) (*models.SnapEntry, *cerror.CustomError)
	GetEntryById(id uuid.UUID, preloadAssociations []string) (*models.SnapEntry, *cerror.CustomError)
	GetEntriesByAccountId(accountId uuid.UUID, preloadAssociations []string) ([]*models.SnapEntry, *cerror.CustomError)
	RegisterSnap(snapName string, isPrivate bool) (*models.SnapEntry, *cerror.CustomError)
	AddSnap(name string, size uint64, accountId uuid.UUID) (*models.SnapEntry, *cerror.CustomError)

	GetRevisionBySHA(SHA3_384 string, encoded bool) (*models.SnapRevision, *cerror.CustomError)
	GetUploadByUpDownId(upDownId string) (*models.SnapUpload, *cerror.CustomError)
	UpdateRevision(revision *models.SnapRevision, revisionBytes *[]byte) (*models.SnapRevision, *cerror.CustomError)

	ReleaseSnap(channels []string, snapEntryId uuid.UUID, revisionId uuid.UUID) *cerror.CustomError
	AddUpload(snapName string, upDownId string, size uint, channels []string) (*models.SnapUpload, *cerror.CustomError)

	SetChannelRevision(trackName string, channelName string, revisionId uint, snapId uuid.UUID) (*models.SnapTrack, *cerror.CustomError)

	GetTracksBySnapId(snapId uuid.UUID) (*[]models.SnapTrack, *cerror.CustomError)
	Getchannels(trackId uuid.UUID) (*[]models.SnapChannel, *cerror.CustomError)
	GetRevisionById(id string) (*models.SnapRevision, *cerror.CustomError)
	GetRevisionByChannel(channel string, snapName string) (*models.SnapRevision, *cerror.CustomError)

	GetSections() (*[]string, *cerror.CustomError)

	GetSnaps() (*[]models.SnapEntry, *cerror.CustomError)
}

type SnapsRepository struct {
	db *sqlx.DB
}

func NewSnapsRepository(db *sqlx.DB) *SnapsRepository {
	return &SnapsRepository{db: db}
}

func (sp *SnapsRepository) GetRevisionByChannel(channel string, snapName string) (*models.SnapRevision, *cerror.CustomError) {
	snapEntry, err := sp.GetEntryByName(snapName, nil)
	if err != nil {
		// Already logged in GetEntryByName
		return nil, err
	}
	if snapEntry != nil {
		channelParts := strings.Split(channel, "/")
		var track string
		var channel string
		if len(channelParts) == 1 {
			if channelParts[0] == "beta" || channelParts[0] == "edge" || channelParts[0] == "stable" || channelParts[0] == "candidate" {
				track = "latest"
				channel = channelParts[0]
			} else {
				track = channelParts[0]
				channel = "stable"
			}
		} else if len(channelParts) == 2 {
			track = channelParts[0]
			channel = channelParts[1]
		} else {
			return nil, cerror.NewCustomError(cerror.NotImplemented, "branches not yet supported for channels")
		}

		query := `
			SELECT id
			FROM snap_tracks
			WHERE snap_entry_id = $1 AND name = $2
		`
		var trackId uint
		err := sp.db.Get(&trackId, query, snapEntry.ID, track)
		if err != nil {

			query = `
			SELECT *
			FROM snap_channels
			WHERE snap_entry_id = $1 AND name = $2 AND snap_track_id = $3
		`
			var snapChannel models.SnapChannel
			err = sp.db.Get(&snapChannel, query, snapEntry.ID, channel, trackId)
			if err != nil {
				logrus.Error(err)
				return nil, cerror.ConvertError(err, "resource not found: channel '%s' for snap with id = '%s'", channel, snapEntry.ID.String())
			}
			return snapChannel.Revision, nil

		}
		return nil, cerror.NewCustomError(cerror.ResourceNotFound, "track not found")

	}
	return nil, cerror.NewCustomError(cerror.ResourceNotFound, "snap not found")
}

func (sp *SnapsRepository) GetSections() (*[]string, *cerror.CustomError) {
	// TODO: add these to the database for real
	sections := []string{
		"general",
	}

	return &sections, nil
}

func (sp *SnapsRepository) GetTracksBySnapId(snapId uuid.UUID) (*[]models.SnapTrack, *cerror.CustomError) {
	var tracks []models.SnapTrack

	query := `
		SELECT *
		FROM snap_tracks
		WHERE snap_entry_id = $1
	`
	err := sp.db.Select(&tracks, query, snapId)
	if err != nil {
		logrus.Error(err)
		return nil, cerror.ConvertError(err, "resource not found: tracks for snap with id = '%s'", snapId.String())
	}

	return &tracks, nil
}

func (sp *SnapsRepository) Getchannels(trackId uuid.UUID) (*[]models.SnapChannel, *cerror.CustomError) {
	var channels []models.SnapChannel
	query := `
		SELECT *
		FROM snap_channels
		WHERE snap_track_id = $1
	`
	err := sp.db.Select(&channels, query, trackId)
	if err != nil {
		logrus.Error(err)
		return nil, cerror.ConvertError(err, "resource not found: channels for track with id = '%s'", trackId.String())
	}

	return &channels, nil
}

func (sp *SnapsRepository) GetRevisionById(id string) (*models.SnapRevision, *cerror.CustomError) {
	var revision models.SnapRevision
	query := `
		SELECT *
		FROM snap_revisions
		WHERE id = $1
	`
	err := sp.db.Get(&revision, query, id)
	if err != nil {
		logrus.Error(err)
		return nil, cerror.ConvertError(err, "resource not found: revision with id = '%s'", id)
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
		return nil, cerror.ConvertError(err, "resource not found: snap with name = '%s' while searching for revision", name)
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
		return nil, cerror.ConvertError(err, "resource not found: revision with sequence = '%d' for snap with name = '%s'", fmt.Sprint(sequence), name)
	}

	return &revision, nil
}

func (sp *SnapsRepository) SetChannelRevision(trackName string, channelName string, revisionId uuid.UUID, snapId uuid.UUID) (*models.SnapTrack, *cerror.CustomError) {
	// get all the tracks
	var track models.SnapTrack
	query := `
		SELECT id
		FROM snap_tracks
		WHERE snap_entry_id = $1 AND name = $2
	`
	err := sp.db.Get(&track, query, snapId, trackName)
	if err != nil {
		logrus.Error(err)
		return nil, cerror.ConvertError(err, "resource not found: track '%s' for snap with id = '%s'", trackName, snapId.String())
	}

	var channel models.SnapChannel
	query = `
		SELECT *
		FROM snap_channels
		WHERE snap_entry_id = $1 AND name = $2 AND snap_track_id = $3
	`
	err = sp.db.Get(&channel, query, snapId, channelName, track.ID)
	if err != nil {
		logrus.Error(err)
		return nil, cerror.ConvertError(err, "resource not found: channel '%s' for snap with id = '%s'", channelName, snapId.String())
	}

	var revision models.SnapRevision
	query = `
		SELECT *
		FROM snap_revisions
		WHERE id = $1
	`
	err = sp.db.Get(&revision, query, revisionId)
	if err != nil {
		logrus.Error(err)
		return nil, cerror.ConvertError(err, "resource not found: revision with id = '%s'", revisionId.String())
	}

	query = `
		UPDATE snap_channels
		SET revision_id = $1
		WHERE id = $2
	`
	_, err = sp.db.Exec(query, revision.ID, channel.ID)
	if err != nil {
		logrus.Error(err)
		return nil, cerror.ConvertError(err, "resource not found: channel '%s' for snap with id = '%s'", channelName, snapId.String())
	}

	return &track, nil
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
		return nil, cerror.ConvertError(err, "resource not found: snap with name = '%s'", snapName)
	}

	snapUpload := models.SnapUpload{
		Name:        snapName,
		UpDownID:    upDownId,
		Filesize:    fileSize,
		SnapEntryID: snap.ID,
	}

	logrus.Infof("Uploading: %s", snapName)

	// TODO: fix lazy; this should be converted to a table so that the channels can be stored separately or maybe redis
	if len(channels) > 0 {
		channelsString := ""
		for _, chn := range channels {
			if channelsString == "" {
				channelsString = chn
			} else {
				channelsString = channelsString + "," + chn
			}
		}

		snapUpload.Channels = channelsString
	}

	query = `
		INSERT INTO snap_uploads (name, up_down_id, filesize, channels, snap_entry_id)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`
	err = sp.db.Get(&snapUpload.ID, query, snapName, upDownId, fileSize, snapUpload.Channels, snap.ID)
	if err != nil {
		logrus.Error(err)
		return nil, cerror.ConvertError(err, "resource not found: snap with name = '%s'", snapName)
	}

	return &snapUpload, nil
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
		return nil, cerror.ConvertError(err, "resource not found: upload with up_down_id = '%s'", upDownId)
	}

	return &snapUpload, nil
}

func (sp *SnapsRepository) UpdateRevision(revision *models.SnapRevision, revisionBytes *[]byte) (*models.SnapRevision, *cerror.CustomError) {
	var newRevision models.SnapRevision
	query := `
		UPDATE snap_revisions
		SET snap_filename = $1, sha3_384 = $2, sha3_384_encoded = $3, size = $4, sequence_number = $5, architectures = $6, status = $7, version = $8, since = $9
		WHERE id = $10
		RETURNING *
	`
	err := sp.db.Get(&newRevision, query, revision.SnapFilename, revision.SHA3_384, revision.SHA3_384_Encoded, revision.Size, revision.SequenceNumber, revision.Architectures, revision.Status, revision.Version, revision.Since, revision.ID)
	if err != nil {
		logrus.Error(err)
		return nil, cerror.ConvertError(err, "resource not found: revision with id = '%s'", revision.ID.String())
	}

	return &newRevision, nil
}

func (sp *SnapsRepository) GetSnaps() (*[]models.SnapEntry, *cerror.CustomError) {
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

	return &snaps, nil
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
		return nil, cerror.ConvertError(err, "resource not found: snap with id = '%s'", Id.String())
	}

	all := slices.Contains(preloadAssociations, models.ALL)

	switch {
	case all || slices.Contains(preloadAssociations, models.COMMENT):
		resp, err := sp.GetCommentsByEntryId(snapEntry.ID)
		if err != nil {
			// Already logged in GetCommentsByEntryId
			return nil, err
		}
		snapEntry.LatestComments = resp
		fallthrough

	case all || slices.Contains(preloadAssociations, models.REVISION) || all:
		resp, err := sp.GetRevisionsByEntryId(snapEntry.ID)
		if err != nil {
			// Already logged in GetRevisionsByEntryId
			return nil, err
		}
		snapEntry.Revisions = resp
		fallthrough

	case all || slices.Contains(preloadAssociations, models.UPLOAD):
		resp, err := sp.GetUploadsByEntryId(snapEntry.ID)
		if err != nil {
			// Already logged in GetUploadsByEntryId
			return nil, err
		}
		snapEntry.Uploads = resp
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
		logrus.Error(err)
		return nil, cerror.ConvertError(err, "resource not found: snap with name = '%s'", name)
	}

	all := slices.Contains(preloadAssociations, models.ALL)

	switch {
	case all || slices.Contains(preloadAssociations, models.COMMENT):
		resp, err := sp.GetCommentsByEntryId(snapEntry.ID)
		if err != nil {
			// Already logged in GetCommentsByEntryId
			return nil, err
		}
		snapEntry.LatestComments = resp
		fallthrough

	case all || slices.Contains(preloadAssociations, models.REVISION) || all:
		resp, err := sp.GetRevisionsByEntryId(snapEntry.ID)
		if err != nil {
			// Already logged in GetRevisionsByEntryId
			return nil, err
		}
		snapEntry.Revisions = resp
		fallthrough

	case all || slices.Contains(preloadAssociations, models.UPLOAD):
		resp, err := sp.GetUploadsByEntryId(snapEntry.ID)
		if err != nil {
			// Already logged in GetUploadsByEntryId
			return nil, err
		}
		snapEntry.Uploads = resp
	}

	return &snapEntry, nil
}

// Used when a new snap gets uploaded for the first time (=registering a snap)

// AddSnap registers a new snap with the given name, size, and accountId.
// It ensures the snap does not already exist, creates a new SnapEntry,
// adds an initial upload, and sets up default tracks and channels.
func (sp *SnapsRepository) AddSnap(name string, size uint64, accountId uuid.UUID) (*models.SnapEntry, *cerror.CustomError) {
	existingSnap, err := sp.GetEntryByName(name, nil)
	if err != nil {
		// Already logged in GetEntryByName
		return nil, err
	}

	// if the snap already exists, return an *cerror.CustomError
	if existingSnap != nil {
		return nil, cerror.NewCustomError(cerror.AlreadyRegistered, fmt.Sprintf("snap with name '%s' already exists", name))
	}

	// when registering a snap, not finding one is what you want
	var newSnapEntry models.SnapEntry
	newSnapEntry.Name = name
	newSnapEntry.AccountID = accountId
	typeStr := "app"
	newSnapEntry.Type = &typeStr
	//newSnapEntry.Confinement = "strict"
	//newSnapEntry.Base = "core18" // default base

	// snap_entries table contains snaps with unique names (doesn't keep track of revisions or channels)
	query := `
		INSERT INTO snap_entries (name, account_id, type)
		VALUES ($1, $2, $3)
		RETURNING id
	`
	err2 := sp.db.Get(&newSnapEntry.ID, query, name, accountId, "app")
	if err2 != nil {
		logrus.Error(err2)
		return nil, cerror.ConvertError(err2)
	}

	// snap_uploads table contains channels where the snap is uploaded
	sp.AddUpload(name, newSnapEntry.ID.String(), uint(size), []string{"latest/stable"})

	// For now when we register a snap we are going to create the default tracks/channels
	track := models.SnapTrack{
		Name:        "latest", // first upload of a snap is always the current latest
		SnapEntryID: newSnapEntry.ID,
	}
	query = `
		INSERT INTO snap_tracks (name, snap_entry_id)
		VALUES ($1, $2)
		RETURNING id
	`
	err2 = sp.db.Get(&track.ID, query, track.Name, newSnapEntry.ID)
	if err2 != nil {
		logrus.Error(err2)
		return nil, cerror.ConvertError(err2)
	}

	newRevision, _ := sp.addRevision(newSnapEntry, size)

	sp.addChannels(newSnapEntry, *newRevision, track.ID)

	return &newSnapEntry, nil

}

func (sp *SnapsRepository) AddDefaultChannels(newSnapEntry models.SnapEntry, newRevision models.SnapRevision, trackId uuid.UUID) *cerror.CustomError {
	return sp.addChannels(newSnapEntry, newRevision, trackId)
}

func (sp *SnapsRepository) ReleaseSnap(channels []string, snapEntryId uuid.UUID, revisionId uuid.UUID) *cerror.CustomError {
	var trackForRelease string
	var channelForRelease string
	for _, cn := range channels {
		// It's possible this comes in the form:
		//   - single string values "edge" where the track is assumed to be "latest" there is no branch
		//   - two values "latest/edge" where the channel is proceeded by the track
		//   - three values "latest/edge/some_branch"
		parts := strings.Split(cn, "/")
		if len(parts) == 1 {
			channelForRelease = parts[0]
			trackForRelease = "latest"
		} else if len(parts) == 2 {
			trackForRelease = parts[0]
			channelForRelease = parts[1]
		} else if len(parts) == 3 {
			return cerror.NewCustomError(cerror.NotImplemented, "branches not yet supported for channels")
		}

		// get all the tracks
		var track models.SnapTrack
		query := `
			SELECT id
			FROM snap_tracks
			WHERE snap_entry_id = $1 AND name = $2
		`
		err := sp.db.Get(&track, query, snapEntryId, trackForRelease)
		if err != nil {
			logrus.Error(err)
			return cerror.ConvertError(err, "resource not found: track '%s' for snap with id = '%s'", trackForRelease, snapEntryId.String())
		}

		// get all the channels
		var channel models.SnapChannel
		query = `
			SELECT *
			FROM snap_channels
			WHERE snap_entry_id = $1 AND name = $2 AND snap_track_id = $3
		`
		err = sp.db.Get(&channel, query, snapEntryId, channelForRelease, track.ID)
		if err != nil {
			logrus.Error(err)
			return cerror.ConvertError(err, "resource not found: channel '%s' for snap with id = '%s'", channelForRelease, snapEntryId.String())
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
			return cerror.ConvertError(err, "resource not found: revision with id = '%s'", revisionId.String())
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

func (sp *SnapsRepository) addRevision(snapEntry models.SnapEntry, size uint64) (*models.SnapRevision, *cerror.CustomError) {
	// TODO: fix the need for an empty revision
	snapRevision := models.SnapRevision{
		SnapFilename: snapEntry.Name,
		SnapEntryID:  snapEntry.ID,
		SHA3_384:     "",
		Size:         size,
	}

	query := `
		INSERT INTO snap_revisions (snap_filename, snap_entry_id, sha3_384, size)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`
	err := sp.db.Get(&snapRevision.ID, query, snapRevision.SnapFilename, snapRevision.SnapEntryID, snapRevision.SHA3_384, snapRevision.Size)
	if err != nil {
		logrus.Error(err)
		return nil, cerror.ConvertError(err)
	}

	return &snapRevision, nil
}

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
		return cerror.ConvertError(err, "resource not found: snap with name = '%s'", snapMeta.Name)
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

// sqlx version
func (sp *SnapsRepository) GetEntriesByAccountId(accountId uuid.UUID, preloadAssociations []string) ([]*models.SnapEntry, *cerror.CustomError) {
	query := `
            SELECT * 
            FROM snap_entries 
            WHERE account_id = $1
        `
	var entries []*models.SnapEntry
	err := sp.db.Select(&entries, query, accountId)
	if err != nil {
		logrus.Error(err)
		return nil, cerror.ConvertError(err, "resource not found: snaps for account with id = '%s'", accountId.String())
	}

	for _, entry := range entries {

		all := slices.Contains(preloadAssociations, models.ALL)

		switch {
		case all || slices.Contains(preloadAssociations, models.COMMENT):
			resp, err := sp.GetCommentsByEntryId(entry.ID)
			if err != nil {
				logrus.Error(err)
				return nil, err
			}
			entry.LatestComments = resp
			fallthrough

		case all || slices.Contains(preloadAssociations, models.REVISION) || all:
			resp, err := sp.GetRevisionsByEntryId(entry.ID)
			if err != nil {
				logrus.Error(err)
				return nil, err
			}
			entry.Revisions = resp
			fallthrough

		case all || slices.Contains(preloadAssociations, models.UPLOAD):
			resp, err := sp.GetUploadsByEntryId(entry.ID)
			if err != nil {
				logrus.Error(err)
				return nil, err
			}
			entry.Uploads = resp
		}
	}

	return entries, nil
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
		return nil, cerror.ConvertError(err, "resource not found: uploads for snap with id = '%s'", d.String())
	}

	return uploads, nil
}

func (sp *SnapsRepository) GetCommentsByEntryId(d uuid.UUID) ([]*models.SnapComment, *cerror.CustomError) {
	query := `
			SELECT *
			FROM snap_comments
			WHERE snap_entry_id = $1
		`
	var comments []*models.SnapComment
	err := sp.db.Select(&comments, query, d)
	if err != nil {
		logrus.Error(err)
		return nil, cerror.ConvertError(err, "resource not found: comments for snap with id = '%s'", d.String())
	}

	return comments, nil
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
		return nil, cerror.ConvertError(err, "resource not found: revisions for snap with id = '%s'", entryId.String())
	}

	return revisions, nil
}

func (sp *SnapsRepository) getSnap(whereModel *models.SnapEntry, preloadAssociations []string) (*models.SnapEntry, *cerror.CustomError) {
	var snapEntry models.SnapEntry
	query := `
		SELECT *
		FROM snap_entries
		WHERE id = $1
	`
	err := sp.db.Get(&snapEntry, query, whereModel.ID)
	if err != nil {
		logrus.Error(err)
		return nil, cerror.ConvertError(err, "resource not found: snap with id = '%s'", whereModel.ID.String())
	}

	all := slices.Contains(preloadAssociations, models.ALL)

	switch {
	case all || slices.Contains(preloadAssociations, models.COMMENT):
		resp, err := sp.GetCommentsByEntryId(snapEntry.ID)
		if err != nil {
			logrus.Error(err)
			return nil, err
		}
		snapEntry.LatestComments = resp
		fallthrough

	case all || slices.Contains(preloadAssociations, models.REVISION) || all:
		resp, err := sp.GetRevisionsByEntryId(snapEntry.ID)
		if err != nil {
			logrus.Error(err)
			return nil, err
		}
		snapEntry.Revisions = resp
		fallthrough

	case all || slices.Contains(preloadAssociations, models.UPLOAD):
		resp, err := sp.GetUploadsByEntryId(snapEntry.ID)
		if err != nil {
			logrus.Error(err)
			return nil, err
		}
		snapEntry.Uploads = resp
	}

	return &snapEntry, nil
}
