package logic

import (
	"github.com/idlab-discover/kebeng/services/assertion/internal/config"
	"github.com/idlab-discover/kebeng/services/assertion/internal/repository"
	proto "github.com/idlab-discover/kebeng/services/assertion/proto"
)

// TODO: maybe call this different?
// this should contain the business logic of the service
// so doing checks and stuff and calling the database logic

type AssertionService struct {
	config *config.Config
	repo   *repository.AssertionRepository
	proto.UnimplementedAssertionServiceServer
}

func NewAssertionService(repo *repository.AssertionRepository, config *config.Config) *AssertionService {
	return &AssertionService{repo: repo, config: config}
}
