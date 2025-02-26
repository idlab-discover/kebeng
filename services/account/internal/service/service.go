package service

import (
	"context"
    "fmt"

	"github.com/idlab-discover/kebeng/services/account/internal/models"
	"github.com/idlab-discover/kebeng/services/account/internal/repository"
	proto "github.com/idlab-discover/kebeng/services/account/proto"
    "github.com/google/uuid"
)

// TODO: maybe call this different?
// this should contain the business logic of the service
// so doing checks and stuff and calling the database logic

type AccountService struct {
    proto.UnimplementedAccountServiceServer
    repo *repository.AccountRepository
} 

func NewAccountService(repo *repository.AccountRepository) *AccountService {
    return &AccountService{repo: repo}
}


func (a *AccountService) CreateAccount(ctx context.Context, req *proto.CreateAccountRequest) (*proto.Account, error) {
    account := &models.Account{
        DisplayName: req.DisplayName,
        Username: req.Username,
        Email: req.Email,
    }
    
    createdAccount, err := a.repo.CreateAccount(ctx, account)
    if err != nil {
        return nil, err
    }
    
    return a.convertToProtoAccount(createdAccount), nil
}

func (a *AccountService) UpdateAccount(ctx context.Context, req *proto.UpdateAccountRequest) (*proto.Account, error) {
    accountID, err := uuid.Parse(req.Id)
    if err != nil {
        return nil, fmt.Errorf("invalid UUID format: %v", err)
    }
    
    account := &models.Account{
        ID: accountID,
        DisplayName: req.DisplayName,
        Username: req.Username,
        Email: req.Email,
    }
    
    updatedAccount, err := a.repo.UpdateAccount(ctx, account)
    if err != nil {
        return nil, err
    }

    return a.convertToProtoAccount(updatedAccount), nil
}


//TODO: decide what to return
func (a *AccountService) DeleteAccount(ctx context.Context, req *proto.DeleteAccountRequest) (*proto.DeleteAccountResponse, error) {
    accountID, err := uuid.Parse(req.Id)
    if err != nil {
        return nil, fmt.Errorf("invalid UUID format: %v", err)
    }
    
    if err := a.repo.DeleteAccount(ctx, accountID); err != nil {
        return nil, err
    }

    return &proto.DeleteAccountResponse{Success: true}, nil
}

func (a *AccountService) GetAccountByEmail(ctx context.Context, req *proto.GetAccountByEmailRequest) (*proto.Account, error) {
    account, err := a.repo.GetAccountByEmail(ctx, req.Email, true)
    if err != nil {
        return nil, err
    }

    return a.convertToProtoAccount(account), nil
}

func (a *AccountService) GetAccountByID(ctx context.Context, req *proto.GetAccountByIDRequest) (*proto.Account, error) {
    accountID, err := uuid.Parse(req.Id)
    if err != nil {
        return nil, fmt.Errorf("invalid UUID format: %v", err)
    }
    
    account, err := a.repo.GetAccountByID(ctx, accountID, true)
    if err != nil {
        return nil, err
    }

    return a.convertToProtoAccount(account), nil
}

func (a *AccountService) GetAccountByUsername(ctx context.Context, req *proto.GetAccountByUsernameRequest) (*proto.Account, error) {
    account, err := a.repo.GetAccountByUsername(ctx, req.Username, true)
    if err != nil {
        return nil, err
    }

    return a.convertToProtoAccount(account), nil
}

func (a *AccountService) convertToProtoAccount(account *models.Account) *proto.Account {
    return &proto.Account{
        Id: account.ID.String(),
        DisplayName: account.DisplayName,
        Username: account.Username,
        Email: account.Email,
    }
}
