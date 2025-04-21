package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"net"
	"time"

	"github.com/idlab-discover/kebeng/services/assertion/internal/config"
	"github.com/idlab-discover/kebeng/services/assertion/internal/database"
	"github.com/idlab-discover/kebeng/services/assertion/internal/logic"
	"github.com/idlab-discover/kebeng/services/assertion/internal/repository"
	proto "github.com/idlab-discover/kebeng/services/assertion/proto"
	"github.com/sirupsen/logrus"
	"github.com/snapcore/snapd/asserts"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func main() {
	ctx := context.Background()
	cfg, err := config.LoadConfig()
	if err != nil {
		logrus.Fatalf("Failed to load configuration: %v", err)
	}
	logrus.Infof("Loaded configuration: %+v", cfg)

	db, err := database.NewDatabase(cfg)
	if err != nil {
		logrus.Fatalf("Failed to connect to database: %v", err)
	}

	logrus.Infof("Connected to database: %v", db)

	assertionDB, err := asserts.OpenDatabase(&asserts.DatabaseConfig{
		KeypairManager: asserts.NewMemoryKeypairManager(),
	})

	if err != nil {
		logrus.Fatalf("Failed to open assertion database: %s", err)
	}

	if err := assertionDB.ImportKey(cfg.RootKey); err != nil {
		logrus.Fatalf("Failed to import root key: %s in assertions db", err)
	}

	repo := repository.NewAssertionRepository(db)
	assertionLogic := logic.NewAssertionLogic(cfg, repo, assertionDB)

	if cfg.TestMode {
		logrus.Infof("Running in test mode")
	}

	// after creating the rootkey, a account key assertion has to be made this will be used by snapd to verify by who the other assertions were signed
	// TODO: somehow fix that root account is created and matches the accountId here
	// for now just random uuid that doesn't match real account
	now := time.Now()
	serializedPub, err := asserts.EncodePublicKey(cfg.RootKey.PublicKey())
	if err != nil {
		logrus.Fatalf("Failed to serialize public key: %v", err)
	}

	b64pub := base64.StdEncoding.EncodeToString(serializedPub)

	req := &proto.AddAccountKeyAssertionRequest{
		EncodedPublicKey:  b64pub,
		PublicKeySha3_384: cfg.RootKey.PublicKey().ID(),
		AccountId:         cfg.RootAccountID.String(),
		Name:              "root",
		Since:             &timestamppb.Timestamp{Seconds: now.Unix()},
		Until:             &timestamppb.Timestamp{Seconds: now.Add(24 * time.Hour).Unix()},
	}
	accountKeyAssertion, err := assertionLogic.AddAccountKeyAssertion(ctx, req)
	if len(accountKeyAssertion.Errors) != 0 {
		logrus.Fatalf("Failed to create account key assertion: %v", accountKeyAssertion.Errors)
	}

	// also create a root account assertion
	req2 := &proto.AddAccountAssertionRequest{
		AccountId:   cfg.RootAccountID.String(),
		DisplayName: "kebeng",
		Username:    "kebeng",
		Validation:  "certified",
		Timestamp:   &timestamppb.Timestamp{Seconds: now.Unix()},
	}
	accountAssertion, err := assertionLogic.AddAccountAssertion(ctx, req2)
	if len(accountAssertion.Errors) != 0 {
		logrus.Fatalf("Failed to create account assertion: %v", accountAssertion.Errors)
	}

	// start grpc server
	lis, err := net.Listen("tcp", fmt.Sprintf("%s:%d", cfg.GRPCHost, cfg.GRPCPort))
	if err != nil {
		logrus.Fatalf("Failed to listen: %v", err)
	}
	grpcServer := grpc.NewServer()
	// this line will match the logic functionality to the proto interface
	proto.RegisterAssertionServiceServer(grpcServer, assertionLogic)

	logrus.Infof("Starting gRPC server on %s:%d", cfg.GRPCHost, cfg.GRPCPort)
	if err := grpcServer.Serve(lis); err != nil {
		logrus.Fatalf("Failed to serve: %v", err)
	}
}
