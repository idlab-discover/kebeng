package repositories

import (
	"errors"
	"fmt"
	"strings"

	"gorm.io/gorm/clause"

	"github.com/idlab-discover/kebeng/pkg/snap"
	"github.com/sirupsen/logrus"

	"github.com/google/uuid"

	"github.com/idlab-discover/kebeng/pkg/database"
	"github.com/idlab-discover/kebeng/pkg/models"
	"gorm.io/gorm"
)

type ISnapsRepository interface {
	GetSnap(name string, preloadAssociations bool) (*models.SnapEntry, error)
	GetSnapById(id uuid.UUID, preloadAssociations bool) (*models.SnapEntry, error)
	AddSnap(name string, size uint64, accountId uuid.UUID) (*models.SnapEntry, error)

	GetRevisionBySHA(SHA3_384 string, encoded bool) (*models.SnapRevision, error)
	GetUploadByUpDownId(upDownId string) (*models.SnapUpload, error)
	UpdateRevision(revision *models.SnapRevision, revisionBytes *[]byte) (*models.SnapRevision, error)

	ReleaseSnap(channels []string, snapEntryId uuid.UUID, revisionId uint) error
	AddUpload(snapName string, upDownId string, size uint, channels []string) (*models.SnapUpload, error)

	SetChannelRevision(trackName string, riskName string, revisionId uint, snapId uuid.UUID) (*models.SnapTrack, error)

	GetTracks(snapId uuid.UUID) (*[]models.SnapTrack, error)
	GetRisks(trackId uint) (*[]models.SnapRisk, error)
	GetRevision(id uint) (*models.SnapRevision, error)
	GetRevisionByChannelAndTrack(channel string, snapName string) (*models.SnapRevision, error)

	GetSections() (*[]string, error)

	GetSnaps(filter *models.SnapFilter) ([]models.SnapEntry, error)
}

type SnapsRepository struct {
	db *gorm.DB
}

func NewSnapsRepository(db *gorm.DB) *SnapsRepository {
	return &SnapsRepository{db: db}
}

// TODO: Channel can have multiple revisions so not sure if this function is correct/useful check it 
func (sp *SnapsRepository) GetRevisionByChannelAndTrack(channel string, snapName string) (*models.SnapRevision, error) {
   snapEntry, err := sp.GetSnapByName(snapName, true)
   if err != nil {
      logrus.Errorf("Error getting snap by name %s: %v",snapName, err)
      return nil, err
   }
   if snapEntry == nil {
      logrus.Errorf("No snap found for name %s", snapName)
      return nil, errors.New("no snap found")
   }
   
   track, parsedChannel, err := parseTrackAndChannel(channel)
   if err != nil {
      logrus.Errorf("Error parsing track and channel: %v", err)
      return nil, err
   }

   // find SnapTrack
   var snapTrack models.SnapTrack
   err = sp.db.Where(&models.SnapTrack{
       SnapEntryID: snapEntry.ID,
       Name:        track,
   }).Find(&snapTrack).Error

   if err != nil {
       if errors.Is(err, gorm.ErrRecordNotFound) {
           return nil, fmt.Errorf("track %s not found for snap %s", track, snapName)
       }
       logrus.Errorf("Database error fetching snap track: %v", err)
       return nil, err
   }

   // find SnapChannel
   var snapChannel models.SnapChannel
   err = sp.db.Where(&models.SnapChannel{
       SnapEntryID: snapEntry.ID,
       Name:        parsedChannel,
       SnapTrackID: snapTrack.ID,
   }).Find(&snapChannel).Error

   if err != nil {
    if errors.Is(err, gorm.ErrRecordNotFound) {
        return nil, fmt.Errorf("channel %s not found for snap %s", parsedChannel, snapName)
    }
    logrus.Errorf("Database error fetching snap channel: %v", err)
    return nil, err
}

   return &snapChannel.Revision, nil
}

// not implemented yet
func (sp *SnapsRepository) GetSections() (*[]string, error) {
	// TODO: add these to the database for real
	sections := []string{
		"general",
	}

	return &sections, nil
}

