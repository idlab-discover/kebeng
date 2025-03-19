package repository

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/google/uuid"
	cerror "github.com/idlab-discover/kebeng/common/cerror"
	"github.com/idlab-discover/kebeng/services/account/internal/models"
	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
)

// IAccountRepository defines the interface for account-related database operations
type IAccountRepository interface {
	CreateAccount(ctx context.Context, account *models.Account) (*models.Account, *cerror.CustomError)
	UpdateAccount(ctx context.Context, account *models.Account) (*models.Account, *cerror.CustomError)
	DeleteAccount(ctx context.Context, accountID uuid.UUID) *cerror.CustomError

	GetAccountByEmail(ctx context.Context, email string, associations []string) (*models.Account, *cerror.CustomError)
	GetAccountByID(ctx context.Context, accountID uuid.UUID, associations []string) (*models.Account, *cerror.CustomError)
	GetAccountByUsername(ctx context.Context, username string, associations []string) (*models.Account, *cerror.CustomError)

	AddKeyToAccountByEmail(ctx context.Context, name, sha3384, encodedPublicKey, accountEmail string) (*models.Key, *cerror.CustomError)
	GetKeyBySHA3384(ctx context.Context, sha3384 string) (*models.Key, *cerror.CustomError)
	GetKeysByAccountID(ctx context.Context, accountID uuid.UUID) ([]*models.Key, *cerror.CustomError)
	GetSSHKeysByAccountID(ctx context.Context, accountID uuid.UUID) ([]models.SSHKey, *cerror.CustomError)

	FilterKeys(ctx context.Context, filter *models.Key, takeFirst bool) (*models.Key, *cerror.CustomError)
	FilterAccounts(ctx context.Context, filter *models.Account, takeFirst bool) ([]*models.Account, *cerror.CustomError)
}

type AccountRepository struct {
	db *sqlx.DB
}

func NewAccountRepository(db *sqlx.DB) *AccountRepository {
	return &AccountRepository{db: db}
}

func (a *AccountRepository) CreateAccount(ctx context.Context, account *models.Account) (*models.Account, *cerror.CustomError) {
	query := `
        INSERT INTO accounts (display_name, username, email, password_hash, updated_at)
        VALUES (:display_name, :username, :email, :password_hash, :updated_at)
        RETURNING id, display_name, username, email, password_hash, validation, created_at, updated_at, deleted_at
    `
	rows, err := a.db.NamedQueryContext(ctx, query, account)
	if err != nil {
		logrus.Error(err)
		return nil, cerror.ConvertError(err, fmt.Sprintf("could not create account with email '%s'", account.Email))
	}
	defer rows.Close()

	var newAccount models.Account
	if rows.Next() {
		if err := rows.StructScan(&newAccount); err != nil {
			logrus.Error(err)
			return nil, cerror.ConvertError(err)
		}
	} else {
		logrus.Error(err)
		return nil, cerror.NewCustomError(cerror.DatabaseError, fmt.Sprintf("failed to insert account with email '%s'", account.Email))
	}

	return account, nil
}

func (a *AccountRepository) UpdateAccount(ctx context.Context, account *models.Account) (*models.Account, *cerror.CustomError) {
	query := `
        UPDATE accounts
        SET display_name    = :display_name,
            username        = :username,
            email           = :email,
            password_hash   = :password_hash,
            validation      = :validation,
            updated_at      = :updated_at
        WHERE id = :id
        RETURNING id, display_name, username, email, password_hash, validation, created_at, updated_at, deleted_at
    `
	rows, err := a.db.NamedQueryContext(ctx, query, account)
	if err != nil {
		logrus.Error(err)
		return nil, cerror.ConvertError(err)
	}
	defer rows.Close()

	var updated models.Account
	if rows.Next() {
		if err := rows.StructScan(&updated); err != nil {
			logrus.Error(err)
			return nil, cerror.ConvertError(err)
		}
	} else {
		logrus.Error(err)
		return nil, cerror.NewCustomError(cerror.DatabaseError, "failed to update account")
	}
	return &updated, nil
}

func (a *AccountRepository) DeleteAccount(ctx context.Context, accountID uuid.UUID) *cerror.CustomError {
	query := `DELETE FROM accounts WHERE id = $1`
	_, err := a.db.ExecContext(ctx, query, accountID)
	if err != nil {
		logrus.Error(err)
		return cerror.ConvertError(err)
	}
	return nil
}

