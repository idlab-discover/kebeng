package server

import (
	"context"
	"io"
	"strings"
	"sync"

	"github.com/idlab-discover/kebeng/pkg/crypto"
	"github.com/snapcore/snapd/asserts"
	"github.com/snapcore/snapd/asserts/assertstest"

	"github.com/gin-gonic/gin"
	"github.com/idlab-discover/kebeng/config"
	"github.com/idlab-discover/kebeng/config/configkey"
	"github.com/idlab-discover/kebeng/pkg/database"
	"github.com/idlab-discover/kebeng/pkg/middleware"
	"github.com/idlab-discover/kebeng/pkg/objectstore"
	"github.com/idlab-discover/kebeng/pkg/repositories"
	"github.com/idlab-discover/kebeng/pkg/store"
	"github.com/minio/minio-go/v7"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
    
    client "github.com/idlab-discover/kebeng/services/account/client"
)

type Server struct {
}

func (s *Server) Run() {
	logrus.SetLevel(logrus.TraceLevel)
    config.LoadConfig()

	logLevelConfig := viper.GetString(configkey.LogLevel)
	l, errLevel := logrus.ParseLevel(logLevelConfig)
	if errLevel != nil {
		logrus.Error(errLevel)
	} else {
		logrus.SetLevel(l)
	}

	// Setup gin and routes
	r := gin.Default()
	if viper.GetBool(configkey.DebugMode) {
		logrus.Info("Debug mode enabled")
		r.Use(middleware.RequestLoggerMiddleware())
	} else {
		logrus.Info("Debug mode disabled")
	}

	r.NoRoute(func(c *gin.Context) {
		c.JSON(404, gin.H{"code": "KEBE STORE: PAGE_NOT_FOUND", "message": "Page not found"})
	})

	db, _ := database.CreateDatabase()
	assertsDatabase := GetDatabaseWithRootKey()

    //TODO: TEST create root account
    accountClient, err := client.NewAccountClient(configkey.AccountServiceHost, viper.GetInt(configkey.AccountServicePort))
    if err != nil {
        logrus.Errorf("could not create account client: %v", err)
    }
    
    account, err := accountClient.CreateAccount("uwbollema", "uwbollmausername","uwbollema@icloud.com")
    if err != nil {
        logrus.Errorf("could not create account: %v", err)
    }
    logrus.Infof("created account: %v", account)

	obs := objectstore.NewObjectStore()
	bytes, _ := obs.GetFileFromBucket("root", "private-key.pem")
	rootPrivateKey, err := crypto.ParseRSAPrivateKeyFromPEM(*bytes)
	if err != nil {
		logrus.Error(err)
		panic(err)
	}

	// TODO: assertions next step
	rootAuthorityId := config.MustGetString(configkey.RootAuthority)
	if rootAuthorityId == "" {
		panic("Root authority id is not set")
	}
	signingDB := assertstest.NewSigningDB(rootAuthorityId, asserts.RSAPrivateKey(rootPrivateKey))

	bytes, _ = obs.GetFileFromBucket("generic", "private-key.pem")
	genericPrivateKey, err := crypto.ParseRSAPrivateKeyFromPEM(*bytes)
	if err != nil {
		logrus.Error(err)
		panic(err)
	}

	err = signingDB.ImportKey(asserts.RSAPrivateKey(genericPrivateKey))
	if err != nil {
		panic(err)
	}

	handler := store.NewHandler(repositories.NewAccountRepository(db), repositories.NewSnapsRepository(db), obs)
	store := store.New(handler, assertsDatabase, rootPrivateKey, genericPrivateKey, signingDB, rootAuthorityId)
	if store == nil {
		panic("store was not created, cannot continue")
	}
	store.SetupEndpoints(r)

	// Make sure all the necessary buckets exists
	err = objectstore.GetMinioClient().MakeBucket(context.Background(), "snaps", minio.MakeBucketOptions{})
	if err != nil {
		if _, ok := err.(minio.ErrorResponse); !ok {
			panic(err)
		}
	}

	err = objectstore.GetMinioClient().MakeBucket(context.Background(), "unscanned", minio.MakeBucketOptions{})
	if err != nil {
		if _, ok := err.(minio.ErrorResponse); !ok {
			panic(err)
		}
	}

	_ = r.Run()
}

var databaseCreationMutex sync.Mutex

func GetDatabaseWithRootKey() *asserts.Database {
	minioClient := objectstore.GetMinioClient()

	databaseCreationMutex.Lock()
	defer databaseCreationMutex.Unlock()

	return GetDatabaseWithRootKeyS3(minioClient)
}

func GetDatabaseWithRootKeyS3(minioClient *minio.Client) *asserts.Database {
	databaseCfg, err := getDatabaseConfig(minioClient)
	if err != nil {
		panic(err)
	}

	db, err := asserts.OpenDatabase(databaseCfg)
	if err != nil {
		panic(err)
	}

	buckets := []string{"root", "generic"}
	ctx, cancel := context.WithCancel(context.Background())

	defer cancel()

	for _, bucket := range buckets {
		objectCh := minioClient.ListObjects(ctx, bucket, minio.ListObjectsOptions{
			Recursive: true,
		})
		for object := range objectCh {
			if strings.Contains(object.Key, "pem") {
				objectPtr, err := minioClient.GetObject(ctx, bucket, object.Key, minio.GetObjectOptions{})
				if err != nil {
					panic(err)
				}
				bytes, _ := io.ReadAll(objectPtr)

				rsaPK, err := crypto.ParseRSAPrivateKeyFromPEM(bytes)
				if err != nil {
					panic(err)
				}

				assertPK := asserts.RSAPrivateKey(rsaPK)

				err = db.ImportKey(assertPK)
				if err != nil {
					panic(err)
				}
			}
		}
	}

	return db
}

func getDatabaseConfig(minioClient *minio.Client) (*asserts.DatabaseConfig, error) {
	var trusted []asserts.Assertion
	var otherPredefined []asserts.Assertion
	buckets := []string{"root", "generic"}
	ctx, cancel := context.WithCancel(context.Background())

	defer cancel()

	for _, bucket := range buckets {
		objectCh := minioClient.ListObjects(ctx, bucket, minio.ListObjectsOptions{
			Recursive: true,
		})
		for object := range objectCh {
			if strings.Contains(object.Key, "assertion") {
				logrus.Tracef("Assertion key: %s", object.Key)
				filename := object.Key
				logrus.Tracef("Assertion filename: %s", filename)

				objectPtr, err := minioClient.GetObject(ctx, bucket, object.Key, minio.GetObjectOptions{})
				if err != nil {
					panic(err)
				}

				assertionBytes, _ := io.ReadAll(objectPtr)
				logrus.Trace("assertion:")
				logrus.Trace(string(assertionBytes))
				assertion, err := asserts.Decode(assertionBytes)
				if err != nil {
					panic(err)
				} else {
					logrus.Tracef("assertion type: %s", assertion.Type().Name)

					if assertion.Type() == asserts.AccountKeyType {
						trusted = append(trusted, assertion)
					} else if assertion.Type() == asserts.AccountType {
						trusted = append(trusted, assertion)
					} else {
						otherPredefined = append(otherPredefined, assertion)
					}
				}
			}
		}
	}

	cfg := asserts.DatabaseConfig{
		Trusted:         trusted,
		OtherPredefined: otherPredefined,
		Backstore:       asserts.NewMemoryBackstore(),
		KeypairManager:  asserts.NewMemoryKeypairManager(),
		Checkers:        nil,
	}

	return &cfg, nil
}
