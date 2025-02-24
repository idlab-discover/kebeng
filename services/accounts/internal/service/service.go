package service

import (
	"context"

	"github.com/idlab-discover/kebeng/services/accounts/internal/models"
	"github.com/idlab-discover/kebeng/services/accounts/internal/repository"
	proto "github.com/idlab-discover/kebeng/services/accounts/proto"
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


func (a *AccountService) CreateAccount(ctx context.Context, req *proto.CreateAccountRequest) (*proto.CreateAccountResponse, error) {
    account := &models.Account{
        DisplayName: req.DisplayName,
        Username: req.Username,
        Email: req.Email,
    }
    
    createdAccount, err := a.repo.CreateAccount(ctx, account)
    if err != nil {
        return nil, err
    }

    return &proto.CreateAccountResponse{
        Id: createdAccount.ID.String(),
        DisplayName: createdAccount.DisplayName,
        Username: createdAccount.Username,
        Email: createdAccount.Email,
    }, nil
}

