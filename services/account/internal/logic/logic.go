package logic

import (
	"context"

	"github.com/idlab-discover/kebeng/services/account/internal/models"
	"github.com/idlab-discover/kebeng/services/account/internal/repository"
	proto "github.com/idlab-discover/kebeng/services/account/proto"
    "github.com/google/uuid"
    "github.com/idlab-discover/kebeng/services/account/internal/config"
    "github.com/idlab-discover/kebeng/services/account/internal/errors"
)
    
// NOTE: in all the endpoints the errors of the actual logic are included in the response and NOT returned as an error
// SO NEVER RETURN AN ERROR IN THIS LOGIC but insert them in the response

// TODO: maybe call this different?
// this should contain the business logic of the service
// so doing checks and stuff and calling the database logic

type AccountService struct {
    config *config.Config
    proto.UnimplementedAccountServiceServer
    repo *repository.AccountRepository

} 

func NewAccountService(repo *repository.AccountRepository, config *config.Config) *AccountService {
    return &AccountService{repo: repo, config: config}
}


func (a *AccountService) CreateAccount(ctx context.Context, req *proto.CreateAccountRequest) (*proto.AccountResponse, error) {
    el := make([]*proto.Error, 0)
    account := &models.Account{
        DisplayName: req.DisplayName,
        Username: req.Username,
        Email: req.Email,
    }
    
    createdAccount, err := a.repo.CreateAccount(ctx, account)
    if err != nil {
        el = append(el, &proto.Error{
            Code: errors.InternalServerError,
            Message: err.Error(),
        })
        return &proto.AccountResponse{Errors: el}, nil
    }
    
    return a.convertToProtoAccount(createdAccount,el), nil
}

func (a *AccountService) UpdateAccount(ctx context.Context, req *proto.UpdateAccountRequest) (*proto.AccountResponse, error) {
    el := make([]*proto.Error, 0)
    accountID, err := uuid.Parse(req.Id)
    if err != nil {
        el = append(el, &proto.Error{
            Code: errors.BadRequest,
            Message: "invalid UUID format",
        })
        return &proto.AccountResponse{Errors: el}, nil
    }
    
    account := &models.Account{
        ID: accountID,
        DisplayName: req.DisplayName,
        Username: req.Username,
        Email: req.Email,
    }
    
    updatedAccount, err := a.repo.UpdateAccount(ctx, account)
    if err != nil {
        el = append(el, &proto.Error{
            Code: errors.InternalServerError,
            Message: err.Error(),
        })
        return &proto.AccountResponse{Errors: el}, nil
    }

    return a.convertToProtoAccount(updatedAccount,el), nil
}


func (a *AccountService) DeleteAccount(ctx context.Context, req *proto.DeleteAccountRequest) (*proto.DeleteAccountResponse, error) {
    el := make([]*proto.Error, 0)
    accountID, err := uuid.Parse(req.Id)
    if err != nil {
        el = append(el, &proto.Error{
            Code: errors.BadRequest,
            Message: "invalid UUID format",
        })
        return &proto.DeleteAccountResponse{Success: false, Errors: el}, nil
    }
    
    if err := a.repo.DeleteAccount(ctx, accountID); err != nil {
        el = append(el, &proto.Error{
            Code: errors.InternalServerError,
            Message: err.Error(),
        })
        return &proto.DeleteAccountResponse{Success: false, Errors: el}, nil
    }

    return &proto.DeleteAccountResponse{Success: true}, nil
}

func (a *AccountService) GetAccountByEmail(ctx context.Context, req *proto.GetAccountByEmailRequest) (*proto.AccountResponse, error) {
    el := make([]*proto.Error, 0)
    account, err := a.repo.GetAccountByEmail(ctx, req.Email, true)
    if err != nil {
        el = append(el, &proto.Error{
            Code: errors.InternalServerError,
            Message: err.Error(),
        })
        return &proto.AccountResponse{Errors: el}, nil
    }

    return a.convertToProtoAccount(account,el), nil
}

func (a *AccountService) GetAccountByID(ctx context.Context, req *proto.GetAccountByIDRequest) (*proto.AccountResponse, error) {
    el := make([]*proto.Error, 0)
    accountID, err := uuid.Parse(req.Id)
    if err != nil {
        el = append(el, &proto.Error{
            Code: errors.BadRequest,
            Message: "invalid UUID format",
        })
        return &proto.AccountResponse{Errors: el}, nil
    }
    
    account, err := a.repo.GetAccountByID(ctx, accountID, true)
    if err != nil {
        el = append(el, &proto.Error{
            Code: errors.InternalServerError,
            Message: err.Error(),
        })
        return &proto.AccountResponse{Errors: el}, nil
    }

    return a.convertToProtoAccount(account,el), nil
}

func (a *AccountService) GetAccountByUsername(ctx context.Context, req *proto.GetAccountByUsernameRequest) (*proto.AccountResponse, error) {
    el := make([]*proto.Error, 0)
    account, err := a.repo.GetAccountByUsername(ctx, req.Username, true)
    if err != nil {
        el = append(el, &proto.Error{
            Code: errors.InternalServerError,
            Message: err.Error(),
        })
        return &proto.AccountResponse{Errors: el}, nil
    }

    return a.convertToProtoAccount(account,el), nil
}

func (a *AccountService) AddKey(ctx context.Context, req *proto.AddKeyRequest) (*proto.KeyResponse, error) {
    el := make([]*proto.Error, 0)
    key, err := a.repo.AddKey(ctx, req.KeyName, req.Sha3384, req.EncodedPublicKey, req.AccountEmail)
    if err != nil {
        el = append(el, &proto.Error{
            Code: errors.InternalServerError,
            Message: err.Error(),
        })
        return &proto.KeyResponse{Errors: el}, nil
    }
    return &proto.KeyResponse{
        Name: key.Name,
        Sha3384: key.SHA3384,
        EncodedPublicKey: key.EncodedPublicKey,
        Errors: el,
    }, nil
}

func (a *AccountService) GetKey(ctx context.Context, req *proto.GetKeyBySHA3384Request) (*proto.KeyResponse, error) {
    el := make([]*proto.Error, 0)
    key, err := a.repo.GetKeyBySHA3384(ctx, req.Sha3384)
    if err != nil {
        el = append(el, &proto.Error{
            Code: errors.InternalServerError,
            Message: err.Error(),
        })
        return &proto.KeyResponse{Errors: el}, nil
    }
    return &proto.KeyResponse{
        Name: key.Name,
        Sha3384: key.SHA3384,
        EncodedPublicKey: key.EncodedPublicKey,
        Errors: el,
    }, nil
}

func (a *AccountService) convertToProtoAccount(account *models.Account, el []*proto.Error) *proto.AccountResponse {
    return &proto.AccountResponse{
        Id: account.ID.String(),
        DisplayName: account.DisplayName,
        Username: account.Username,
        Email: account.Email,
        Errors: el,
    }
}


