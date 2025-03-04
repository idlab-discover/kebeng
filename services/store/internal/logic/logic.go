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

func (s *StoreLogic) RegisterSnapName(ctx context.Context, req *proto.RegisterSnapNameRequest) (*proto.RegisterSnapNameResponse, error) {
	errList := make([]*proto.Error, 0)

	if req.SnapName == "" {
		errList = append(errList, &proto.Error{Code: errors.MissingField, Message: "Snap name is required"})
		return &proto.RegisterSnapNameResponse{Errors: errList}, nil
	}

	// First check if the snap name is already registered
	snapEntry, err := s.repo.GetSnapByName(req.SnapName, false)
	if err != nil {
		logrus.Error(err)
		errList = append(errList, &proto.Error{Code: errors.InternalServerError, Message: "Failed to get snap from database"})
		return &proto.RegisterSnapNameResponse{Errors: errList}, err
	}
	if snapEntry != nil { // if dryRun is true, we only check if the snap name is already registered -> snapEntry != nil
		errList = append(errList, &proto.Error{Code: errors.AlreadyRegistered, Message: "The snap name '" + req.SnapName + "' is already registered."})
		return &proto.RegisterSnapNameResponse{Errors: errList}, err
	}

	if req.DryRun {
		return &proto.RegisterSnapNameResponse{SnapName: req.SnapName}, nil
	}

	// If there is no snap with the same name and dry_run == false, register the snap name
	snapEntry, err = s.repo.RegisterSnap(req.SnapName, req.IsPrivate)
	if err != nil {
		logrus.Error(err)
		errList = append(errList, &proto.Error{Code: errors.InternalServerError, Message: "Failed to register snap name"})
		return &proto.RegisterSnapNameResponse{Errors: errList}, err
	}

	return &proto.RegisterSnapNameResponse{Id: snapEntry.ID.String(), SnapName: snapEntry.Name}, nil
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