func (a *AccountRepository) GetAccountByEmail(ctx context.Context, email string, associations []string) (*models.Account, *cerror.CustomError) {
	var account models.Account
	query := `SELECT * FROM accounts WHERE email = $1`

	err := a.db.Get(&account, query, email)
	if err != nil {
		logrus.Error(err)
		return nil, cerror.ConvertError(err)
	}

	// TODO: check what difference between SSHKeys and Keys
	cerr := a.handleAssociations(ctx, &account, associations)
	if cerr != nil {
		return nil, cerr
	}

	return &account, nil
}

func (a *AccountRepository) GetAccountByID(ctx context.Context, accountID uuid.UUID, associations []string) (*models.Account, *cerror.CustomError) {
	var account models.Account
	query := `SELECT * FROM accounts WHERE id = $1`

	err := a.db.Get(&account, query, accountID)
	if err != nil {
		logrus.Error(err)
		return nil, cerror.ConvertError(err)
	}

	cerr := a.handleAssociations(ctx, &account, associations)
	if cerr != nil {
		return nil, cerr
	}

	return &account, nil
}

func (a *AccountRepository) GetAccountByUsername(ctx context.Context, username string, associations []string) (*models.Account, *cerror.CustomError) {
	var account models.Account
	query := `SELECT * FROM accounts WHERE username = $1`

	err := a.db.Get(&account, query, username)
	if err != nil {
		logrus.Error(err)
		return nil, cerror.ConvertError(err)
	}

	cerr := a.handleAssociations(ctx, &account, associations)
	if cerr != nil {
		return nil, cerr
	}

	return &account, nil
}

func (a *AccountRepository) AddKeyToAccountByEmail(ctx context.Context, name, sha3384, encodedPublicKey, email string) (*models.Key, *cerror.CustomError) {
	acct, cerr := a.GetAccountByEmail(ctx, email, []string{})
	if cerr != nil {
		return nil, cerr
	}

	// Create a new key value.
	until := time.Now().AddDate(1, 0, 0)
	newKey := models.Key{
		Name:             name,
		SHA3384:          sha3384,
		EncodedPublicKey: encodedPublicKey,
		AccountID:        acct.ID,
		Until:            &until, // 1 year from now TODO: check what this value should be
	}

	query := `
	INSERT INTO keys (name, sha3384, encoded_public_key, account_id, until)
	VALUES (:name, :sha3384, :encoded_public_key, :account_id, :until)
        RETURNING id, name, sha3384, encoded_public_key, account_id, until, created_at, updated_at, deleted_at
    `

	rows, err := a.db.NamedQueryContext(ctx, query, newKey)
	if err != nil {
		logrus.Error(err)
		return nil, cerror.ConvertError(err, fmt.Sprintf("could not add key with name '%s' for account id '%s'", name, acct.ID))
	}
	defer rows.Close()

	var key models.Key
	if rows.Next() {
		if err := rows.StructScan(&key); err != nil {
			logrus.Error(err)
			return nil, cerror.ConvertError(err)
		}
	} else {
		logrus.Errorf("failed to insert key for account id '%s'", acct.ID)
		return nil, cerror.NewCustomError(cerror.DatabaseError, "failed to insert key")
	}

	return &key, nil
}

func (a *AccountRepository) GetKeyBySHA3384(ctx context.Context, sha3384 string) (*models.Key, *cerror.CustomError) {
	var key models.Key
	query := `SELECT * FROM keys WHERE sha3384 = $1`

	err := a.db.GetContext(ctx, &key, query, sha3384)
	if err != nil {
		logrus.Error(err)
		return nil, cerror.ConvertError(err, fmt.Sprintf("resource not found by SHA3384, value = '%s'", sha3384))
	}

	return &key, nil
}

func (a *AccountRepository) GetKeysByAccountID(ctx context.Context, accountID uuid.UUID) ([]*models.Key, *cerror.CustomError) {
	var keys []*models.Key
	query := `SELECT * FROM keys WHERE account_id = $1`

	err := a.db.SelectContext(ctx, &keys, query, accountID)
	if err != nil {
		logrus.Error(err)
		return nil, cerror.ConvertError(err)
	}

	if len(keys) == 0 {
		logrus.Errorf("no keys found for account %s", accountID)
		return nil, cerror.NewCustomError(cerror.ResourceNotFound, "no keys found for account")
	}

	return keys, nil
}

func (a *AccountRepository) GetSSHKeysByAccountID(ctx context.Context, accountID uuid.UUID) ([]models.SSHKey, *cerror.CustomError) {
	var sshKeys []models.SSHKey
	query := "SELECT * FROM ssh_keys WHERE account_id = $1"
	if err := a.db.SelectContext(ctx, &sshKeys, query, accountID); err != nil {
		logrus.Error(err)
		return nil, cerror.ConvertError(err)
	}
	return sshKeys, nil
}

