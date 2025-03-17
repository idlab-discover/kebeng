package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/google/uuid"
	cerrors "github.com/idlab-discover/kebeng/services/account/internal/errors"
	"github.com/idlab-discover/kebeng/services/account/internal/models"
	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
)

// TODO: fix error handling better

// IAccountRepository defines the interface for account-related database operations
type IAccountRepository interface {
	CreateAccount(ctx context.Context, account *models.Account) (*models.Account, error)
	UpdateAccount(ctx context.Context, account *models.Account) (*models.Account, error)
	DeleteAccount(ctx context.Context, accountID uuid.UUID) error

	GetAccountByEmail(ctx context.Context, email string, associations []string) (*models.Account, error)
	GetAccountByID(ctx context.Context, accountID uuid.UUID, associations []string) (*models.Account, error)
	GetAccountByUsername(ctx context.Context, username string, associations []string) (*models.Account, error)

	AddKeyToAccountByEmail(ctx context.Context, name, sha3384, encodedPublicKey, accountEmail string) (*models.Key, error)
	GetKeyBySHA3384(ctx context.Context, sha3384 string) (*models.Key, error)
	GetKeysByAccountID(ctx context.Context, accountID uuid.UUID) ([]*models.Key, error)
	GetSSHKeysByAccountID(ctx context.Context, accountID uuid.UUID) ([]models.SSHKey, error)

	FilterKeys(ctx context.Context, whereModel *models.Key, takeFirst bool) (*models.Key, error)
	FilterAccounts(ctx context.Context, filter *models.Account, takeFirst bool) ([]*models.Account, error)
}

type AccountRepository struct {
	db *sqlx.DB
}

func NewAccountRepository(db *sqlx.DB) *AccountRepository {
	return &AccountRepository{db: db}
}

func (a *AccountRepository) CreateAccount(ctx context.Context, account *models.Account) (*models.Account, error) {
	query := `
        INSERT INTO accounts (id, username, email, password_hash, created_at, updated_at)
        VALUES (:id, :username, :email, :password_hash, :created_at, :updated_at)
    `
	rows, err := a.db.NamedQueryContext(ctx, query, account)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var newAccount models.Account
	if rows.Next() {
		if err := rows.StructScan(&newAccount); err != nil {
			return nil, err
		}
	} else {
		return nil, fmt.Errorf("could not create account %+v", account)
	}

	return account, nil
}

func (a *AccountRepository) UpdateAccount(ctx context.Context, account *models.Account) (*models.Account, error) {
	query := `
        UPDATE accounts
        SET display_name   = :display_name,
            username       = :username,
            email          = :email,
            password       = :password,
            validation     = :validation,
            updated_at     = :updated_at
        WHERE id = :id
        RETURNING id, display_name, username, email, password, validation, created_at, updated_at, deleted_at
    `
	rows, err := a.db.NamedQueryContext(ctx, query, account)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var updated models.Account
	if rows.Next() {
		if err := rows.StructScan(&updated); err != nil {
			return nil, err
		}
	} else {
		return nil, fmt.Errorf("no account updated")
	}
	return &updated, nil
}

func (a *AccountRepository) DeleteAccount(ctx context.Context, accountID uuid.UUID) error {
	query := `DELETE FROM accounts WHERE id = $1`
	_, err := a.db.ExecContext(ctx, query, accountID)
	return err
}

func (a *AccountRepository) GetAccountByEmail(ctx context.Context, email string, associations []string) (*models.Account, error) {
	var account models.Account
	query := `SELECT * FROM accounts WHERE email = $1`

	err := a.db.Get(&account, query, email)
	if err == sql.ErrNoRows {
		logrus.Error(err)
		return nil, errors.New(cerrors.ResourceNotFound)
	} else if err != nil {
		logrus.Error(err)
		return nil, errors.New(cerrors.DatabaseError)
	}

	// check if all associations are requested
	if slices.Contains(associations, models.ALL) {
		sshKeys, err := a.GetSSHKeysByAccountID(ctx, account.ID)
		if err != nil {
			return nil, err
		}
		account.SSHKeys = sshKeys
	} else {
		// if not everything is request loop over associations and get them
		// TODO: check what difference between sshkey and key
		for _, association := range associations {
			switch association {
			case models.SSHKEY:
				sshKeys, err := a.GetSSHKeysByAccountID(ctx, account.ID)
				if err != nil {
					logrus.Error(err)
					return nil, err
				}
				account.SSHKeys = sshKeys
			}
		}
	}

	return &account, nil
}

