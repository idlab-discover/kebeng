package database

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/idlab-discover/kebeng/services/account/internal/models"
	"github.com/idlab-discover/kebeng/services/account/internal/repository"
	"github.com/sirupsen/logrus"
)

type TestData struct {
	Accounts []models.Account `json:"accounts"`
	Keys     []models.Key     `json:"keys"`
	SSHKeys  []models.SSHKey  `json:"ssh_keys"`
}

func LoadTestData(filePath string, repo repository.IAccountRepository) error {
	// check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		logrus.Warnf("Test data file does not exist: %s", filePath)
		return nil
	}
	file, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to open test data file: %v", err)
	}
	// Load the test data from the JSON file
	testData := &TestData{}
	err = json.Unmarshal(file, testData)
	if err != nil {
		return fmt.Errorf("failed to unmarshal test data: %v", err)
	}

	ctx := context.Background()
	for i, account := range testData.Accounts {
		logrus.Infof("Creating account: %+v", account)
		_, cerr := repo.CreateAccount(ctx, &testData.Accounts[i])
		if cerr != nil {
			return fmt.Errorf("failed to create account: %v", cerr)
		}
	}

	// NOTE: need accounts >= keys
	for i, key := range testData.Keys {
		if i < len(testData.Accounts) {
			_, cerr := repo.AddKeyToAccountByEmail(ctx, key.Name, key.SHA3384, key.EncodedPublicKey, testData.Accounts[i].Email)
			if cerr != nil {
				return fmt.Errorf("failed to create key: %v", cerr)
			}
		} else {
			logrus.Infof("Did not create key on index %d because there were only %d accounts", i, len(testData.Accounts))
		}
	}

	// not yet a function to insert ssh keys

	return nil
}