func (sp *SnapsRepository) GetTracks(snapId uuid.UUID) (*[]models.SnapTrack, error) {
	var tracks []models.SnapTrack
	db := sp.db.Where(&models.SnapTrack{SnapEntryID: snapId}).Find(&tracks)
	if _, ok := database.CheckDBForErrorOrNoRows(db); ok {
		return &tracks, nil
	}

	if db.Error != nil {
		return nil, db.Error
	}

	logrus.Errorf("Could not find tracks for snapId: %d", snapId)
	return nil, errors.New("unknown error encountered")
}

func (sp *SnapsRepository) GetChannels(trackId uint) (*[]models.SnapChannel, error) {
	var channels []models.SnapChannel
	db := sp.db.Where(&models.SnapChannel{SnapTrackID: trackId}).Find(&channels)
	if _, ok := database.CheckDBForErrorOrNoRows(db); ok {
		return &channels, nil
	}

	if db.Error != nil {
		return nil, db.Error
	}

	logrus.Errorf("Could not find channels for track id: %d", trackId)
	return nil, errors.New("unknown error encountered")
}

func (sp *SnapsRepository) GetRevision(id uint) (*models.SnapRevision, error) {
	var revision models.SnapRevision
	db := sp.db.Where(&models.SnapRevision{Model: gorm.Model{ID: id}}).Find(&revision)
	if _, ok := database.CheckDBForErrorOrNoRows(db); ok {
		return &revision, nil
	}

	if db.Error != nil {
		return nil, db.Error
	}

	return nil, errors.New("unknown error encountered")
}

func (sp *SnapsRepository) SetChannelRevision(trackName string, riskName string, revisionId uint, snapId uuid.UUID) (*models.SnapTrack, error) {
	// get all the tracks
	var track models.SnapTrack
	db := sp.db.Where(&models.SnapTrack{SnapEntryID: snapId, Name: trackName}).Find(&track)
	if _, ok := database.CheckDBForErrorOrNoRows(db); ok {
		// get all the risks
		var risk models.SnapRisk
		db = sp.db.Where(&models.SnapRisk{SnapEntryID: snapId, Name: riskName, SnapTrackID: track.ID}).Find(&risk)
		if _, ok := database.CheckDBForErrorOrNoRows(db); ok {
			var revision models.SnapRevision
			db = sp.db.Where("id", revisionId).Find(&revision)
			if _, ok := database.CheckDBForErrorOrNoRows(db); ok {
				risk.RevisionID = revision.ID
				sp.db.Save(&risk)
				return &track, nil
			}
		} else {
			return nil, errors.New("risk does not exist for track")
		}
	} else {
		return nil, errors.New("track does not exist for snap")
	}

   // find SnapChannel
   var snapChannel models.SnapChannel
   err = sp.db.Where(&models.SnapChannel{
      SnapEntryID: snapId,
      Name:        riskName,
      SnapTrackID: snapTrack.ID,
   }).First(&snapChannel).Error

   if err != nil {
      if errors.Is(err, gorm.ErrRecordNotFound) {
         return nil, fmt.Errorf("channel %s not found for snap %d", riskName, snapId)
      }
      logrus.Errorf("Database error fetching snap channel: %v", err)
      return nil, err
   }
   
   // find SnapRevision
   var snapRevision models.SnapRevision
   err = sp.db.Where(&models.SnapRevision{
      Model: gorm.Model{ID: revisionId},
   }).First(&snapRevision).Error

   if err != nil {
      if errors.Is(err, gorm.ErrRecordNotFound) {
         return nil, fmt.Errorf("revision %d not found", revisionId)
      }
      logrus.Errorf("Database error fetching snap revision: %v", err)
      return nil, err
   }

   // update SnapChannel
   snapChannel.RevisionID = snapRevision.ID
   if err = sp.db.Save(&snapChannel).Error; err != nil {
      logrus.Errorf("Database error saving snap channel: %v", err)
      return nil, err
   }
   return &snapTrack, nil
}

