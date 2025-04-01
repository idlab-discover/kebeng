package database

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/idlab-discover/kebeng/services/assertion/internal/repositories"
	"github.com/sirupsen/logrus"
)

// NOTE: BRAM TODO WHEN CHANGE TO USE SQL INSTEAD OF GORM ALSO IMPLEMENT THIS
type TestData struct {
}

func LoadTestData(filePath string, repo repositories.IAssertionRepository) error {
	logrus.Info("inserting test data")
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		logrus.Warnf("Test data file does not exist: %s", filePath)
		return nil
	}
	file, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to open test data file: %v", err)
	}

	if len(file) == 0 {
		logrus.Info("test data file is empty")
		return nil
	}
	// Load the test data from the JSON file
	testData := &TestData{}
	err = json.Unmarshal(file, testData)
	if err != nil {
		return fmt.Errorf("failed to unmarshal test data: %v", err)
	}

	// not yet a function to insert ssh keys

	return nil
}
