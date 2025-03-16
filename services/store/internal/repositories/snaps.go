package repositories

import (
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/idlab-discover/kebeng/services/store/internal/snap"
	"github.com/sirupsen/logrus"

	"github.com/google/uuid"

	"github.com/idlab-discover/kebeng/services/store/internal/models"
	"github.com/jmoiron/sqlx"

	cerrors "github.com/idlab-discover/kebeng/services/store/internal/errors"
)

type ISnapsRepository interface {
	GetEntryByName(name string, preloadAssociations []string) (*models.SnapEntry, error)
	GetEntryById(id uuid.UUID, preloadAssociations []string) (*models.SnapEntry, error)
	GetEntriesByAccountId(accountId uuid.UUID, preloadAssociations []string) ([]*models.SnapEntry, error)
	RegisterSnap(snapName string, isPrivate bool) (*models.SnapEntry, error)
	AddSnap(name string, size uint64, accountId uuid.UUID) (*models.SnapEntry, error)

	GetRevisionBySHA(SHA3_384 string, encoded bool) (*models.SnapRevision, error)
	GetUploadByUpDownId(upDownId string) (*models.SnapUpload, error)
	UpdateRevision(revision *models.SnapRevision, revisionBytes *[]byte) (*models.SnapRevision, error)

	ReleaseSnap(channels []string, snapEntryId uuid.UUID, revisionId uuid.UUID) error
	AddUpload(snapName string, upDownId string, size uint, channels []string) (*models.SnapUpload, error)

	SetChannelRevision(trackName string, channelName string, revisionId uint, snapId uuid.UUID) (*models.SnapTrack, error)

	GetTracksBySnapId(snapId uuid.UUID) (*[]models.SnapTrack, error)
	Getchannels(trackId uuid.UUID) (*[]models.SnapChannel, error)
	GetRevisionById(id string) (*models.SnapRevision, error)
	GetRevisionByChannel(channel string, snapName string) (*models.SnapRevision, error)

	GetSections() (*[]string, error)

	GetSnaps() (*[]models.SnapEntry, error)
}

type SnapsRepository struct {
	db *sqlx.DB
}

func NewSnapsRepository(db *sqlx.DB) *SnapsRepository {
	return &SnapsRepository{db: db}
}

func (sp *SnapsRepository) GetRevisionByChannel(channel string, snapName string) (*models.SnapRevision, error) {
	snapEntry, err := sp.GetEntryByName(snapName, nil)
	if err == nil && snapEntry != nil {
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
			return nil, errors.New("branches not yet supported for channels")
		}

		query := `
			SELECT id
			FROM snap_tracks
			WHERE snap_entry_id = $1 AND name = $2
		`
		var trackId uint
		err := sp.db.Get(&trackId, query, snapEntry.ID, track)
		if err == sql.ErrNoRows {
			logrus.Error(err)
			return nil, errors.New(cerrors.ResourceNotFound)
		} else if err != nil {
			logrus.Error(err)
			return nil, err
		}

		query = `
			SELECT *
			FROM snap_channels
			WHERE snap_entry_id = $1 AND name = $2 AND snap_track_id = $3
		`
		var snapChannel models.SnapChannel
		err = sp.db.Get(&snapChannel, query, snapEntry.ID, channel, trackId)
		if err == sql.ErrNoRows {
			logrus.Error(err)
			return nil, errors.New(cerrors.ResourceNotFound)
		} else if err != nil {
			logrus.Error(err)
			return nil, err
		}

		return snapChannel.Revision, nil

	} else if err != nil {
		logrus.Error(err)
		return nil, err
	}

	return nil, errors.New("unknown error encountered trying to find revision for snap by channel")
}

func (sp *SnapsRepository) GetSections() (*[]string, error) {
	// TODO: add these to the database for real
	sections := []string{
		"general",
	}

	return &sections, nil
}