func (sp *SnapsRepository) AddUpload(snapName string, upDownId string, fileSize uint, channels []string) (*models.SnapUpload, error) {
   // snapEntry created immediatly after the upload so should exist
   var snap models.SnapEntry
   err := sp.db.Where(&models.SnapEntry{Name: snapName}).First(&snap).Error
   if err != nil {
      if errors.Is(err, gorm.ErrRecordNotFound) {
         return nil, fmt.Errorf("snap %s not found", snapName)
      }
      logrus.Errorf("Database error fetching snap: %v", err)
      return nil, err
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

   logrus.Infof("Uploading: %+v", snapUpload)

   // can be assigned to multiple channels for example to both stable and beta
   // store in the database for historical records and easy querying if needed (no actually live uses i think)
   // TODO: fix lazy; this should be converted to a table so that the channels can be stored separately or maybe redis
   if len(channels) > 0 {
      snapUpload.Channels = strings.Join(channels, ",")
   }

   if err = sp.db.Save(&snapUpload).Error; err != nil {
      logrus.Errorf("Database error saving snap upload: %v", err)
      return nil, err
   }

   return &snapUpload, nil
}

func (sp *SnapsRepository) GetRevisionBySHA(SHA3_384 string, encoded bool) (*models.SnapRevision, error) {
	var revision models.SnapRevision
	var db *gorm.DB

	if encoded {
		logrus.Tracef("Getting snap revision by encoded sha3_384: %s", SHA3_384)
		db = sp.db.Where(&models.SnapRevision{SHA3384Encoded: SHA3_384}).Find(&revision)
	} else {
		logrus.Tracef("Getting snap revision by sha3_384: %s", SHA3_384)
		db = sp.db.Where(&models.SnapRevision{SHA3_384: SHA3_384}).Find(&revision)
	}

	// check for database errors or no rows found
	// could use the helper function CheckDBForErrorOrNoRows but don't like how it's implemented
	if db.Error != nil {
		logrus.Errorf("Database error: %v", db.Error)
		return nil, db.Error
	}
	if db.RowsAffected == 0 {
		logrus.Warnf("No revisions found for %s (encoded=%t)", SHA3_384, encoded)
		return nil, nil
	}
	return &revision, nil
}

func (sp *SnapsRepository) GetUploadByUpDownId(upDownId string) (*models.SnapUpload, error) {
	var snapUpload models.SnapUpload
	db := sp.db.Where(&models.SnapUpload{UpDownID: upDownId}).First(&snapUpload)
	if _, ok := database.CheckDBForErrorOrNoRows(db); ok {
		return &snapUpload, nil
	}

   return nil, fmt.Errorf("error fetching snap upload by upDownId %s: %v", upDownId, db.Error)
}

func (sp *SnapsRepository) UpdateRevision(revision *models.SnapRevision, revisionBytes *[]byte) (*models.SnapRevision, error) {
	db := sp.db.Save(revision)
	if db.Error == nil {
		err := sp.updateMeta(revisionBytes)
		if err != nil {
			logrus.Error(err)
			return nil, err
		}
		return revision, nil
	}
	return nil, db.Error
}

func (sp *SnapsRepository) GetSnaps(filter *models.SnapFilter) ([]models.SnapEntry, error) {
	var snaps []models.SnapEntry

	if err := sp.db.Scopes(models.SnapFilterScope(filter)).Find(&snaps).Error; err != nil {
		logrus.Errorf("Error retrieving snaps: %v", err)
		return nil, err
	}

   // If no records are found, GORM returns a nil error and snaps remains an empty slice.
	return snaps, nil
}

func (sp *SnapsRepository) GetSnapId(Id uuid.UUID, preloadAssociations bool) (*models.SnapEntry, error) {
	whereModel := &models.SnapEntry{ID: Id}
	return sp.getSnap(whereModel, preloadAssociations)
}

func (sp *SnapsRepository) GetSnapById(id uuid.UUID, preloadAssociations bool) (*models.SnapEntry, error) {
	var existingSnap models.SnapEntry
	var db *gorm.DB
	whereModel := &models.SnapEntry{ID: id}
	if preloadAssociations {
		db = sp.db.Preload(clause.Associations).Where(whereModel).Find(&existingSnap)
	} else {
		db = sp.db.Where(whereModel).Find(&existingSnap)
	}

	if _, ok := database.CheckDBForErrorOrNoRows(db); ok {
		return &existingSnap, nil
	}

	if db.Error != nil {
		return nil, db.Error
	}

   return &snap, nil
}

func (sp *SnapsRepository) GetSnapByName(name string, preloadAssociations bool) (*models.SnapEntry, error) {
	var snap models.SnapEntry

	query := sp.db
	if preloadAssociations {
		query = query.Preload(clause.Associations)
	}

	if err := query.Where(&models.SnapEntry{Name: name}).First(&snap).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			logrus.Warnf("Could not find snap %s", name)
			return nil, nil
		}
		logrus.Errorf("Error fetching snap by name %s: %v", name, err)
		return nil, err
	}

	return &snap, nil
}