func (a *AccountRepository) GetAccountByID(ctx context.Context, accountID uuid.UUID, associations []string) (*models.Account, error) {
	var account models.Account
	query := `SELECT * FROM accounts WHERE id = $1`

	err := a.db.Get(&account, query, accountID)
	if err == sql.ErrNoRows {
		logrus.Error(err)
		return nil, errors.New(cerrors.ResourceNotFound)
	} else if err != nil {
		logrus.Error(err)
		return nil, errors.New(cerrors.DatabaseError)
	}

	// Check if all associations are requested.
	if slices.Contains(associations, models.ALL) {
		sshKeys, err := a.GetSSHKeysByAccountID(ctx, account.ID)
		if err != nil {
			return nil, err
		}
		account.SSHKeys = sshKeys
	} else {
		// Loop over associations and load each requested association.
		for _, association := range associations {
			switch association {
			case models.SSHKEY:
				sshKeys, err := a.GetSSHKeysByAccountID(ctx, account.ID)
				if err != nil {
					logrus.Error(err)
					return nil, err
				}
				account.SSHKeys = sshKeys
			}
		}
	}

	return &account, nil
}

func (a *AccountRepository) GetAccountByUsername(ctx context.Context, username string, associations []string) (*models.Account, error) {
	var account models.Account
	query := `SELECT * FROM accounts WHERE username = $1`

	err := a.db.Get(&account, query, username)
	if err == sql.ErrNoRows {
		logrus.Error(err)
		return nil, errors.New(cerrors.ResourceNotFound)
	} else if err != nil {
		logrus.Error(err)
		return nil, errors.New(cerrors.DatabaseError)
	}

	// Check if all associations are requested.
	if slices.Contains(associations, models.ALL) {
		sshKeys, err := a.GetSSHKeysByAccountID(ctx, account.ID)
		if err != nil {
			return nil, err
		}
		account.SSHKeys = sshKeys
	} else {
		// Loop over associations and load each requested association.
		for _, association := range associations {
			switch association {
			case models.SSHKEY:
				sshKeys, err := a.GetSSHKeysByAccountID(ctx, account.ID)
				if err != nil {
					logrus.Error(err)
					return nil, err
				}
				account.SSHKeys = sshKeys
			}
		}
	}

	return &account, nil
}

func (a *AccountRepository) AddKeyToAccountByEmail(ctx context.Context, name, sha3384, encodedPublicKey, email string) (*models.Key, error) {
	acct, err := a.GetAccountByEmail(ctx, email, []string{})
	if err != nil || acct == nil {
		return nil, fmt.Errorf("could not find account with email %s: err: %v", email, err)
	}

	// Create a new key value.
	newKey := models.Key{
		Name:             name,
		SHA3384:          sha3384,
		EncodedPublicKey: encodedPublicKey,
		AccountID:        acct.ID,
		Until:            time.Now().AddDate(1, 0, 0), // 1 year from now TODO: check what this value should be
	}

	query := `
        INSERT INTO keys (name, sha3384, encoded_public_key, account_id)
        VALUES (:name, :sha3384, :encoded_public_key, :account_id)
        RETURNING id, name, sha3384, encoded_public_key, account_id, until, created_at, updated_at, deleted_at
    `

	rows, err := a.db.NamedQueryContext(ctx, query, newKey)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var key models.Key
	if rows.Next() {
		if err := rows.StructScan(&key); err != nil {
			return nil, err
		}
	} else {
		return nil, fmt.Errorf("failed to insert key")
	}

	return &key, nil
}

func (a *AccountRepository) GetKeyBySHA3384(ctx context.Context, sha3384 string) (*models.Key, error) {
	var key models.Key
	query := `SELECT * FROM keys WHERE sha3384 = $1`

	err := a.db.GetContext(ctx, &key, query, sha3384)
	if err == sql.ErrNoRows {
		logrus.Error(err)
		return nil, errors.New(cerrors.ResourceNotFound)
	} else if err != nil {
		logrus.Error(err)
		return nil, errors.New(cerrors.DatabaseError)
	}

	return &key, nil
}

func (a *AccountRepository) GetKeysByAccountID(ctx context.Context, accountID uuid.UUID) ([]*models.Key, error) {
	var keys []*models.Key
	query := `SELECT * FROM keys WHERE account_id = $1`

	err := a.db.SelectContext(ctx, &keys, query, accountID)
	if err != nil {
		logrus.Error(err)
		return nil, errors.New(cerrors.DatabaseError)
	}

	if len(keys) == 0 {
		return nil, errors.New(cerrors.ResourceNotFound)
	}

	return keys, nil
}

