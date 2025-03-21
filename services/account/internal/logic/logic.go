package logic

import (
	"context"
	"time"

	"github.com/google/uuid"
	cerror "github.com/idlab-discover/kebeng/common/cerror"
	"github.com/idlab-discover/kebeng/services/account/internal/config"
	"github.com/idlab-discover/kebeng/services/account/internal/models"
	"github.com/idlab-discover/kebeng/services/account/internal/repository"
	proto "github.com/idlab-discover/kebeng/services/account/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// NOTE: in all the endpoints the cerror of the actual logic are included in the response and NOT returned as an error
// SO NEVER RETURN AN ERROR IN THIS LOGIC but insert them in the response

// TODO: maybe call this different?
// this should contain the business logic of the service
// so doing checks and stuff and calling the database logic

type AccountService struct {
	config *config.Config
	proto.UnimplementedAccountServiceServer
	repo repository.IAccountRepository
}

func NewAccountService(repo repository.IAccountRepository, config *config.Config) *AccountService {
	return &AccountService{repo: repo, config: config}
}

func ptrTimeToPtrTimestamp(t *time.Time) *timestamppb.Timestamp {
	if t == nil {
		return nil
	}
	return timestamppb.New(*t)
}

func (a *AccountService) CreateAccount(ctx context.Context, req *proto.CreateAccountRequest) (*proto.AccountResponse, error) {
	el := make([]*proto.Error, 0)
	account := &models.Account{
		DisplayName: req.DisplayName,
		Username:    req.Username,
		Email:       req.Email,
	}

	createdAccount, err := a.repo.CreateAccount(ctx, account)

	if err != nil {
		el = append(el, &proto.Error{
			Code:    err.GetCode(),
			Message: err.GetMessage(),
		})
		return &proto.AccountResponse{Errors: el}, nil
	}

	return a.convertToProtoAccount(createdAccount, el), nil
}

func (a *AccountService) UpdateAccount(ctx context.Context, req *proto.UpdateAccountRequest) (*proto.AccountResponse, error) {
	el := make([]*proto.Error, 0)
	accountID, err := uuid.Parse(req.Id)
	if err != nil {
		el = append(el, &proto.Error{
			Code:    cerror.BadRequest,
			Message: "invalid UUID format",
		})
		return &proto.AccountResponse{Errors: el}, nil
	}

	account := &models.Account{
		ID:          accountID,
		DisplayName: req.DisplayName,
		Username:    req.Username,
		Email:       req.Email,
	}

	updatedAccount, cerr := a.repo.UpdateAccount(ctx, account)
	if cerr != nil {
		el = append(el, &proto.Error{
			Code:    cerr.GetCode(),
			Message: cerr.GetMessage(),
		})
		return &proto.AccountResponse{Errors: el}, nil
	}

	return a.convertToProtoAccount(updatedAccount, el), nil
}

func (a *AccountService) DeleteAccount(ctx context.Context, req *proto.DeleteAccountRequest) (*proto.DeleteAccountResponse, error) {
	el := make([]*proto.Error, 0)
	accountID, err := uuid.Parse(req.Id)
	if err != nil {
		el = append(el, &proto.Error{
			Code:    cerror.BadRequest,
			Message: "invalid UUID format",
		})
		return &proto.DeleteAccountResponse{Success: false, Errors: el}, nil
	}

	if err := a.repo.DeleteAccount(ctx, accountID); err != nil {
		el = append(el, &proto.Error{
			Code:    err.GetCode(),
			Message: err.GetMessage(),
		})
		return &proto.DeleteAccountResponse{Success: false, Errors: el}, nil
	}

	return &proto.DeleteAccountResponse{Success: true}, nil
}

func (a *AccountService) GetAccountByEmail(ctx context.Context, req *proto.GetAccountByEmailRequest) (*proto.AccountResponse, error) {
	el := make([]*proto.Error, 0)
	account, err := a.repo.GetAccountByEmail(ctx, req.Email, []string{models.ALL})
	if err != nil {
		el = append(el, &proto.Error{
			Code:    err.GetCode(),
			Message: err.GetMessage(),
		})
		return &proto.AccountResponse{Errors: el}, nil
	}

	return a.convertToProtoAccount(account, el), nil
}

func (a *AccountService) GetAccountByID(ctx context.Context, req *proto.GetAccountByIDRequest) (*proto.AccountResponse, error) {
	el := make([]*proto.Error, 0)
	accountID, err := uuid.Parse(req.Id)
	if err != nil {
		el = append(el, &proto.Error{
			Code:    cerror.BadRequest,
			Message: "invalid UUID format",
		})
		return &proto.AccountResponse{Errors: el}, nil
	}

	account, cerr := a.repo.GetAccountByID(ctx, accountID, []string{models.ALL})
	if cerr != nil {
		el = append(el, &proto.Error{
			Code:    cerr.GetCode(),
			Message: cerr.GetMessage(),
		})
		return &proto.AccountResponse{Errors: el}, nil
	}

	return a.convertToProtoAccount(account, el), nil
}

func (a *AccountService) GetAccountsByIds(ctx context.Context, req *proto.GetAccountsByIdsRequest) (*proto.GetAccountsByIdsResponse, error) {
	el := make([]*proto.Error, 0)
	response := &proto.GetAccountsByIdsResponse{}
	accountIDs := make([]uuid.UUID, 0, len(req.Ids))
	for _, id := range req.Ids {
		accountID, err := uuid.Parse(id)
		if err != nil {
			el = append(el, &proto.Error{
				Code:    cerror.BadRequest,
				Message: "invalid UUID format",
			})
		}
		accountIDs = append(accountIDs, accountID)
	}
	// first parse everything than only return if there are cerror so we check all ids
	if len(el) > 0 {
		return &proto.GetAccountsByIdsResponse{Errors: el}, nil
	}

	for _, id := range accountIDs {
		account, err := a.repo.GetAccountByID(ctx, id, []string{models.ALL})
		if err != nil {
			el = append(el, &proto.Error{
				Code:    err.GetCode(),
				Message: err.GetMessage(),
			})
			continue
		}
		// convert to proto
		a.convertToProtoAccount(account, el)
		response.Accounts = append(response.Accounts, a.convertToProtoAccount(account, el))
	}

	if len(el) > 0 {
		response.Errors = el
	}
	return response, nil
}

func (a *AccountService) GetAccountByUsername(ctx context.Context, req *proto.GetAccountByUsernameRequest) (*proto.AccountResponse, error) {
	el := make([]*proto.Error, 0)
	account, err := a.repo.GetAccountByUsername(ctx, req.Username, []string{models.ALL})
	if err != nil {
		el = append(el, &proto.Error{
			Code:    err.GetCode(),
			Message: err.GetMessage(),
		})
		return &proto.AccountResponse{Errors: el}, nil
	}

	return a.convertToProtoAccount(account, el), nil
}

func (a *AccountService) AddKey(ctx context.Context, req *proto.AddKeyRequest) (*proto.KeyResponse, error) {
	el := make([]*proto.Error, 0)
	key, err := a.repo.AddKeyToAccountByEmail(ctx, req.KeyName, req.Sha3384, req.EncodedPublicKey, req.AccountEmail)
	if err != nil {
		el = append(el, &proto.Error{
			Code:    err.GetCode(),
			Message: err.GetMessage(),
		})
		return &proto.KeyResponse{Errors: el}, nil
	}
	return &proto.KeyResponse{
		Name:             key.Name,
		Sha3384:          key.SHA3384,
		EncodedPublicKey: key.EncodedPublicKey,
		Errors:           el,
	}, nil
}

func (a *AccountService) GetKey(ctx context.Context, req *proto.GetKeyBySHA3384Request) (*proto.KeyResponse, error) {
	el := make([]*proto.Error, 0)
	key, err := a.repo.GetKeyBySHA3384(ctx, req.Sha3384)
	if err != nil {
		el = append(el, &proto.Error{
			Code:    err.GetCode(),
			Message: err.GetMessage(),
		})
		return &proto.KeyResponse{Errors: el}, nil
	}
	return &proto.KeyResponse{
		Name:             key.Name,
		Sha3384:          key.SHA3384,
		EncodedPublicKey: key.EncodedPublicKey,
		Errors:           el,
	}, nil
}

func (a *AccountService) GetKeysByAccountId(ctx context.Context, req *proto.GetKeysByAccountIdRequest) (*proto.KeysResponse, error) {
	el := make([]*proto.Error, 0)
	// parse to uuid
	accountID, err := uuid.Parse(req.AccountId)
	if err != nil {
		el = append(el, &proto.Error{
			Code:    cerror.BadRequest,
			Message: "invalid UUID format",
		})
		return &proto.KeysResponse{Errors: el}, nil
	}

	// get keys
	keys, cerr := a.repo.GetKeysByAccountID(ctx, accountID)
	if cerr != nil {
		el = append(el, &proto.Error{
			Code:    cerr.GetCode(),
			Message: cerr.GetMessage(),
		})
		return &proto.KeysResponse{Errors: el}, nil
	}

	// convert to proto
	keyResponses := make([]*proto.KeyResponse, 0)
	for _, key := range keys {
		keyResponses = append(keyResponses, &proto.KeyResponse{
			Name:             key.Name,
			Sha3384:          key.SHA3384,
			EncodedPublicKey: key.EncodedPublicKey,
			Since:            ptrTimeToPtrTimestamp(key.CreatedAt),
			Until:            ptrTimeToPtrTimestamp(key.Until),
		})
	}

	return &proto.KeysResponse{
		Keys:   keyResponses,
		Errors: el,
	}, nil
}

func (a *AccountService) convertToProtoAccount(account *models.Account, el []*proto.Error) *proto.AccountResponse {
	return &proto.AccountResponse{
		Id:          account.ID.String(),
		DisplayName: account.DisplayName,
		Username:    account.Username,
		Email:       account.Email,
		Validation:  account.Validation,
		Errors:      el,
	}
}