func (sp *SnapsRepository) GetTracksBySnapId(snapId uuid.UUID) (*[]models.SnapTrack, error) {
	var tracks []models.SnapTrack

	query := `
		SELECT *
		FROM snap_tracks
		WHERE snap_entry_id = $1
	`
	err := sp.db.Select(&tracks, query, snapId)
	if err == sql.ErrNoRows {
		logrus.Error(err)
		return nil, errors.New(cerrors.ResourceNotFound)
	} else if err != nil {
		logrus.Error(err)
		return nil, err
	}

	return &tracks, nil
}

func (sp *SnapsRepository) Getchannels(trackId uuid.UUID) (*[]models.SnapChannel, error) {
	var channels []models.SnapChannel
	query := `
		SELECT *
		FROM snap_channels
		WHERE snap_track_id = $1
	`
	err := sp.db.Select(&channels, query, trackId)
	if err == sql.ErrNoRows {
		logrus.Error(err)
		return nil, errors.New(cerrors.ResourceNotFound)
	} else if err != nil {
		logrus.Error(err)
		return nil, err
	}

	return &channels, nil
}

func (sp *SnapsRepository) GetRevisionById(id string) (*models.SnapRevision, error) {
	var revision models.SnapRevision
	query := `
		SELECT *
		FROM snap_revisions
		WHERE id = $1
	`
	err := sp.db.Get(&revision, query, id)
	if err == sql.ErrNoRows {
		logrus.Error(err)
		return nil, errors.New(cerrors.ResourceNotFound)
	} else if err != nil {
		logrus.Error(err)
		return nil, err
	}

	return &revision, nil
}

func (sp *SnapsRepository) GetRevisionByNameAndSequence(name string, sequence uint) (*models.SnapRevision, error) {
	var entry models.SnapEntry
	query := `
		SELECT id
		FROM snap_entries
		WHERE name = $1
	`
	err := sp.db.Get(&entry, query, name)
	if err == sql.ErrNoRows {
		logrus.Error(err)
		return nil, errors.New(cerrors.ResourceNotFound)
	} else if err != nil {
		logrus.Error(err)
		return nil, err
	}

	var revision models.SnapRevision
	query = `
		SELECT *
		FROM snap_revisions
		WHERE snap_entry_id = $1 AND sequence_number = $2
	`
	err = sp.db.Get(&revision, query, entry.ID, sequence)
	if err == sql.ErrNoRows {
		logrus.Error(err)
		return nil, errors.New(cerrors.ResourceNotFound)
	} else if err != nil {
		logrus.Error(err)
		return nil, err
	}

	return &revision, nil
}

func (sp *SnapsRepository) SetChannelRevision(trackName string, channelName string, revisionId uint, snapId uuid.UUID) (*models.SnapTrack, error) {
	// get all the tracks
	var track models.SnapTrack
	query := `
		SELECT id
		FROM snap_tracks
		WHERE snap_entry_id = $1 AND name = $2
	`
	err := sp.db.Get(&track, query, snapId, trackName)
	if err == sql.ErrNoRows {
		logrus.Error(err)
		return nil, errors.New(cerrors.ResourceNotFound)
	} else if err != nil {
		logrus.Error(err)
		return nil, err
	}

	var channel models.SnapChannel
	query = `
		SELECT *
		FROM snap_channels
		WHERE snap_entry_id = $1 AND name = $2 AND snap_track_id = $3
	`
	err = sp.db.Get(&channel, query, snapId, channelName, track.ID)
	if err == sql.ErrNoRows {
		logrus.Error(err)
		return nil, errors.New(cerrors.ResourceNotFound)
	} else if err != nil {
		logrus.Error(err)
		return nil, err
	}

	var revision models.SnapRevision
	query = `
		SELECT *
		FROM snap_revisions
		WHERE id = $1
	`
	err = sp.db.Get(&revision, query, revisionId)
	if err == sql.ErrNoRows {
		logrus.Error(err)
		return nil, errors.New(cerrors.ResourceNotFound)
	} else if err != nil {
		logrus.Error(err)
		return nil, err
	}

	query = `
		UPDATE snap_channels
		SET revision_id = $1
		WHERE id = $2
	`
	_, err = sp.db.Exec(query, revision.ID, channel.ID)
	if err != nil {
		logrus.Error(err)
		return nil, err
	}

	return &track, nil
}