func (a *AccountRepository) GetSSHKeysByAccountID(ctx context.Context, accountID uuid.UUID) ([]models.SSHKey, error) {
	var sshKeys []models.SSHKey
	query := "SELECT * FROM ssh_keys WHERE account_id = $1"
	if err := a.db.SelectContext(ctx, &sshKeys, query, accountID); err != nil {
		logrus.Error(err)
		return nil, err
	}
	return sshKeys, nil
}

// filter function that returns keys based on filter
func (a *AccountRepository) FilterKeys(ctx context.Context, whereModel *models.Key, takeFirst bool) (*models.Key, error) {
	// Start with a base query.
	query := "SELECT * FROM keys WHERE 1=1"
	params := make(map[string]interface{})

	if whereModel.ID != uuid.Nil {
		query += " AND id = :id"
		params["id"] = whereModel.ID
	}
	if whereModel.Name != "" {
		query += " AND name = :name"
		params["name"] = whereModel.Name
	}
	if whereModel.SHA3384 != "" {
		query += " AND sha3384 = :sha3384"
		params["sha3384"] = whereModel.SHA3384
	}
	if whereModel.EncodedPublicKey != "" {
		query += " AND encoded_public_key = :encoded_public_key"
		params["encoded_public_key"] = whereModel.EncodedPublicKey
	}
	if whereModel.AccountID != uuid.Nil {
		query += " AND account_id = :account_id"
		params["account_id"] = whereModel.AccountID
	}
	if !whereModel.Until.IsZero() {
		query += " AND until = :until"
		params["until"] = whereModel.Until
	}
	if !whereModel.CreatedAt.IsZero() {
		query += " AND created_at = :created_at"
		params["created_at"] = whereModel.CreatedAt
	}
	if !whereModel.UpdatedAt.IsZero() {
		query += " AND updated_at = :updated_at"
		params["updated_at"] = whereModel.UpdatedAt
	}
	if whereModel.DeletedAt != nil {
		query += " AND deleted_at = :deleted_at"
		params["deleted_at"] = whereModel.DeletedAt
	}

	if takeFirst {
		query += " LIMIT 1"
	}

	rows, err := a.db.NamedQueryContext(ctx, query, params)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var keys []*models.Key
	for rows.Next() {
		var key models.Key
		if err := rows.StructScan(&key); err != nil {
			return nil, err
		}
		keys = append(keys, &key)
	}

	if len(keys) == 0 {
		return nil, fmt.Errorf("no key found with filter %+v", whereModel)
	}

	return keys[0], nil
}

// filter function that returns accounts based on filter
func (a *AccountRepository) FilterAccounts(ctx context.Context, filter *models.Account, takeFirst bool) ([]*models.Account, error) {
	// do WHERE 1=1 to have base query to append to
	query := "SELECT * FROM accounts WHERE 1=1"
	params := make(map[string]interface{})

	if filter.ID != uuid.Nil {
		query += " AND id = :id"
		params["id"] = filter.ID
	}
	if filter.DisplayName != "" {
		query += " AND display_name = :display_name"
		params["display_name"] = filter.DisplayName
	}
	if filter.Username != "" {
		query += " AND username = :username"
		params["username"] = filter.Username
	}
	if filter.Email != "" {
		query += " AND email = :email"
		params["email"] = filter.Email
	}
	if !filter.CreatedAt.IsZero() {
		query += " AND created_at = :created_at"
		params["created_at"] = filter.CreatedAt
	}
	if !filter.UpdatedAt.IsZero() {
		query += " AND updated_at = :updated_at"
		params["updated_at"] = filter.UpdatedAt
	}
	if filter.DeletedAt != nil {
		query += " AND deleted_at = :deleted_at"
		params["deleted_at"] = filter.DeletedAt
	}
	if filter.Validation != "" {
		query += " AND validation = :validation"
		params["validation"] = filter.Validation
	}

	if takeFirst {
		query += " LIMIT 1"
	}

	rows, err := a.db.NamedQueryContext(ctx, query, params)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var accounts []*models.Account
	for rows.Next() {
		var account models.Account
		if err := rows.StructScan(&account); err != nil {
			return nil, err
		}
		accounts = append(accounts, &account)
	}

	if len(accounts) == 0 {
		return nil, fmt.Errorf("no accounts found with filter %+v", filter)
	}

	return accounts, nil
}
