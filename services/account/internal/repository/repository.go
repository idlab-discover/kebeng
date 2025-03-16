package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/idlab-discover/kebeng/services/account/internal/models"
	"github.com/jmoiron/sqlx"
	"gorm.io/gorm/clause"
)

// IAccountRepository defines the interface for account-related database operations
type IAccountRepository interface {
	CreateAccount(ctx context.Context, account *models.Account) (*models.Account, error)
	UpdateAccount(ctx context.Context, account *models.Account) (*models.Account, error)
	DeleteAccount(ctx context.Context, accountID uuid.UUID) error
	GetAccountByEmail(ctx context.Context, email string, preload bool) (*models.Account, error)
	GetAccountByID(ctx context.Context, accountID uuid.UUID, preload bool) (*models.Account, error)
	GetAccountByUsername(ctx context.Context, username string, preload bool) (*models.Account, error)

	AddKey(ctx context.Context, name, sha3384, encodedPublicKey, accountEmail string) (*models.Key, error)
	GetKeyBySHA3384(ctx context.Context, sha3384 string) (*models.Key, error)
	GetKeysByAccountID(ctx context.Context, accountID uuid.UUID) ([]*models.Key, error)
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
	if err := a.db.WithContext(ctx).Save(account).Error; err != nil {
		return nil, err
	}
	return account, nil
}

func (a *AccountRepository) DeleteAccount(ctx context.Context, accountID uuid.UUID) error {
	return a.db.WithContext(ctx).Where(&models.Account{ID: accountID}).Delete(&models.Account{}).Error
}

func (a *AccountRepository) GetAccountByEmail(ctx context.Context, email string, preload bool) (*models.Account, error) {
	return a.getAccountByWhereModel(ctx, &models.Account{Email: email}, preload)
}

func (a *AccountRepository) GetAccountByID(ctx context.Context, accountID uuid.UUID, preload bool) (*models.Account, error) {
	return a.getAccountByWhereModel(ctx, &models.Account{ID: accountID}, preload)
}

func (a *AccountRepository) GetAccountByUsername(ctx context.Context, username string, preload bool) (*models.Account, error) {
	return a.getAccountByWhereModel(ctx, &models.Account{Username: username}, preload)
}

// AddKey associates a key with an account
func (a *AccountRepository) AddKey(ctx context.Context, name, sha3384, encodedPublicKey, email string) (*models.Key, error) {
	acct, err := a.GetAccountByEmail(ctx, email, false)
	if err != nil || acct == nil {
		return nil, fmt.Errorf("could not find account with email %s: err: %v", email, err)
	}

	accountKey := &models.Key{
		Name:             name,
		SHA3384:          sha3384,
		AccountID:        acct.ID,
		EncodedPublicKey: encodedPublicKey,
	}

	if err := a.db.WithContext(ctx).Create(accountKey).Error; err != nil {
		return nil, err
	}

	return accountKey, nil
}

func (a *AccountRepository) GetKeyBySHA3384(ctx context.Context, sha3384 string) (*models.Key, error) {
	return a.getKeyByWhereModel(ctx, &models.Key{SHA3384: sha3384})
}

func (a *AccountRepository) GetKeysByAccountID(ctx context.Context, accountID uuid.UUID) ([]*models.Key, error) {
	var keys []*models.Key
	if err := a.db.WithContext(ctx).Where(&models.Key{AccountID: accountID}).Find(&keys).Error; err != nil {
		return nil, err
	}
	return keys, nil
}

// getKeyByWhereModel retrieves a key based on a filter
func (a *AccountRepository) getKeyByWhereModel(ctx context.Context, whereModel *models.Key) (*models.Key, error) {
	var key models.Key
	if err := a.db.WithContext(ctx).Where(whereModel).Preload(clause.Associations).First(&key).Error; err != nil {
		return nil, fmt.Errorf("could not find key %+v: %w", whereModel, err)
	}
	return &key, nil
}

// getAccountByWhereModel retrieves an account based on a filter
func (a *AccountRepository) getAccountByWhereModel(ctx context.Context, whereModel *models.Account, preload bool) (*models.Account, error) {
	var account models.Account
	query := a.db.WithContext(ctx).Where(whereModel)
	if preload {
		query = query.Preload(clause.Associations)
	}

	if err := query.First(&account).Error; err != nil {
		return nil, fmt.Errorf("could not find account %+v: %w", whereModel, err)
	}
	return &account, nil
}
