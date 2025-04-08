package snap

import (
	"os"
	"os/exec"
	"path"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

type SnapMeta struct {
	Name          string   `yaml:"name"`
	Version       string   `yaml:"version"`
	Summary       string   `yaml:"summary"`
	Description   string   `yaml:"description"`
	Type          string   `yaml:"type"`
	Architectures []string `yaml:"architectures"`
	Confinement   string   `yaml:"confinement"`
	Grade         string   `yaml:"grade"`
	Base          string   `yaml:"base"`
}

// GetSnapMetaFromFile will return SnapMeta from a byte array representing a snap file
// This is an inefficient but expedient process
func GetSnapMetaFromFile(snapFilePath string, workingDirectory string) (*SnapMeta, error) {
	bytes, err := os.ReadFile(snapFilePath)
	if err == nil {
		return GetSnapMetaFromBytes(bytes, workingDirectory)
	}

	return nil, err
}

func GetSnapMetaFromBytes(bytes []byte, workingDirectory string) (*SnapMeta, error) {
	tmpFilePath := path.Join(workingDirectory, uuid.New().String()+".snap")
	err := os.WriteFile(tmpFilePath, bytes, 0755)
	if err != nil {
		logrus.Errorf("error writing file: %s", err)
		return nil, err
	}

	logrus.Infof("Temporary file path: %s", tmpFilePath)

	// Log the first few bytes of the file to verify its contents
	logrus.Infof("File contents (first 10 bytes): %x", bytes[:10])

	defer func(name string) {
		errIn := os.Remove(name)
		if errIn != nil {
			logrus.Errorf("error deleting temporary file, %s", errIn)
		}
	}(tmpFilePath)

	err = os.Chdir(workingDirectory)
	if err != nil {
		logrus.Errorf("error changing working directory: %s", err)
		return nil, err
	}

	logrus.Infof("Working directory: %s", workingDirectory)

	cmd := exec.Command("unsquashfs", tmpFilePath, "-e", "meta/snap.yaml")
	defer func() {
		errIn := os.RemoveAll(path.Join(workingDirectory, "squashfs-root"))
		if errIn != nil {
			logrus.Errorf("error removing directory: %s", err)
		}
	}()
	cmd.Stderr = os.Stderr

	err = cmd.Run()
	if err != nil {
		logrus.Error(err)
		return nil, err
	}

	bytes, err = os.ReadFile(path.Join(workingDirectory, "squashfs-root", "meta", "snap.yaml"))
	if err != nil {
		logrus.Errorf("error reading file: %s", err)
		return nil, err
	}

	var snapMeta SnapMeta
	err = yaml.Unmarshal(bytes, &snapMeta)
	if err != nil {
		logrus.Error(err)
		return nil, err
	}

	return &snapMeta, nil
}
