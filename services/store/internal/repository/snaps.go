package repository

import (
	"fmt"
	"slices"

	"github.com/sirupsen/logrus"

	"github.com/google/uuid"

	"github.com/idlab-discover/kebeng/services/store/internal/models"
	"github.com/jmoiron/sqlx"

	cerror "github.com/idlab-discover/kebeng/common/cerror"
)

type ISnapsRepository interface {
	// CREATE
	AddChannel(snapEntryId uuid.UUID, snapTrackId uuid.UUID, channelName string, errorList *cerror.ErrorList) (*models.SnapChannel, *cerror.CustomError)
	AddDefaultChannels(snapEntryId uuid.UUID, snapTrackId uuid.UUID, errorList *cerror.ErrorList) *cerror.CustomError
	AddRevision(entryId uuid.UUID, trackId uuid.UUID, channelId uuid.UUID, snapName string, size uint64, sequenceNumber uint32, architectures []string, sha3_384_encoded string, minioFilePath string, errorList *cerror.ErrorList) (*models.SnapRevision, *cerror.CustomError)
	AddTrack(entryId uuid.UUID, trackName string, errorList *cerror.ErrorList) (*models.SnapTrack, *cerror.CustomError)
	AddUpload(entryId uuid.UUID, accountId uuid.UUID, snapName string, status string, unscannedFileName string, revision uint32, errorList *cerror.ErrorList) (*models.SnapUpload, *cerror.CustomError)
	RegisterSnap(snapName string, snapType string, confinement string, base string, isPrivate bool, status string, price float64, storeName string, iconURL string, accountId uuid.UUID, errorList *cerror.ErrorList) (*models.SnapEntry, *cerror.CustomError)

	// READ
	GetAllSnapEntries(errorList *cerror.ErrorList) (*[]models.SnapEntry, *cerror.CustomError)
	GetChannelById(id uuid.UUID, errorList *cerror.ErrorList) (*models.SnapChannel, *cerror.CustomError)
	GetChannelByTrackIdAndName(trackId uuid.UUID, channelName string, errorList *cerror.ErrorList) (*models.SnapChannel, *cerror.CustomError)
	GetChannelsByTrackId(trackId uuid.UUID, errorList *cerror.ErrorList) ([]*models.SnapChannel, *cerror.CustomError)
	GetCommentsByEntryId(entryId uuid.UUID, errorList *cerror.ErrorList) ([]*models.SnapComment, *cerror.CustomError)
	GetEntriesByAccountId(accountId uuid.UUID, preloadAssociations []string, errorList *cerror.ErrorList) ([]*models.SnapEntry, *cerror.CustomError)
	GetEntryById(id uuid.UUID, preloadAssociations []string, errorList *cerror.ErrorList) (*models.SnapEntry, *cerror.CustomError)
	GetEntryByName(name string, preloadAssociations []string, errorList *cerror.ErrorList) (*models.SnapEntry, *cerror.CustomError)
	GetLatestRevisionByEntryId(entryId uuid.UUID, errorList *cerror.ErrorList) (*models.SnapRevision, *cerror.CustomError)
	GetLatestRevisionByTrackAndChannel(snapName string, track string, channel string, errorList *cerror.ErrorList) (*models.SnapRevision, *cerror.CustomError)
	GetPreloadAssociations(entry *models.SnapEntry, preloadAssociations *[]string, errorList *cerror.ErrorList) *cerror.CustomError
	GetRevisionsByEntryId(entryId uuid.UUID, errorList *cerror.ErrorList) ([]*models.SnapRevision, *cerror.CustomError)
	GetRevisionById(id uuid.UUID, errorList *cerror.ErrorList) (*models.SnapRevision, *cerror.CustomError)
	GetRevisionByNameAndSequence(name string, sequence uint32, errorList *cerror.ErrorList) (*models.SnapRevision, *cerror.CustomError)
	GetRevisionBySHA(SHA3_384_encoded string, errorList *cerror.ErrorList) (*models.SnapRevision, *cerror.CustomError)
	GetTrackByEntryIdAndName(entryId uuid.UUID, trackName string, errorList *cerror.ErrorList) (*models.SnapTrack, *cerror.CustomError)
	GetTracksByEntryId(snapId uuid.UUID, errorList *cerror.ErrorList) ([]*models.SnapTrack, *cerror.CustomError)
	GetTrackById(id uuid.UUID, errorList *cerror.ErrorList) (*models.SnapTrack, *cerror.CustomError)
	GetUploadById(id uuid.UUID, errorList *cerror.ErrorList) (*models.SnapUpload, *cerror.CustomError)

	// UPDATE
	UpdateUploadStatus(uploadId uuid.UUID, status string, revision uint32, errorList *cerror.ErrorList) *cerror.CustomError
	UpdateSnapEntryWithMetadata(entryId uuid.UUID, metadata *models.SnapMeta, errorList *cerror.ErrorList) (*models.SnapEntry, *cerror.CustomError)
}

