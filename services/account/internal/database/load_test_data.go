package database

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/idlab-discover/kebeng/services/account/internal/models"
	"github.com/idlab-discover/kebeng/services/account/internal/repository"
	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
)

type TestData struct {
	Accounts []models.Account `json:"accounts"`
	Keys     []models.Key     `json:"keys"`
	SSHKeys  []models.SSHKey  `json:"ssh_keys"`
}

func LoadTestData(filePath string, db *sqlx.DB, repo repository.IAccountRepository) error {
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
	for _, account := range testData.Accounts {
		logrus.Infof("Creating account: %+v", account)
		// sadly we need to decide what the account id has to be and that is not possible with the repo functionality
		// so we need to use the db directly
		_, err := db.ExecContext(ctx,
			`
			INSERT INTO account (id, display_name, username, email, password_hash, created_at, updated_at, validation)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			RETURNING id, display_name, username, email, password_hash, validation, created_at, updated_at, deleted_at
			`,
			account.ID, account.DisplayName, account.Username, account.Email, account.PasswordHash, account.CreatedAt, account.UpdatedAt, account.Validation)
		if err != nil {
			return fmt.Errorf("failed to create account: %v", err)
		}
		logrus.Infof("Created account: %+v", account)

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
