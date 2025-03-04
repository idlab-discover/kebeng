package logic

import (
	"bytes"
	"context"
	"io"
	"os"
	"path"

	"github.com/google/uuid"
	"github.com/idlab-discover/kebeng/services/store/internal/errors"
	"github.com/idlab-discover/kebeng/services/store/internal/objectstore"
	"github.com/idlab-discover/kebeng/services/store/internal/repositories"
	proto "github.com/idlab-discover/kebeng/services/store/proto"
	"github.com/sirupsen/logrus"
)

type StoreLogic struct {
	proto.UnimplementedStoreServiceServer
	repo *repositories.SnapsRepository
}

func NewStoreLogic(repo *repositories.SnapsRepository) *StoreLogic {
	return &StoreLogic{repo: repo}
}

func (s *StoreLogic) UploadSnap(ctx context.Context, req *proto.UploadSnapRequest) (*proto.UploadSnapResponse, error) {
	// TODO: check which fields are required and which are optional in req
	errList := make([]*proto.Error, 0)
	snapFileName, id, err := saveFileToTemp(bytes.NewReader(req.File))
	if err != nil {
		logrus.Errorf("Failed to save file to temp storage: %v", err)
		errList = append(errList, &proto.Error{Code: errors.InternalServerError, Message: "Failed to save file to temp storage"})
		return &proto.UploadSnapResponse{Errors: errList}, err
	}

	objectstore := objectstore.NewObjectStore()
	tmpPath := path.Join(os.TempDir(), snapFileName)

	size, err := objectstore.SaveFileToBucket("unscanned", tmpPath)
	if err != nil {
		logrus.Errorf("Failed to save file to object store: %v", err)
		errList = append(errList, &proto.Error{Code: errors.InternalServerError, Message: "Failed to save file to object store"})
		return &proto.UploadSnapResponse{Errors: errList}, err
	}

	// addSnap() adds snap to snap_entries table
	_, err = s.repo.AddSnap(snapFileName, size, uuid.New()) // uuid.New() is a placeholder for account id that is going to be added later throught the context
	if err != nil {
		logrus.Error(err)
		errList = append(errList, &proto.Error{Code: errors.InternalServerError, Message: "Failed to add snap to database"})
		return &proto.UploadSnapResponse{Errors: errList}, err
	}

	return &proto.UploadSnapResponse{Id: id, DisplayName: snapFileName}, nil
}

func saveFileToTemp(snapFile io.Reader) (string, string, error) {
	// Generate random file name for the new uploaded file so it doesn't override the old file with same name
	snapFileId := uuid.New().String()
	newFileName := snapFileId + ".snap"

	out, err := os.Create(path.Join("/tmp", newFileName))
	if err != nil {
		return "", "", err
	}
	defer out.Close()

	_, err = io.Copy(out, snapFile)
	if err != nil {
		return "", "", err
	}

	return newFileName, snapFileId, nil
}