type SnapsRepository struct {
	db *sqlx.DB
}

func NewSnapsRepository(db *sqlx.DB) ISnapsRepository {
	return &SnapsRepository{db: db}
}

// ============ CREATE =============

func (sp *SnapsRepository) AddChannel(snapEntryId uuid.UUID, snapTrackId uuid.UUID, channelName string, el *cerror.ErrorList) (*models.SnapChannel, *cerror.CustomError) {
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
		cerr := cerror.ConvertError(err, fmt.Sprintf("error adding channel '%s' for snap with id = '%s'", channelName, snapEntryId.String()))
		logrus.Error(cerr)
		el.AddCustomError(cerr)
		return nil, cerr
	}

	return &channel, nil
}

func (sp *SnapsRepository) AddDefaultChannels(snapEntryId uuid.UUID, snapTrackId uuid.UUID, el *cerror.ErrorList) *cerror.CustomError {
	channels := []string{"stable", "candidate", "beta", "edge"}

	for _, channel := range channels {
		_, cerr := sp.AddChannel(snapEntryId, snapTrackId, channel, el)
		if cerr != nil {
			return cerr
		}
	}

	return nil
}

func (sp *SnapsRepository) AddRevision(entryId uuid.UUID, trackId uuid.UUID, channelId uuid.UUID, snapName string, size uint64, sequenceNumber uint32, architectures []string, sha3_384_encoded string, minioFilePath string, el *cerror.ErrorList) (*models.SnapRevision, *cerror.CustomError) {
	// TODO: fix the need for an empty revision
	// TODO: add build_assertion_filename if an assertion exists -> doesn't get checked in official snap store either (as of 2021)
	snapRevision := models.SnapRevision{
		SnapName:               snapName,
		BuildAssertionFileName: "", // TODO: add this if an assertion exists
		SHA3_384_Encoded:       sha3_384_encoded,
		Size:                   size,
		SequenceNumber:         sequenceNumber,
		Architectures:          architectures,
		MinioFilePath:          minioFilePath,
		SnapEntryID:            entryId,
		SnapTrackID:            trackId,
		SnapChannelID:          channelId,
		SnapBranchID:           uuid.Nil, // TODO: add this once branches are implemented
	}
	query := `
		INSERT INTO revision (entry_id, snap_track_id, snap_channel_id, snap_name, build_assertion_filename ,sha3_384_encoded, size, sequence_number, architectures, minio_file_path)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id
	`
	err := sp.db.Get(&snapRevision.ID, query, snapRevision.SnapEntryID, snapRevision.SnapTrackID, snapRevision.SnapChannelID, snapRevision.SnapName, snapRevision.BuildAssertionFileName, snapRevision.SHA3_384_Encoded, snapRevision.Size, snapRevision.SequenceNumber, snapRevision.Architectures, snapRevision.MinioFilePath)
	if err != nil {
		cerr := cerror.ConvertError(err, fmt.Sprintf("error adding revision for snap with id = '%s'", entryId.String()))
		logrus.Error(cerr)
		el.AddCustomError(cerr)
		return nil, cerr
	}

	return &snapRevision, nil
}

