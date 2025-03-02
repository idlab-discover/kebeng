package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/idlab-discover/kebeng/services/account/internal/models"
	"gorm.io/gorm"
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

  AddSnapEntryToAccount(ctx context.Context, accountID uuid.UUID, snapEntryID uuid.UUID) error
  RemoveSnapEntryFromAccount(ctx context.Context, accountID uuid.UUID, snapEntryID uuid.UUID) error
  GetSnapEntryIDsByAccountID(ctx context.Context, accountID uuid.UUID) ([]uuid.UUID, error)
}

type AccountRepository struct {
	db *gorm.DB
}

func NewAccountRepository(db *gorm.DB) *AccountRepository {
	return &AccountRepository{db: db}
}

func (a *AccountRepository) CreateAccount(ctx context.Context, account *models.Account) (*models.Account, error) {
	if err := a.db.WithContext(ctx).Create(account).Error; err != nil {
		return nil, err
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

/* TODO: not used since SnapEntryID is not stored in this database so remove but comments for now
func (a *AccountRepository) AddSnapEntryToAccount(ctx context.Context, accountID uuid.UUID, snapEntryID uuid.UUID) error {
	account, err := a.GetAccountByID(ctx, accountID, true)
	if err != nil || account == nil {
		return fmt.Errorf("could not find account with id %s: err: %v", accountID, err)
	}

	account.SnapEntryIDs = append(account.SnapEntryIDs, snapEntryID)
	return a.db.WithContext(ctx).Save(account).Error
}

func (a *AccountRepository) RemoveSnapEntryFromAccount(ctx context.Context, accountID uuid.UUID, snapEntryID uuid.UUID) error {
	account, err := a.GetAccountByID(ctx, accountID, true)
	if err != nil || account == nil {
		return fmt.Errorf("could not find account with id %s: err: %v", accountID, err)
	}

	var updatedSnapEntryIDs []uuid.UUID
	for _, id := range account.SnapEntryIDs {
		if id != snapEntryID {
			updatedSnapEntryIDs = append(updatedSnapEntryIDs, id)
		}
	}

	account.SnapEntryIDs = updatedSnapEntryIDs
	return a.db.WithContext(ctx).Save(account).Error
}

func (a *AccountRepository) GetSnapEntryIDsByAccountID(ctx context.Context, accountID uuid.UUID) ([]uuid.UUID, error) {
	account, err := a.GetAccountByID(ctx, accountID, false)
	if err != nil || account == nil {
		return nil, fmt.Errorf("could not find account with id %s: err: %v", accountID, err)
	}
	return account.SnapEntryIDs, nil
}
*/

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