func (sp *SnapsRepository) RegisterSnap(snapName string, isPrivate bool) (*models.SnapEntry, error) {
	snapEntry := models.SnapEntry{
		Name:    snapName,
		Private: sql.NullBool{Bool: isPrivate, Valid: true},
	}

	query := `
		INSERT INTO snap_entries (name, private)
		VALUES ($1, $2)
		RETURNING id
	`
	err := sp.db.Get(&snapEntry.ID, query, snapName, isPrivate)
	if err == sql.ErrNoRows {
		logrus.Error(err)
		return nil, errors.New(cerrors.ResourceNotFound)
	} else if err != nil {
		logrus.Error(err)
		return nil, err
	}

	return &snapEntry, nil
}

func (sp *SnapsRepository) AddUpload(snapName string, upDownId string, fileSize uint, channels []string) (*models.SnapUpload, error) {
	var snap models.SnapEntry
	query := `
		SELECT id
		FROM snap_entries
		WHERE name = $1
	`
	err := sp.db.Get(&snap.ID, query, snapName)
	if err == sql.ErrNoRows {
		logrus.Error(err)
		return nil, errors.New(cerrors.ResourceNotFound)
	} else if err != nil {
		logrus.Error(err)
		return nil, err
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
	if err == sql.ErrNoRows {
		logrus.Error(err)
		return nil, errors.New(cerrors.ResourceNotFound)
	} else if err != nil {
		logrus.Error(err)
		return nil, err
	}

	return &snapUpload, nil
}

func (sp *SnapsRepository) GetRevisionBySHA(SHA3_384 string, encoded bool) (*models.SnapRevision, error) {
	var revision models.SnapRevision

	if encoded {
		logrus.Tracef("Getting snap revision by encoded sha3_384: %s", SHA3_384)
		query := `
			SELECT *
			FROM snap_revisions
			WHERE sha3_384_encoded = $1
		`
		err := sp.db.Get(&revision, query, SHA3_384)
		if err == sql.ErrNoRows {
			logrus.Error(err)
			return nil, errors.New(cerrors.ResourceNotFound)
		} else if err != nil {
			logrus.Error(err)
			return nil, err
		}
	} else {
		logrus.Tracef("Getting snap revision by sha3_384: %s", SHA3_384)
		query := `
			SELECT *
			FROM snap_revisions
			WHERE sha3_384 = $1
		`
		err := sp.db.Get(&revision, query, SHA3_384)
		if err == sql.ErrNoRows {
			logrus.Error(err)
			return nil, errors.New(cerrors.ResourceNotFound)
		} else if err != nil {
			logrus.Error(err)
			return nil, err
		}
	}

	return &revision, nil
}

func (sp *SnapsRepository) GetUploadByUpDownId(upDownId string) (*models.SnapUpload, error) {
	var snapUpload models.SnapUpload
	query := `
		SELECT *
		FROM snap_uploads
		WHERE up_down_id = $1
	`
	err := sp.db.Get(&snapUpload, query, upDownId)
	if err == sql.ErrNoRows {
		logrus.Error(err)
		return nil, errors.New(cerrors.ResourceNotFound)
	} else if err != nil {
		logrus.Error(err)
		return nil, err
	}

	return &snapUpload, nil
}

func (sp *SnapsRepository) UpdateRevision(revision *models.SnapRevision, revisionBytes *[]byte) (*models.SnapRevision, error) {
	var newRevision models.SnapRevision
	query := `
		UPDATE snap_revisions
		SET snap_filename = $1, sha3_384 = $2, sha3_384_encoded = $3, size = $4, sequence_number = $5, architectures = $6, status = $7, version = $8, since = $9
		WHERE id = $10
		RETURNING *
	`
	err := sp.db.Get(&newRevision, query, revision.SnapFilename, revision.SHA3_384, revision.SHA3_384_Encoded, revision.Size, revision.SequenceNumber, revision.Architectures, revision.Status, revision.Version, revision.Since, revision.ID)
	if err == sql.ErrNoRows {
		logrus.Error(err)
		return nil, errors.New(cerrors.ResourceNotFound)
	} else if err != nil {
		logrus.Error(err)
		return nil, err
	}

	return &newRevision, nil
}

func (sp *SnapsRepository) GetSnaps() (*[]models.SnapEntry, error) {
	var snaps []models.SnapEntry
	query := `
		SELECT *
		FROM snap_entries
	`
	err := sp.db.Select(&snaps, query)
	if err == sql.ErrNoRows {
		logrus.Error(err)
		return nil, errors.New(cerrors.ResourceNotFound)
	} else if err != nil {
		logrus.Error(err)
		return nil, err
	}

	return &snaps, nil
}

func (sp *SnapsRepository) GetEntryById(Id uuid.UUID, preloadAssociations []string) (*models.SnapEntry, error) {
	var snapEntry models.SnapEntry
	query := `
		SELECT *
		FROM snap_entries
		WHERE id = $1
	`
	err := sp.db.Get(&snapEntry, query, Id)
	if err == sql.ErrNoRows {
		logrus.Error(err)
		return nil, errors.New(cerrors.ResourceNotFound)
	} else if err != nil {
		logrus.Error(err)
		return nil, errors.New(cerrors.DatabaseError)
	}

	switch {
	case slices.Contains(preloadAssociations, models.COMMENT):
		resp, err := sp.GetCommentsByEntryId(snapEntry.ID)
		if err != nil {
			logrus.Error(err)
			return nil, err
		}
		snapEntry.LatestComments = resp
		fallthrough

	case slices.Contains(preloadAssociations, models.REVISION):
		resp, err := sp.GetRevisionsByEntryId(snapEntry.ID)
		if err != nil {
			logrus.Error(err)
			return nil, err
		}
		snapEntry.Revisions = resp
		fallthrough

	case slices.Contains(preloadAssociations, models.UPLOAD):
		resp, err := sp.GetUploadsByEntryId(snapEntry.ID)
		if err != nil {
			logrus.Error(err)
			return nil, err
		}
		snapEntry.Uploads = resp
	}

	return &snapEntry, nil
}

func (sp *SnapsRepository) GetEntryByName(name string, preloadAssociations []string) (*models.SnapEntry, error) {
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
	if err == sql.ErrNoRows {
		logrus.Error(err)
		return nil, errors.New(cerrors.ResourceNotFound)
	} else if err != nil {
		logrus.Error(err)
		return nil, errors.New(cerrors.DatabaseError)
	}

	switch {
	case slices.Contains(preloadAssociations, models.COMMENT):
		resp, err := sp.GetCommentsByEntryId(snapEntry.ID)
		if err != nil {
			logrus.Error(err)
			return nil, err
		}
		snapEntry.LatestComments = resp
		fallthrough

	case slices.Contains(preloadAssociations, models.REVISION):
		resp, err := sp.GetRevisionsByEntryId(snapEntry.ID)
		if err != nil {
			logrus.Error(err)
			return nil, err
		}
		snapEntry.Revisions = resp
		fallthrough

	case slices.Contains(preloadAssociations, models.UPLOAD):
		resp, err := sp.GetUploadsByEntryId(snapEntry.ID)
		if err != nil {
			logrus.Error(err)
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
func (sp *SnapsRepository) AddSnap(name string, size uint64, accountId uuid.UUID) (*models.SnapEntry, error) {
	existingSnap, err := sp.GetEntryByName(name, nil)
	if err != nil {
		logrus.Error(err)
		return nil, err
	}

	// if the snap already exists, return an error
	if existingSnap != nil {
		return nil, fmt.Errorf("snap with name=%s already exists", name)
	}

	// when registering a snap, not finding one is what you want
	var newSnapEntry models.SnapEntry
	newSnapEntry.Name = name
	newSnapEntry.AccountID = accountId
	newSnapEntry.Type = sql.NullString{String: "app", Valid: true}
	//newSnapEntry.Confinement = "strict"
	//newSnapEntry.Base = "core18" // default base

	// snap_entries table contains snaps with unique names (doesn't keep track of revisions or channels)
	query := `
		INSERT INTO snap_entries (name, account_id, type)
		VALUES ($1, $2, $3)
		RETURNING id
	`
	err = sp.db.Get(&newSnapEntry.ID, query, name, accountId, "app")
	if err == sql.ErrNoRows {
		logrus.Error(err)
		return nil, errors.New(cerrors.ResourceNotFound)
	} else if err != nil {
		logrus.Error(err)
		return nil, errors.New(cerrors.DatabaseError)
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
	err = sp.db.Get(&track.ID, query, track.Name, newSnapEntry.ID)
	if err == sql.ErrNoRows {
		logrus.Error(err)
		return nil, errors.New(cerrors.ResourceNotFound)
	} else if err != nil {
		logrus.Error(err)
		return nil, errors.New(cerrors.DatabaseError)
	}

	newRevision, _ := sp.addRevision(newSnapEntry, size)

	sp.addChannels(newSnapEntry, *newRevision, track.ID)

	return &newSnapEntry, nil

}

func (sp *SnapsRepository) AddDefaultChannels(newSnapEntry models.SnapEntry, newRevision models.SnapRevision, trackId uuid.UUID) error {
	return sp.addChannels(newSnapEntry, newRevision, trackId)
}

func (sp *SnapsRepository) ReleaseSnap(channels []string, snapEntryId uuid.UUID, revisionId uuid.UUID) error {
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
			return errors.New("branches not supported yet")
		}

		// get all the tracks
		var track models.SnapTrack
		query := `
			SELECT id
			FROM snap_tracks
			WHERE snap_entry_id = $1 AND name = $2
		`
		err := sp.db.Get(&track, query, snapEntryId, trackForRelease)
		if err == sql.ErrNoRows {
			logrus.Error(err)
			return errors.New(cerrors.ResourceNotFound)
		} else if err != nil {
			logrus.Error(err)
			return errors.New(cerrors.DatabaseError)
		}

		// get all the channels
		var channel models.SnapChannel
		query = `
			SELECT *
			FROM snap_channels
			WHERE snap_entry_id = $1 AND name = $2 AND snap_track_id = $3
		`
		err = sp.db.Get(&channel, query, snapEntryId, channelForRelease, track.ID)
		if err == sql.ErrNoRows {
			logrus.Error(err)
			return errors.New(cerrors.ResourceNotFound)
		} else if err != nil {
			logrus.Error(err)
			return errors.New(cerrors.DatabaseError)
		}

		var revision models.SnapRevision
		query = `
			SELECT id
			FROM snap_revisions
			WHERE id = $1
		`
		err = sp.db.Get(&revision, query, revisionId)
		if err == sql.ErrNoRows {
			logrus.Error(err)
			return errors.New(cerrors.ResourceNotFound)
		} else if err != nil {
			logrus.Error(err)
			return errors.New(cerrors.DatabaseError)
		}

		query = `
			UPDATE snap_channels
			SET revision_id = $1
			WHERE id = $2
		`
		_, err = sp.db.Exec(query, revision.ID, channel.ID)
		if err == sql.ErrNoRows {
			logrus.Error(err)
			return errors.New(cerrors.ResourceNotFound)
		} else if err != nil {
			logrus.Error(err)
			return errors.New(cerrors.DatabaseError)
		}
	}

	return nil
}

func (sp *SnapsRepository) addRevision(snapEntry models.SnapEntry, size uint64) (*models.SnapRevision, error) {
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
	if err == sql.ErrNoRows {
		logrus.Error(err)
		return nil, errors.New(cerrors.ResourceNotFound)
	} else if err != nil {
		logrus.Error(err)
		return nil, errors.New(cerrors.DatabaseError)
	}

	return &snapRevision, nil
}

func (sp *SnapsRepository) addChannels(snapEntry models.SnapEntry, snapRevision models.SnapRevision, trackId uuid.UUID) error {
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
		if err == sql.ErrNoRows {
			logrus.Error(err)
			return errors.New(cerrors.ResourceNotFound)
		} else if err != nil {
			logrus.Error(err)
			return errors.New(cerrors.DatabaseError)
		}
	}

	return nil
}

func (sp *SnapsRepository) updateMeta(metaBytes *[]byte) error {
	snapMeta, err2 := snap.GetSnapMetaFromBytes(*metaBytes, "/tmp")
	if err2 != nil {
		logrus.Error(err2)
		return err2
	}
	logrus.Tracef("snapMeta: %+v", snapMeta)
	var snapEntry models.SnapEntry
	query := `
		SELECT *
		FROM snap_entries
		WHERE name = $1
	`
	err := sp.db.Get(&snapEntry, query, snapMeta.Name)
	if err == sql.ErrNoRows {
		logrus.Error(err)
		return errors.New(cerrors.ResourceNotFound)
	} else if err != nil {
		logrus.Error(err)
		return errors.New(cerrors.DatabaseError)
	}

	snapEntry.Type = sql.NullString{String: "app", Valid: true}
	if snapMeta.Type != "" {
		snapEntry.Type = sql.NullString{String: snapMeta.Type, Valid: snapMeta.Type != ""}
	} else {
		logrus.Warnf("Snap %s had an emtpy type from its metadata, using default '%s'", snapEntry.Name, snapEntry.Type.String)
	}
	snapEntry.Confinement = sql.NullString{String: snapMeta.Confinement, Valid: snapMeta.Confinement != ""}
	snapEntry.Base = sql.NullString{String: snapMeta.Base, Valid: snapMeta.Base != ""}

	query = `
		UPDATE snap_entries
		SET type = $1, confinement = $2, base = $3
		WHERE id = $4
	`
	_, err = sp.db.Exec(query, snapEntry.Type, snapEntry.Confinement, snapEntry.Base, snapEntry.ID)
	if err == sql.ErrNoRows {
		logrus.Error(err)
		return errors.New(cerrors.ResourceNotFound)
	} else if err != nil {
		logrus.Error(err)
		return errors.New(cerrors.DatabaseError)
	}

	return nil
}

// sqlx version
func (sp *SnapsRepository) GetEntriesByAccountId(accountId uuid.UUID, preloadAssociations []string) ([]*models.SnapEntry, error) {
	query := `
            SELECT * 
            FROM snap_entries 
            WHERE account_id = $1
        `
	var entries []*models.SnapEntry
	err := sp.db.Select(&entries, query, accountId)
	if err == sql.ErrNoRows {
		logrus.Error(err)
		return nil, errors.New(cerrors.ResourceNotFound)
	} else if err != nil {
		logrus.Error(err)
		return nil, errors.New(cerrors.DatabaseError)
	}

	for _, entry := range entries {

		switch {
		case slices.Contains(preloadAssociations, models.COMMENT):
			resp, err := sp.GetCommentsByEntryId(entry.ID)
			if err != nil {
				logrus.Error(err)
				return nil, err
			}
			entry.LatestComments = resp
			fallthrough

		case slices.Contains(preloadAssociations, models.REVISION):
			resp, err := sp.GetRevisionsByEntryId(entry.ID)
			if err != nil {
				logrus.Error(err)
				return nil, err
			}
			entry.Revisions = resp
			fallthrough

		case slices.Contains(preloadAssociations, models.UPLOAD):
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

func (sp *SnapsRepository) GetUploadsByEntryId(d uuid.UUID) ([]*models.SnapUpload, error) {
	query := `
			SELECT *
			FROM snap_uploads
			WHERE snap_entry_id = $1
		`
	var uploads []*models.SnapUpload
	err := sp.db.Select(&uploads, query, d)
	if err == sql.ErrNoRows {
		logrus.Error(err)
		return nil, errors.New(cerrors.ResourceNotFound)
	} else if err != nil {
		logrus.Error(err)
		return nil, errors.New(cerrors.DatabaseError)
	}
	return uploads, nil
}

func (sp *SnapsRepository) GetCommentsByEntryId(d uuid.UUID) ([]*models.SnapComment, error) {
	query := `
			SELECT *
			FROM snap_comments
			WHERE snap_entry_id = $1
		`
	var comments []*models.SnapComment
	err := sp.db.Select(&comments, query, d)
	if err == sql.ErrNoRows {
		logrus.Error(err)
		return nil, errors.New(cerrors.ResourceNotFound)
	} else if err != nil {
		logrus.Error(err)
		return nil, err
	}
	return comments, nil
}

/*
// This is the grom version trying sqlx version
func (sp *SnapsRepository) GetEntriesByAccountId(accountId uuid.UUID, preloadAssociations bool) ([]*models.SnapEntry, error) {
    var snaps []*models.SnapEntry
    var db *gorm.DB
    if preloadAssociations {
        db = sp.db.Model(&models.SnapEntry{}).
            Preload("").
            Where(&models.SnapEntry{AccountID: accountId}).
            Find(&snaps)
    } else {
        db = sp.db.Model(&models.SnapEntry{}).
            Where(&models.SnapEntry{AccountID: accountId}).
            Find(&snaps)
    }

    if _, ok := database.CheckDBForErrorOrNoRows(db); ok {
        return snaps, nil
    }

    if db.Error != nil {
        return nil, db.Error
    }

    logrus.Errorf("Could not find snaps for accountId: %s", accountId)
    return nil, nil
}

*/

func (sp *SnapsRepository) GetRevisionsByEntryId(entryId uuid.UUID) ([]*models.SnapRevision, error) {
	var revisions []*models.SnapRevision
	query := `
			SELECT *
			FROM snap_revisions
			WHERE snap_entry_id = $1
		`
	err := sp.db.Select(&revisions, query, entryId)
	if err == sql.ErrNoRows {
		logrus.Error(err)
		return nil, errors.New(cerrors.ResourceNotFound)
	} else if err != nil {
		logrus.Error(err)
		return nil, errors.New(cerrors.DatabaseError)
	}

	return revisions, nil
}

func (sp *SnapsRepository) getSnap(whereModel *models.SnapEntry, preloadAssociations []string) (*models.SnapEntry, error) {
	var snapEntry models.SnapEntry
	query := `
		SELECT *
		FROM snap_entries
		WHERE id = $1
	`
	err := sp.db.Get(&snapEntry, query, whereModel.ID)
	if err == sql.ErrNoRows {
		logrus.Error(err)
		return nil, errors.New(cerrors.ResourceNotFound)
	} else if err != nil {
		logrus.Error(err)
		return nil, errors.New(cerrors.DatabaseError)
	}

	switch {
	case slices.Contains(preloadAssociations, models.COMMENT):
		resp, err := sp.GetCommentsByEntryId(snapEntry.ID)
		if err != nil {
			logrus.Error(err)
			return nil, err
		}
		snapEntry.LatestComments = resp
		fallthrough

	case slices.Contains(preloadAssociations, models.REVISION):
		resp, err := sp.GetRevisionsByEntryId(snapEntry.ID)
		if err != nil {
			logrus.Error(err)
			return nil, err
		}
		snapEntry.Revisions = resp
		fallthrough

	case slices.Contains(preloadAssociations, models.UPLOAD):
		resp, err := sp.GetUploadsByEntryId(snapEntry.ID)
		if err != nil {
			logrus.Error(err)
			return nil, err
		}
		snapEntry.Uploads = resp
	}

	return &snapEntry, nil
}