func (sp *SnapsRepository) AddTrack(entryId uuid.UUID, trackName string, el *cerror.ErrorList) (*models.SnapTrack, *cerror.CustomError) {
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
		cerr := cerror.ConvertError(err, fmt.Sprintf("error adding track '%s' for snap with id = '%s'", trackName, entryId.String()))
		logrus.Error(cerr)
		el.AddCustomError(cerr)
		return nil, cerr
	}

	return &track, nil

}

func (sp *SnapsRepository) AddUpload(entryId uuid.UUID, accountId uuid.UUID, snapName, status, unscannedFileName string, revision uint32, el *cerror.ErrorList) (*models.SnapUpload, *cerror.CustomError) {
	upload := models.SnapUpload{
		EntryID:           entryId,
		AccountID:         accountId,
		UnscannedFileName: unscannedFileName,
		SnapName:          snapName,
		Status:            status,
		Revision:          revision,
	}
	query := `
		INSERT INTO upload (snap_name, entry_id, status, account_id, unscanned_file_name, revision)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`
	err := sp.db.Get(&upload, query, upload.SnapName, upload.EntryID, upload.Status, upload.AccountID, upload.UnscannedFileName, upload.Revision)
	if err != nil {
		cerr := cerror.ConvertError(err, fmt.Sprintf("error adding upload for snap with id = '%s'", entryId.String()))
		logrus.Error(cerr)
		el.AddCustomError(cerr)
		return nil, cerr
	}

	return &upload, nil
}

// QUESTION: maybe we can just internaly call this AddEntry -> clearer name?
// QUESTION: right now an snap entry is bound to an account. Wouldn't it be better to bound snap revisions to an account?
func (sp *SnapsRepository) RegisterSnap(snapName string, snapType string, confinement string, base string, isPrivate bool, status string, price float64, storeName string, iconURL string, accountId uuid.UUID, el *cerror.ErrorList) (*models.SnapEntry, *cerror.CustomError) {
	snapEntry := models.SnapEntry{
		Name:        snapName,
		Type:        snapType,
		Confinement: confinement,
		Base:        base,
		Private:     isPrivate,
		Status:      status,
		Price:       price,
		Store:       storeName,
		IconURL:     iconURL,
		AccountID:   accountId,
	}

	query := `
		INSERT INTO entry (name, type, confinement, base, private, status, price, store, icon_url, account_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id
	`
	err := sp.db.Get(&snapEntry.ID, query, snapEntry.Name, snapEntry.Type, snapEntry.Confinement, snapEntry.Base, snapEntry.Private, snapEntry.Status, snapEntry.Price, snapEntry.Store, snapEntry.IconURL, snapEntry.AccountID)
	if err != nil {
		cerr := cerror.ConvertError(err, fmt.Sprintf("error adding snap entry with name = '%s'", snapName))
		logrus.Error(cerr)
		el.AddCustomError(cerr)
		return nil, cerr
	}

	return &snapEntry, nil
}

// ============ READ =============

func (sp *SnapsRepository) GetAllSnapEntries(el *cerror.ErrorList) (*[]models.SnapEntry, *cerror.CustomError) {
	var snaps []models.SnapEntry
	query := `
		SELECT *
		FROM entry
	`
	err := sp.db.Select(&snaps, query)
	if err != nil {
		cerr := cerror.ConvertError(err, "error getting all snaps")
		logrus.Error(cerr)
		el.AddCustomError(cerr)
		return nil, cerr
	}

	if len(snaps) == 0 {
		cerr := cerror.NewCustomError(cerror.ResourceNotFound, "resource not found: error getting all snaps")
		logrus.Error(cerr.GetMessage())
		el.AddCustomError(cerr)
		return nil, cerr
	}

	return &snaps, nil
}

func (sp *SnapsRepository) GetChannelByTrackIdAndName(trackId uuid.UUID, channelName string, errorList *cerror.ErrorList) (*models.SnapChannel, *cerror.CustomError) {
	var channel models.SnapChannel
	query := `
		SELECT *
		FROM channel
		WHERE snap_track_id = $1 AND name = $2
	`
	err := sp.db.Get(&channel, query, trackId, channelName)
	if err != nil {
		cerr := cerror.ConvertError(err, fmt.Sprintf("error getting channel with name '%s' for track with id = '%s'", channelName, trackId.String()))
		logrus.Error(cerr)
		errorList.AddCustomError(cerr)
		return nil, cerr
	}

	return &channel, nil
}

