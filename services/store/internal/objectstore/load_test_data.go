package objectstore

import (
	"context"
	"os"

	"github.com/minio/minio-go/v7"
	"github.com/sirupsen/logrus"
)

func LoadTestData(client *minio.Client, minioPath string, snapPaths []string) error {
	// load an actual snap file here and upload it to minio
	ok, err := client.BucketExists(context.Background(), "snaps")
	if err != nil {
		logrus.Errorf("Error checking if bucket exists: %v", err)
		return err
	}
	if !ok {
		logrus.Errorf("Bucket does not exist")
		return err
	}

	logrus.Infof("snapPAths: %v", snapPaths)
	// Upload all files in the minioPath directory to the "snaps" bucket
	files, err := os.ReadDir(minioPath)
	if err != nil {
		logrus.Errorf("Error reading directory: %v", err)
		return err
	}
	for i, file := range files {
		if file.IsDir() {
			logrus.Errorf("Skipping directory: %s", file.Name())
			continue
		}
		if len(snapPaths) > 0 && i < len(snapPaths) {
			logrus.Infof("Using snap path: %s", snapPaths[i])
			filePath := minioPath + "/" + file.Name()
			_, err := client.FPutObject(context.Background(), "snaps", snapPaths[i], filePath, minio.PutObjectOptions{})
			if err != nil {
				logrus.Errorf("Error uploading file: %v", err)
				return err
			}
		}
	}
	return nil
}