// filter function that returns keys based on filter
func (a *AccountRepository) FilterKeys(ctx context.Context, filter *models.Key, takeFirst bool) (*models.Key, *cerror.CustomError) {
	// start with base query
	query := "SELECT * FROM keys WHERE 1=1"
	params := make(map[string]interface{})

	if filter.ID != uuid.Nil {
		query += " AND id = :id"
		params["id"] = filter.ID
	}
	if filter.Name != "" {
		query += " AND name = :name"
		params["name"] = filter.Name
	}
	if filter.SHA3384 != "" {
		query += " AND sha3384 = :sha3384"
		params["sha3384"] = filter.SHA3384
	}
	if filter.EncodedPublicKey != "" {
		query += " AND encoded_public_key = :encoded_public_key"
		params["encoded_public_key"] = filter.EncodedPublicKey
	}
	if filter.AccountID != uuid.Nil {
		query += " AND account_id = :account_id"
		params["account_id"] = filter.AccountID
	}
	if filter.Until != nil && !filter.Until.IsZero() {
		query += " AND until = :until"
		params["until"] = filter.Until
	}
	if filter.CreatedAt != nil && !filter.CreatedAt.IsZero() {
		query += " AND created_at = :created_at"
		params["created_at"] = filter.CreatedAt
	}
	if filter.UpdatedAt != nil && !filter.UpdatedAt.IsZero() {
		query += " AND updated_at = :updated_at"
		params["updated_at"] = filter.UpdatedAt
	}
	if filter.DeletedAt != nil && !filter.DeletedAt.IsZero() {
		query += " AND deleted_at = :deleted_at"
		params["deleted_at"] = filter.DeletedAt
	}

	if takeFirst {
		query += " LIMIT 1"
	}

	rows, err := a.db.NamedQueryContext(ctx, query, params)
	if err != nil {
		logrus.Error(err)
		return nil, cerror.ConvertError(err)
	}
	defer rows.Close()

	var keys []*models.Key
	for rows.Next() {
		var key models.Key
		if err := rows.StructScan(&key); err != nil {
			logrus.Error(err)
			return nil, cerror.ConvertError(err)
		}
		keys = append(keys, &key)
	}

	if len(keys) == 0 {
		logrus.Errorf("no keys found with filter %+v", filter)
		return nil, cerror.NewCustomError(cerror.ResourceNotFound, fmt.Sprintf("no keys found with filter %+v", filter))
	}

	return keys[0], nil
}

// filter function that returns accounts based on filter
func (a *AccountRepository) FilterAccounts(ctx context.Context, filter *models.Account, takeFirst bool) ([]*models.Account, *cerror.CustomError) {
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
	if filter.CreatedAt != nil && !filter.CreatedAt.IsZero() {
		query += " AND created_at = :created_at"
		params["created_at"] = filter.CreatedAt
	}
	if filter.UpdatedAt != nil && !filter.UpdatedAt.IsZero() {
		query += " AND updated_at = :updated_at"
		params["updated_at"] = filter.UpdatedAt
	}
	if filter.DeletedAt != nil && !filter.DeletedAt.IsZero() {
		query += " AND deleted_at = :deleted_at"
		params["deleted_at"] = filter.DeletedAt
	}
	if filter.Validation != nil {
		query += " AND validation = :validation"
		params["validation"] = filter.Validation
	}

	if takeFirst {
		query += " LIMIT 1"
	}

	rows, err := a.db.NamedQueryContext(ctx, query, params)
	if err != nil {
		logrus.Error(err)
		return nil, cerror.ConvertError(err)
	}
	defer rows.Close()

	var accounts []*models.Account
	for rows.Next() {
		var account models.Account
		if err := rows.StructScan(&account); err != nil {
			logrus.Error(err)
			return nil, cerror.ConvertError(err)
		}
		accounts = append(accounts, &account)
	}

	if len(accounts) == 0 {
		logrus.Errorf("no accounts found with filter %+v", filter)
		return nil, cerror.NewCustomError(cerror.ResourceNotFound, fmt.Sprintf("no accounts found with filter %+v", filter))
	}

	return accounts, nil
}

func (a *AccountRepository) handleAssociations(ctx context.Context, account *models.Account, associations []string) *cerror.CustomError {
	all := slices.Contains(associations, models.ALL)
	switch {
	case all || slices.Contains(associations, models.SSHKEY):
		sshKeys, err := a.GetSSHKeysByAccountID(ctx, account.ID)
		if err != nil {
			return err
		}
		account.SSHKeys = sshKeys
	}
	return nil
}