func (sp *SnapsRepository) GetChannelsByTrackId(trackId uuid.UUID, el *cerror.ErrorList) ([]*models.SnapChannel, *cerror.CustomError) {
	var channels []*models.SnapChannel
	query := `
		SELECT *
		FROM channel
		WHERE snap_track_id = $1
	`
	err := sp.db.Select(&channels, query, trackId)
	if err != nil {
		cerr := cerror.ConvertError(err, fmt.Sprintf("error getting channels for track with id = '%s'", trackId.String()))
		logrus.Error(cerr)
		el.AddCustomError(cerr)
		return nil, cerr
	}

	// manual check for empty result because db.Select doesn't return an error for empty results
	if len(channels) == 0 {
		cerr := cerror.NewCustomError(cerror.ResourceNotFound, fmt.Sprintf("resource not found: channels for track with id = '%s'", trackId.String()))
		logrus.Error(cerr.GetMessage())
		el.AddCustomError(cerr)
		return nil, cerr
	}

	return channels, nil
}

func (sp *SnapsRepository) GetCommentsByEntryId(entryId uuid.UUID, el *cerror.ErrorList) ([]*models.SnapComment, *cerror.CustomError) {
	var comments []*models.SnapComment
	query := `
			SELECT *
			FROM comment
			WHERE entry_id = $1
		`
	err := sp.db.Select(&comments, query, entryId)
	if err != nil {
		cerr := cerror.ConvertError(err, fmt.Sprintf("error getting comments for snap with id = '%s'", entryId.String()))
		logrus.Error(cerr)
		el.AddCustomError(cerr)
		return nil, cerr
	}

	// manual check for empty result because db.Select doesn't return an error for empty results
	if len(comments) == 0 {
		cerr := cerror.NewCustomError(cerror.ResourceNotFound, fmt.Sprintf("resource not found: comments for snap with id = '%s'", entryId.String()))
		logrus.Error(cerr.GetMessage())
		el.AddCustomError(cerr)
		return nil, cerr
	}

	return comments, nil
}

func (sp *SnapsRepository) GetEntriesByAccountId(accountId uuid.UUID, preloadAssociations []string, el *cerror.ErrorList) ([]*models.SnapEntry, *cerror.CustomError) {
	var entries []*models.SnapEntry
	query := `
            SELECT * 
            FROM entry 
            WHERE account_id = $1
        `
	err := sp.db.Select(&entries, query, accountId)
	if err != nil {
		cerr := cerror.ConvertError(err, fmt.Sprintf("error getting snaps for account with id = '%s'", accountId.String()))
		logrus.Error(cerr)
		el.AddCustomError(cerr)
		return nil, cerror.ConvertError(err)
	}

	// manual check for empty result because db.Select doesn't return an error for empty results
	if len(entries) == 0 {
		cerr := cerror.NewCustomError(cerror.ResourceNotFound, fmt.Sprintf("resource not found: snaps for account with id = '%s'", accountId.String()))
		logrus.Error(cerr.GetMessage())
		el.AddCustomError(cerr)
		return nil, cerr
	}

	for _, entry := range entries {
		cerr := sp.GetPreloadAssociations(entry, &preloadAssociations, el)
		if cerr != nil {
			return nil, cerr
		}
	}

	return entries, nil
}

func (sp *SnapsRepository) GetEntryById(Id uuid.UUID, preloadAssociations []string, el *cerror.ErrorList) (*models.SnapEntry, *cerror.CustomError) {
	var snapEntry models.SnapEntry
	query := `
		SELECT *
		FROM entry
		WHERE id = $1
	`
	err := sp.db.Get(&snapEntry, query, Id)
	if err != nil {
		cerr := cerror.ConvertError(err, fmt.Sprintf("error getting snap with id = '%s'", Id.String()))
		logrus.Error(cerr)
		el.AddCustomError(cerr)
		return nil, cerr
	}

	cerr := sp.GetPreloadAssociations(&snapEntry, &preloadAssociations, el)
	if cerr != nil {
		return nil, cerr
	}

	return &snapEntry, nil
}

