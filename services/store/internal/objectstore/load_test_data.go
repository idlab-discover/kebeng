package objectstore

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/idlab-discover/kebeng/common/cerror"
	"github.com/idlab-discover/kebeng/services/store/internal/models"
	"github.com/idlab-discover/kebeng/services/store/internal/repository"
	"github.com/minio/minio-go/v7"
	"github.com/sirupsen/logrus"
)

// TODO: should get snap path out of database instead of getting it as var
func (obs *ObjectStore) LoadTestData(client *minio.Client, repo repository.ISnapsRepository, minioPath string) error {
	el := cerror.NewErrorList()
	ok, err := client.BucketExists(context.Background(), "snaps")
	if err != nil {
		logrus.Errorf("Error checking if bucket exists: %v", err)
		return err
	}
	if !ok {
		logrus.Errorf("Bucket does not exist")
		return fmt.Errorf("bucket 'snaps' does not exist")
	}

	files, err := os.ReadDir(minioPath)
	if err != nil {
		logrus.Errorf("Error reading directory: %v", err)
		return err
	}

	for _, file := range files {
		if file.IsDir() {
			logrus.Infof("Skipping directory: %s", file.Name())
			continue
		}

		fileName := file.Name()
		if !strings.HasSuffix(fileName, ".snap") {
			logrus.Infof("Skipping non-snap file: %s", fileName)
			continue
		}

		snapName := strings.TrimSuffix(fileName, ".snap")
		snapEntry, errObj := repo.GetEntryByName(snapName, nil, el)
		if errObj != nil {
			logrus.Errorf("Could not find snap entry for %s: %v", snapName, errObj)
			continue
		}

		track, errObj := repo.GetTracksByEntryId(snapEntry.ID, el)
		if errObj != nil || len(track) == 0 {
			logrus.Errorf("No track found for snap entry %s", snapEntry.Name)
			continue
		}
		trackName := track[0].Name // use first track NOTE: this is a simplification, normally when generating we would know which track to use

		channels, errObj := repo.GetChannelsByTrackId(track[0].ID, el)
		if errObj != nil || len(channels) == 0 {
			logrus.Errorf("No channel found for track %s", trackName)
			continue
		}
		channelName := channels[0].Name // use first channel NOTE: this is a simplification, normally when generating we would know which channel to use

		revs, errObj := repo.GetRevisionsByEntryId(snapEntry.ID, el)
		if errObj != nil {
			logrus.Errorf("Could not get revisions for snap %s: %v", snapName, errObj)
			continue
		}

		var matchedRev *models.SnapRevision
		for _, rev := range revs {
			if rev.SnapTrackID == track[0].ID && rev.SnapChannelID == channels[0].ID {
				matchedRev = rev
				break
			}
		}
		if matchedRev == nil {
			logrus.Errorf("No revision matched snap name %s", snapName)
			continue
		}

		if matchedRev.SequenceNumber == nil {
			logrus.Errorf("Revision missing sequence number for %s", snapName)
			continue
		}

		s3Path := fmt.Sprintf("%s/%s/%s/%s_%d.snap",
			snapEntry.Name,
			trackName,
			channelName,
			snapEntry.Name,
			*matchedRev.SequenceNumber,
		)

		filePath := filepath.Join(minioPath, fileName)
		logrus.Infof("Uploading file %s to path %s", filePath, s3Path)
		_, err = client.FPutObject(context.Background(), "snaps", s3Path, filePath, minio.PutObjectOptions{})
		if err != nil {
			logrus.Errorf("Error uploading file %s: %v", filePath, err)
			return err
		}
	}
	return nil
}
