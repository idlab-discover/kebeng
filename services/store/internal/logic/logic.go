package logic

import (
	"context"
	"io"
	"os"
	"path"

	"github.com/google/uuid"
	"github.com/idlab-discover/kebeng/services/store/internal/models"
	"github.com/idlab-discover/kebeng/services/store/internal/repositories"
	proto "github.com/idlab-discover/kebeng/services/store/proto"
	"github.com/sirupsen/logrus"
)

// TODO: maybe call this different?
// this should contain the business logic of the service
// so doing checks and stuff and calling the database logic

type StoreService struct {
	proto.UnimplementedStoreServiceServer
	repo *repositories.SnapsRepository
}

func NewStoreService(repo *repositories.SnapsRepository) *StoreService {
	return &StoreService{repo: repo}
}

func (s *StoreService) UploadSnap(ctx context.Context, req *proto.UploadSnapRequest) (*proto.UploadSnapResponse, error) {
	snapFileName, id, err := saveFileToTemp(snapFile)
	if err != nil {
		logrus.Errorf("Failed to save file to temp storage: %v", err)
		return "", err
	}
	
	snap := &models.SnapEntry{
		Name: req.DisplayName,
	}

	createdSnap, err := s.repo.AddSnap(snap.Name, snap.)
	if err != nil {
		return nil, err
	}

	return &proto.UploadSnapResponse{
		Id:          createdSnap.ID.String(),
		DisplayName: createdSnap.Name,
	}, nil
}

func (h *Handler) UnscannedUpload(snapFile io.Reader) (string, error) {
	snapFileName, id, err := saveFileToTemp(snapFile)
	if err != nil {
		logrus.Errorf("Failed to save file to temp storage: %v", err)
		return "", err
	}

	// CHECK: can upload handle reusing the same connection or should there be a new connection for each upload?
	//objStore := objectstore.NewObjectStore()
	tmpPath := path.Join(os.TempDir(), snapFileName)

	// err = objStore.SaveFileToBucket("unscanned", tmpPath)
	size, err := h.obs.SaveFileToBucket("unscanned", tmpPath)
	if err != nil {
		logrus.Errorf("Failed to save file to object store: %v", err)
		return "", err
	}

	// addSnap() adds snap to snap_entries table
	_, err = h.snaps.AddSnap(snapFileName, size, uuid.New()) // uuid.New() is a placeholder for account id that is going to be added later throught the context
	if err != nil {
		logrus.Error(err)
	}

	return id, nil
}