// Used when a new snap gets uploaded for the first time (=registering a snap)

// AddSnap registers a new snap with the given name, size, and accountId.
// It ensures the snap does not already exist, creates a new SnapEntry,
// adds an initial upload, and sets up default tracks and risks.
func (sp *SnapsRepository) AddSnap(name string, size uint64, accountId uuid.UUID) (*models.SnapEntry, error) {
	existingSnap, err := sp.GetSnap(name, false)
	if err != nil {
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
	newSnapEntry.Type = "app"
	//newSnapEntry.Confinement = "strict"
	//newSnapEntry.Base = "core18" // default base

	// snap_entries table contains snaps with unique names (doesn't keep track of revisions or channels)
	sp.db.Save(&newSnapEntry)

	// snap_uploads table contains channels where the snap is uploaded
	sp.AddUpload(name, newSnapEntry.ID.String(), uint(size), []string{"latest/stable"})

	// For now when we register a snap we are going to create the default tracks/risks
	track := models.SnapTrack{
		Name:        "latest", // first upload of a snap is always the current latest
		SnapEntryID: newSnapEntry.ID,
	}

	sp.db.Save(&track)

	newRevision := sp.addRevision(newSnapEntry, size)

	sp.addRisks(newSnapEntry, *newRevision, track.ID)

	return &newSnapEntry, nil

}

func (sp *SnapsRepository) AddDefaultRisks(newSnapEntry models.SnapEntry, newRevision models.SnapRevision, trackId uint) {
	sp.addRisks(newSnapEntry, newRevision, trackId)
}

func (sp *SnapsRepository) ReleaseSnap(channels []string, snapEntryId uuid.UUID, revisionId uint) error {
	var trackForRelease string
	var riskForRelease string
	for _, cn := range channels {
		var trackForRelease, channelForRelease string

		// Parse the channel string.
		// Supported formats:
		//   "edge"            -> track defaults to "latest", channel = "edge"
		//   "latest/edge"     -> explicit track and channel
		//   "latest/edge/..." -> branches not supported
		parts := strings.Split(cn, "/")
		switch len(parts) {
		case 1:
			trackForRelease = "latest"
			channelForRelease = parts[0]
		case 2:
			trackForRelease = parts[0]
			channelForRelease = parts[1]
		case 3:
			return errors.New("branches not supported yet")
		default:
			return fmt.Errorf("invalid channel format: %s", cn)
		}

      // get tracks for snapEntryId
		var track models.SnapTrack
		if err := sp.db.
			Where(&models.SnapTrack{SnapEntryID: snapEntryId, Name: trackForRelease}).
			First(&track).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				logrus.Warnf("Track %s not found for snap %d", trackForRelease, snapEntryId)
				continue // check next channel if not found
			}
			return fmt.Errorf("error fetching track %s: %w", trackForRelease, err)
		}

      // get channels for snapEntryId, trackId, channelName
		var channel models.SnapChannel
		if err := sp.db.
			Where(&models.SnapChannel{SnapEntryID: snapEntryId, Name: channelForRelease, SnapTrackID: track.ID}).
			First(&channel).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				logrus.Warnf("Channel %s not found for snap %d on track %s", channelForRelease, snapEntryId, trackForRelease)
				continue 			
         }
			return fmt.Errorf("error fetching channel %s: %w", channelForRelease, err)
		}

      // get revision that is being released
		var revision models.SnapRevision
		if err := sp.db.First(&revision, revisionId).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				logrus.Warnf("Revision with id %d not found", revisionId)
				continue 
			}
			return fmt.Errorf("error fetching revision %d: %w", revisionId, err)
		}

      // update channel to point to new revision
		channel.RevisionID = revision.ID
		if err := sp.db.Save(&channel).Error; err != nil {
			return fmt.Errorf("error updating channel %s: %w", channelForRelease, err)
		}
	}

	return nil
}