func (sp *SnapsRepository) GetEntryByName(name string, preloadAssociations []string, el *cerror.ErrorList) (*models.SnapEntry, *cerror.CustomError) {
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
		cerr := cerror.ConvertError(err, fmt.Sprintf("error getting snap with name = '%s'", name))
		logrus.Error(cerr)
		el.AddCustomError(cerr)
		return nil, cerr
	}

	cerr := sp.GetPreloadAssociations(&snapEntry, &preloadAssociations, el)
	if cerr != nil {
		return nil, cerr
	}

	return &snapEntry, nil
}

func (sp *SnapsRepository) GetRevisionsByEntryId(entryId uuid.UUID, el *cerror.ErrorList) ([]*models.SnapRevision, *cerror.CustomError) {
	var revisions []*models.SnapRevision
	query := `
			SELECT *
			FROM revision
			WHERE entry_id = $1
		`
	err := sp.db.Select(&revisions, query, entryId)
	if err != nil {
		cerr := cerror.ConvertError(err, fmt.Sprintf("error getting revisions for snap with id = '%s'", entryId.String()))
		logrus.Error(cerr)
		el.AddCustomError(cerr)
		return nil, cerror.ConvertError(err)
	}

	// manual check for empty result because db.Select doesn't return an error for empty results
	if len(revisions) == 0 {
		cerr := cerror.NewCustomError(cerror.ResourceNotFound, fmt.Sprintf("resource not found: revisions for snap with id = '%s'", entryId.String()))
		logrus.Error(cerr.GetMessage())
		el.AddCustomError(cerr)
		return nil, cerr
	}

	return revisions, nil
}

func (sp *SnapsRepository) GetRevisionById(id uuid.UUID, el *cerror.ErrorList) (*models.SnapRevision, *cerror.CustomError) {
	var revision models.SnapRevision
	query := `
		SELECT *
		FROM revision
		WHERE id = $1
	`
	err := sp.db.Get(&revision, query, id)
	if err != nil {
		cerr := cerror.ConvertError(err, fmt.Sprintf("error getting revision with id = '%s'", id.String()))
		logrus.Error(cerr)
		el.AddCustomError(cerr)
		return nil, cerr
	}

	return &revision, nil
}

func (sp *SnapsRepository) GetRevisionByNameAndSequence(name string, sequence uint32, el *cerror.ErrorList) (*models.SnapRevision, *cerror.CustomError) {
	var entry models.SnapEntry
	query := `
		SELECT id
		FROM entry
		WHERE name = $1
	`
	err := sp.db.Get(&entry, query, name)
	if err != nil {
		cerr := cerror.ConvertError(err, fmt.Sprintf("error getting snap with name = '%s'", name))
		logrus.Error(cerr)
		el.AddCustomError(cerr)
		return nil, cerr
	}

	var revision models.SnapRevision
	query = `
		SELECT *
		FROM revision
		WHERE entry_id = $1 AND sequence_number = $2
	`
	err = sp.db.Get(&revision, query, entry.ID, sequence)
	if err != nil {
		cerr := cerror.ConvertError(err, fmt.Sprintf("error getting revision with sequence = '%d' for snap with name = '%s'", sequence, name))
		logrus.Error(cerr)
		el.AddCustomError(cerr)
		return nil, cerr
	}

	return &revision, nil
}

func (sp *SnapsRepository) GetRevisionBySHA(SHA3_384_encoded string, el *cerror.ErrorList) (*models.SnapRevision, *cerror.CustomError) {
	var revision models.SnapRevision
	query := `
			SELECT *
			FROM revision
			WHERE sha3_384_encoded = $1
		`
	err := sp.db.Get(&revision, query, SHA3_384_encoded)
	if err != nil {
		cerr := cerror.ConvertError(err, fmt.Sprintf("error getting revision with sha3_384_encoded = '%s'", SHA3_384_encoded))
		logrus.Error(cerr)
		el.AddCustomError(cerr)
		return nil, cerr
	}

	return &revision, nil
}