func (sp *SnapsRepository) addRevision(snapEntry models.SnapEntry, size uint64) *models.SnapRevision {
	// TODO: fix the need for an empty revision
	snapRevision := models.SnapRevision{
		SnapFilename: snapEntry.Name,
		SnapEntryID:  snapEntry.ID,
		SHA3_384:     "",
		Size:         size,
	}

	sp.db.Save(&snapRevision)

	return &snapRevision
}

func (sp *SnapsRepository) addRisks(snapEntry models.SnapEntry, snapRevision models.SnapRevision, trackId uint) {
	// TODO: fix me
	risks := []string{"stable", "candidate", "beta", "edge"}

	for _, risk := range risks {
		var snapRisk models.SnapRisk
		snapRisk.SnapEntryID = snapEntry.ID
		snapRisk.SnapTrackID = trackId
		snapRisk.Name = risk

	for _, channel := range channels {
		snapChannel := models.SnapChannel{
			SnapEntryID: snapEntryId,
			SnapTrackID: trackId,
			Name:        channel,
			RevisionID:  snapRevision.ID,
		}

		if err := sp.db.Create(&snapChannel).Error; err != nil {
			return fmt.Errorf("failed to create snap channel '%s': %w", channel, err)
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
	db := sp.db.Where(&models.SnapEntry{Name: snapMeta.Name}).Find(&snapEntry)
	if _, ok := database.CheckDBForErrorOrNoRows(db); ok {
		snapEntry.Type = "app"
		if snapMeta.Type != "" {
			snapEntry.Type = snapMeta.Type
		} else {
			logrus.Warnf("Snap %s had an emtpy type from its metadata, using default '%s'", snapEntry.Name, snapEntry.Type)
		}

		snapEntry.Confinement = snapMeta.Confinement
		snapEntry.Base = snapMeta.Base

		sp.db.Save(&snapEntry)
	} else {
		logrus.Errorf("No rows found for: %s", snapMeta.Name)
	}
	return nil
}
func (sp *SnapsRepository) getSnapBySnapEntry(whereModel *models.SnapEntry, preloadAssociations bool) (*models.SnapEntry, error) {
	var snap models.SnapEntry
	query := sp.db

	if preloadAssociations {
		query = query.Preload(clause.Associations)
	}

	if err := query.Where(whereModel).First(&snap).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			logrus.Errorf("Could not find snap %+v", *whereModel)
			return nil, fmt.Errorf("snap entry not found for %+v", *whereModel)
		}
		return nil, err
	}

	return &snap, nil
}

func parseTrackAndChannel(channel string) (string, string, error) {
	channelParts := strings.Split(channel, "/")
	switch len(channelParts) {
	case 1:
		if channelParts[0] == "beta" || channelParts[0] == "edge" || channelParts[0] == "stable" || channelParts[0] == "candidate" {
			return "latest", channelParts[0], nil
		}
		return channelParts[0], "stable", nil
	case 2:
		return channelParts[0], channelParts[1], nil
	default:
		return "", "", errors.New("branches not yet supported for channels")
	}
}