func (sp *SnapsRepository) GetTracksByEntryId(snapId uuid.UUID, el *cerror.ErrorList) ([]*models.SnapTrack, *cerror.CustomError) {
	var tracks []*models.SnapTrack

	query := `
		SELECT *
		FROM track
		WHERE entry_id = $1
	`
	err := sp.db.Select(&tracks, query, snapId)
	if err != nil {
		cerr := cerror.ConvertError(err, fmt.Sprintf("error getting tracks for snap with id = '%s'", snapId.String()))
		logrus.Error(cerr)
		el.AddCustomError(cerr)
		return nil, cerr
	}

	if len(tracks) == 0 {
		cerr := cerror.NewCustomError(cerror.ResourceNotFound, fmt.Sprintf("resource not found: tracks for snap with id = '%s'", snapId.String()))
		logrus.Error(cerr.GetMessage())
		el.AddCustomError(cerr)
		return nil, cerr
	}

	return tracks, nil
}

func (sp *SnapsRepository) GetLatestRevisionByTrackAndChannel(snapName string, track string, channel string, el *cerror.ErrorList) (*models.SnapRevision, *cerror.CustomError) {
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
		ORDER BY r.sequence_number DESC
		LIMIT 1;
	`
	err := sp.db.Get(&revision, query, snapName, track, channel)
	if err != nil {
		cerr := cerror.ConvertError(err, fmt.Sprintf("error getting latest revision for snap with name = '%s' and track = '%s' and channel = '%s'", snapName, track, channel))
		logrus.Error(cerr)
		el.AddCustomError(cerr)
		return nil, cerr
	}

	return &revision, nil
}

func (sp *SnapsRepository) GetLatestRevisionByEntryId(entryId uuid.UUID, el *cerror.ErrorList) (*models.SnapRevision, *cerror.CustomError) {
	var revision models.SnapRevision
	query := `
		SELECT *
		FROM revision
		WHERE entry_id = $1
		ORDER BY updated_at DESC
		LIMIT 1
	`
	err := sp.db.Get(&revision, query, entryId)
	if err != nil {
		cerr := cerror.ConvertError(err, fmt.Sprintf("error getting latest revision for snap with id = '%s'", entryId.String()))
		logrus.Error(cerr)
		el.AddCustomError(cerr)
		return nil, cerr
	}

	return &revision, nil
}

func (sp *SnapsRepository) GetTrackByEntryIdAndName(entryId uuid.UUID, trackName string, el *cerror.ErrorList) (*models.SnapTrack, *cerror.CustomError) {
	var track models.SnapTrack
	query := `
		SELECT *
		FROM track
		WHERE entry_id = $1 AND name = $2
	`
	err := sp.db.Get(&track, query, entryId, trackName)
	if err != nil {
		cerr := cerror.ConvertError(err, fmt.Sprintf("error getting track with name = '%s' for snap with id = '%s'", trackName, entryId.String()))
		logrus.Error(cerr)
		el.AddCustomError(cerr)
		return nil, cerr
	}

	return &track, nil
}

func (sp *SnapsRepository) GetTrackById(id uuid.UUID, el *cerror.ErrorList) (*models.SnapTrack, *cerror.CustomError) {
	var track models.SnapTrack
	query := `
		SELECT *
		FROM track
		WHERE id = $1
	`
	err := sp.db.Get(&track, query, id)
	if err != nil {
		cerr := cerror.ConvertError(err, fmt.Sprintf("error getting track with id = '%s'", id.String()))
		logrus.Error(cerr)
		el.AddCustomError(cerr)
		return nil, cerr
	}

	return &track, nil
}

func (sp *SnapsRepository) GetChannelById(id uuid.UUID, el *cerror.ErrorList) (*models.SnapChannel, *cerror.CustomError) {
	var channel models.SnapChannel
	query := `
		SELECT *
		FROM channel
		WHERE id = $1
	`
	err := sp.db.Get(&channel, query, id)
	if err != nil {
		cerr := cerror.ConvertError(err, fmt.Sprintf("error getting channel with id = '%s'", id.String()))
		logrus.Error(cerr)
		el.AddCustomError(cerr)
		return nil, cerr
	}

	return &channel, nil
}

func (sp *SnapsRepository) GetUploadById(id uuid.UUID, el *cerror.ErrorList) (*models.SnapUpload, *cerror.CustomError) {
	var upload models.SnapUpload
	query := `
		SELECT *
		FROM upload
		WHERE id = $1
	`
	err := sp.db.Get(&upload, query, id)
	if err != nil {
		cerr := cerror.ConvertError(err, fmt.Sprintf("error getting upload with id = '%s'", id.String()))
		logrus.Error(cerr)
		el.AddCustomError(cerr)
		return nil, cerr
	}

	return &upload, nil
}

// ============ UPDATE =============

func (sp *SnapsRepository) UpdateUploadStatus(uploadId uuid.UUID, status string, revision uint32, el *cerror.ErrorList) *cerror.CustomError {
	upload := models.SnapUpload{
		ID:     uploadId,
		Errors: el,
	}
	query := `
		UPDATE upload
		SET status = $1, revision = $2, errors = $3
		WHERE id = $4
		RETURNING *
	`
	err := sp.db.Get(&upload, query, status, revision, upload.Errors, uploadId)
	if err != nil {
		cerr := cerror.ConvertError(err, fmt.Sprintf("error updating upload with id = '%s'", uploadId.String()))
		logrus.Error(cerr)
		el.AddCustomError(cerr)
		return cerr
	}

	return nil
}

func (sp *SnapsRepository) UpdateSnapEntryWithMetadata(entryId uuid.UUID, metadata *models.SnapMeta, el *cerror.ErrorList) (*models.SnapEntry, *cerror.CustomError) {
	var snapEntry models.SnapEntry
	query := `
		UPDATE entry
		SET confinement = $1, base = $2, summary = $3, description = $4, architectures = $5, version = $6, grade = $7, updated_at = now()
		WHERE id = $8
		RETURNING *
	`
	err := sp.db.Get(&snapEntry, query, metadata.Confinement, metadata.Base, metadata.Summary, metadata.Description, metadata.Architectures, metadata.Version, metadata.Grade, entryId)
	if err != nil {
		cerr := cerror.ConvertError(err, fmt.Sprintf("error updating snap entry with id = '%s'", entryId.String()))
		logrus.Error(cerr)
		el.AddCustomError(cerr)
		return nil, cerr
	}
	return &snapEntry, nil
}

// ============ HELPER ==============

func (sp *SnapsRepository) GetPreloadAssociations(entry *models.SnapEntry, preloadAssociations *[]string, el *cerror.ErrorList) *cerror.CustomError {
	elLength := len(*el)
	all := slices.Contains(*preloadAssociations, models.ALL)

	switch {
	case all || slices.Contains(*preloadAssociations, models.COMMENT):
		resp, cerr := sp.GetCommentsByEntryId(entry.ID, el)
		if cerr != nil {
			// Already logged in GetCommentsByEntryId
		} else {
			entry.LatestComments = resp
		}
		fallthrough

	case all || slices.Contains(*preloadAssociations, models.TRACK):
		resp, cerr := sp.GetTracksByEntryId(entry.ID, el)
		if cerr != nil {
			// Already logged in GetTracksByEntryId
		} else {
			entry.Tracks = resp
		}
		fallthrough

	case all || slices.Contains(*preloadAssociations, models.CHANNEL):
		resp, cerr := sp.GetChannelsByTrackId(entry.ID, el)
		if cerr != nil {
			// Already logged in GetChannelsByTrackId
		} else {
			entry.Channels = resp
		}
		fallthrough

	case all || slices.Contains(*preloadAssociations, models.REVISION) || all:
		resp, cerr := sp.GetRevisionsByEntryId(entry.ID, el)
		if cerr != nil {
			// Already logged in GetRevisionsByEntryId
		} else {
			entry.Revisions = resp
		}
	}

	if elLength != len(*el) {
		logrus.Error("failed to preload associations")
		return cerror.NewCustomError(cerror.ResourceNotFound, "failed to preload associations")
	}

	return nil
}
