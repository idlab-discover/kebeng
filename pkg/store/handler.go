package store

import (
	"crypto/rsa"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"strconv"

	"github.com/idlab-discover/kebeng/pkg/database"
	"github.com/idlab-discover/kebeng/pkg/models"

	"github.com/google/uuid"

	"github.com/snapcore/snapd/asserts/assertstest"

	asserts2 "github.com/idlab-discover/kebeng/pkg/store/asserts"

	//	"github.com/idlab-discover/kebeng/config"
	//	"github.com/idlab-discover/kebeng/config/configkey"

	"github.com/snapcore/snapd/asserts"

	"github.com/idlab-discover/kebeng/pkg/objectstore"

	"github.com/idlab-discover/kebeng/pkg/store/requests"

	"github.com/snapcore/snapd/snap"

	"github.com/sirupsen/logrus"

	"github.com/idlab-discover/kebeng/pkg/repositories"
	"github.com/idlab-discover/kebeng/pkg/store/responses"
)

type IStoreHandler interface {
	GetSections() (*responses.SectionResults, error)
	GetSnapNames() (*responses.CatalogResults, error)
	FindSnap(name string) (*responses.SearchV2Results, error)
	SnapRefresh(actions *[]*requests.SnapActionJSON) (*responses.SnapActionResultList, error)
	SnapDownload(snapFilename string) (*[]byte, error)
	GetSnapRevisionAssertion(SHA3384Encoded string, rootStoreKey *rsa.PrivateKey, assertsDB *asserts.Database, storeAuthorityId string) (*asserts.SnapRevision, error)
	GetSnapDeclarationAssertion(snapId string, rootStoreKey *rsa.PrivateKey, assertsDB *asserts.Database, storeAuthorityId string) (*asserts.SnapDeclaration, error)
	GetAccountKeyAssertion(keySHA3384 string, rootStoreKey *rsa.PrivateKey, signingDB *assertstest.SigningDB) (*asserts.AccountKey, error)
	GetAccountAssertion(accountId string, rootStoreKey *rsa.PrivateKey, signingDB *assertstest.SigningDB) (*asserts.Account, error)
	UnscannedUpload(snapFile io.Reader) (string, error)
	AuthRequest() *responses.AuthRequestIDResp
	AuthDevice(serialRequest *asserts.SerialRequest, genericPrivateKey asserts.PrivateKey, signingDB *assertstest.SigningDB) (*asserts.Serial, error)
	AuthNonce() *responses.Nonce
	AuthSession() *responses.Session
}

// TODO add objectstore to the handler
type Handler struct {
	accounts repositories.IAccountRepository
	snaps    repositories.ISnapsRepository
	obs      objectstore.ObjectStore
}

func NewHandler(accts repositories.IAccountRepository, snaps repositories.ISnapsRepository, obs objectstore.ObjectStore) *Handler {
	return &Handler{
		accts,
		snaps,
		obs,
	}
}

func (h *Handler) AuthRequest() *responses.AuthRequestIDResp {
	nextAuthRequest := models.AuthRequest{}
	database.DB.Save(&nextAuthRequest)

	resp := &responses.AuthRequestIDResp{RequestID: strconv.Itoa(int(nextAuthRequest.ID))}
	return resp
}

func (h *Handler) AuthDevice(serialRequest *asserts.SerialRequest, genericPrivateKey asserts.PrivateKey, signingDB *assertstest.SigningDB) (*asserts.Serial, error) {
	// TODO: this private key needs to be handled differently

	// TODO: store session information in the database
	serial := uuid.New().String()
	encodedKeyBytes, err := asserts.EncodePublicKey(serialRequest.DeviceKey())
	if err != nil {
		panic(err)
	}

	serialHeaders := map[string]interface{}{
		"brand-id":            serialRequest.BrandID(),
		"model":               serialRequest.Model(),
		"authority-id":        serialRequest.BrandID(),
		"serial":              serial,
		"device-key":          string(encodedKeyBytes),
		"device-key-sha3-384": serialRequest.DeviceKey().ID(),
		// TODO: fix this static timestamp
		"timestamp": "2021-03-06T15:04:00Z",
	}

	// TODO: look up actual account, get key, etc.
	assertion, err := signingDB.Sign(asserts.SerialType, serialHeaders, nil, genericPrivateKey.PublicKey().ID())

	if err == nil && assertion != nil {
		if serialAssertion, ok := assertion.(*asserts.Serial); ok {
			return serialAssertion, nil
		} else {
			return nil, errors.New("unable to assert type on serial assertion")
		}
	} else if err != nil {
		logrus.Error(err)
		return nil, err
	}

	return nil, errors.New("unknown error encountered trying to get serial assertion")
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
	err = h.obs.SaveFileToBucket("unscanned", tmpPath)
	if err != nil {
		logrus.Errorf("Failed to save file to object store: %v", err)
		return "", err
	}

	return id, nil
}

func (h *Handler) AuthSession() *responses.Session {
	session := &responses.Session{Macaroon: "12345-678-9101112"}
	return session
}

func (h *Handler) AuthNonce() *responses.Nonce {
	nonce := responses.Nonce{Nonce: uuid.New().String()}
	return &nonce
}

func (h *Handler) GetAccountKeyAssertion(keySHA3384 string, rootStoreKey *rsa.PrivateKey, signingDB *assertstest.SigningDB) (*asserts.AccountKey, error) {
	accountKey, err := h.accounts.GetKeyBySHA3384(keySHA3384)
	if err == nil && accountKey != nil {
		logrus.Tracef("Found account-key: %+v", accountKey)

		bytes, err2 := base64.StdEncoding.DecodeString(accountKey.EncodedPublicKey)
		if err2 != nil {
			panic(err2)
		}

		pbk, err2 := asserts.DecodePublicKey([]byte(bytes))
		if err2 != nil {
			panic(err2)
		}

		trustedAcct := getTrustedAccount(accountKey.Account.AccountId, signingDB, accountKey.Account.DisplayName)

		// TODO: what do do about these dates?
		trustedAcctKeyHeaders := map[string]interface{}{
			"since":               "2015-11-20T15:04:00Z",
			"until":               "2500-11-20T15:04:00Z",
			"public-key-sha3-384": accountKey.SHA3384,
			"name":                accountKey.Name,
		}
		//
		trustedAccKey := assertstest.NewAccountKey(signingDB, trustedAcct, trustedAcctKeyHeaders, pbk, "")
		if trustedAccKey != nil {
			return trustedAccKey, nil
		}
	} else if err != nil {
		logrus.Error(err)
		return nil, err
	}

	return nil, errors.New("account key could not be found or there was an error")
}

func (h *Handler) GetAccountAssertion(accountId string, rootStoreKey *rsa.PrivateKey, signingDB *assertstest.SigningDB) (*asserts.Account, error) {
	account, err := h.accounts.GetAccountById(accountId, false)
	if err == nil && account != nil {
		//
		pk := asserts.RSAPrivateKey(rootStoreKey)
		acct := createAccountAssertion(signingDB, pk.PublicKey().ID(), account.AccountId, account.Username)
		return acct, nil
	} else if err != nil {
		return nil, err
	}

	logrus.Errorf("Unknown error, could not find account: %s", accountId)
	return nil, errors.New("account not found")
}

func (h *Handler) GetSnapDeclarationAssertion(snapId string, rootStoreKey *rsa.PrivateKey, assertsDB *asserts.Database, storeAuthorityId string) (*asserts.SnapDeclaration, error) {
	logrus.Tracef("Requested snap-declaration: %s", snapId)

	snapEntry, err := h.snaps.GetSnapByStoreId(snapId, true)
	if err != nil {
		logrus.Errorf("Failed to get snap entry for snap-id %s: %v", snapId, err)
		return nil, err
	}
	if snapEntry == nil {
		logrus.Warnf("No snap entry found for snap-id: %s", snapId)
		return nil, errors.New("snap entry not found")
	}

	// TODO: this again seems wrong, you should not create a new assertion and sign it again
	assertion, err := asserts2.MakeSnapDeclarationAssertion(
		storeAuthorityId,
		snapEntry.Account.AccountId,
		snapEntry,
		asserts.RSAPrivateKey(rootStoreKey),
		assertsDB,
	)
	if err != nil {
		logrus.Errorf("Failed to create snap-declaration assertion: %v", err)
		return nil, err
	}
	if assertion == nil {
		logrus.Warn("Assertion creation returned nil")
		return nil, errors.New("failed to create snap-declaration assertion")
	}

	return assertion, nil
}

func (h *Handler) GetSnapRevisionAssertion(SHA3384Encoded string, rootStoreKey *rsa.PrivateKey, assertsDB *asserts.Database, storeAuthorityId string) (*asserts.SnapRevision, error) {
	revision, err := h.snaps.GetRevisionBySHA(SHA3384Encoded, true)
	if err != nil {
		logrus.Errorf("Failed to get revision by SHA: %s", err)
		return nil, err
	}
	if revision == nil {
		logrus.Warnf("Revision not found for SHA: %s", SHA3384Encoded)
		return nil, fmt.Errorf("revision not found for SHA: %s", SHA3384Encoded)
	}

	snapEntry, err := h.snaps.GetSnapById(revision.SnapEntryID, true)
	if err != nil {
		logrus.Errorf("Failed to get snap by id: %s", err)
		return nil, err
	}
	if snapEntry == nil {
		logrus.Warnf("Snap entry not found for revision: %d", revision.ID)
		return nil, fmt.Errorf("snap entry not found for revision: %d", revision.ID)
	}

	// TODO: this is wrong i think, i don't think you are supposed to create a new assertion and sign it again
	// whenever the assertion is created (here for upload) you sign it then and store it somewhere (i think snapd database asserts.Database)
	// could store it in the database aswel (kinda what already happens? idk why you would sign it again)
	// here you just fetch the assertion and return it CHECK THIS!!!!
	assertion, err := asserts2.MakeSnapRevisionAssertion(
		storeAuthorityId,
		SHA3384Encoded,
		snapEntry.SnapStoreID,
		revision.Size,
		int(revision.ID),
		snapEntry.Account.AccountId,
		asserts.RSAPrivateKey(rootStoreKey).PublicKey().ID(),
		assertsDB,
	)

	if err != nil {
		logrus.Errorf("Failed to make snap revision assertion: %s", err)
		return nil, err
	}

	return assertion, nil
}

func (h *Handler) SnapDownload(snapFilename string) (*[]byte, error) {
	bytes, err := h.obs.GetFileFromBucket("snaps", snapFilename)

	if err != nil {
		logrus.Error(err)
		return nil, fmt.Errorf("error fetching snap: %w", err)
	}

	if bytes != nil && len(*bytes) == 0 {
		return nil, fmt.Errorf("snap file %s not found", snapFilename)
	}

	return bytes, nil
}

func (h *Handler) SnapRefresh(actions *[]*requests.SnapActionJSON) (*responses.SnapActionResultList, error) {
	var actionResults []*responses.SnapActionResult
	for _, action := range *actions {
		snapEntry, err := h.snaps.GetSnapByName(action.Name, true)
		if err == nil && snapEntry != nil {
			// TODO: support other actions "refresh", etc.
			if action.Action == "download" {
				logrus.Infof("We know about this snap %s, its id is %s we we'll try to handle it.", snapEntry.Name, snapEntry.SnapStoreID)

				snapRevision, err2 := h.snaps.GetRevisionByChannelAndTrack(action.Channel, action.Name)
				if err2 == nil && snapRevision != nil {
					storeSnap, err3 := snapEntry.ToStoreSnap(snapRevision)
					if err3 == nil && storeSnap != nil {
						actionResult := responses.SnapActionResult{
							Result:      "download",
							InstanceKey: "download-1",
							SnapID:      snapEntry.SnapStoreID,
							Name:        snapEntry.Name,
							Snap:        storeSnap,
						}

						actionResults = append(actionResults, &actionResult)
					}
					logrus.Errorf("unable to process action %s for snap %s: %s", action.Action, action.Name, err3)
				}
			} else if action.Action == "install" {
				logrus.Infof("We know about this snap %s, its id is %s we we'll try to handle it.", snapEntry.Name, snapEntry.SnapStoreID)
				snapRevision, err2 := h.snaps.GetRevisionByChannelAndTrack(action.Channel, action.Name)
				if err2 == nil && snapRevision != nil {
					storeSnap, err3 := snapEntry.ToStoreSnap(snapRevision)
					if err3 == nil && storeSnap != nil {
						// TODO: this shouldn't be a fixed architecture
						storeSnap.Architectures = []string{"amd64"}
						storeSnap.Confinement = snapEntry.Confinement

						actionResult := responses.SnapActionResult{
							Result:      "install",
							InstanceKey: "install-1",
							SnapID:      snapEntry.SnapStoreID,
							Name:        snapEntry.Name,
							Snap:        storeSnap,
						}

						actionResults = append(actionResults, &actionResult)
					}
				}
			}
		} else if err != nil {
			logrus.Error(err)
		} else {
			logrus.Errorf("cannot process action %s for %s, snap unknown", action.Action, action.Name)
		}
	}

	actionResultList := responses.SnapActionResultList{
		Results:   actionResults,
		ErrorList: nil,
	}

	return &actionResultList, nil
}

func (h *Handler) FindSnap(name string) (*responses.SearchV2Results, error) {
	searchResult := responses.SearchV2Results{
		ErrorList: nil,
	}

	snapEntry, err := h.snaps.GetSnapByName(name, true)
	if err == nil && snapEntry != nil {
		results := func() []responses.StoreSearchResult {
			var results []responses.StoreSearchResult

			snapType := snap.TypeApp
			switch snapEntry.Type {
			case "os":
				snapType = snap.TypeOS
			case "snapd":
				snapType = snap.TypeSnapd
			case "base":
				snapType = snap.TypeBase
			case "gadget":
				snapType = snap.TypeGadget
			case "kernel":
				snapType = snap.TypeKernel
			}

			results = append(results, responses.StoreSearchResult{
				Revision: responses.StoreSearchChannelSnap{
					StoreSnap: responses.StoreSnap{
						Confinement: snapEntry.Confinement,
						CreatedAt:   snapEntry.CreatedAt.String(),
						Name:        snapEntry.Name,
						// TODO: need to fix this properly
						Revision:  1,
						SnapID:    snapEntry.SnapStoreID,
						Type:      snapType,
						Publisher: snap.StoreAccount{ID: snapEntry.Account.AccountId, Username: snapEntry.Account.Username, DisplayName: snapEntry.Account.DisplayName},
					},
				},
				Snap: responses.StoreSnap{
					Confinement: snapEntry.Confinement,
					CreatedAt:   snapEntry.CreatedAt.String(),
					Name:        snapEntry.Name,
					// TODO: need to fix this properly
					Revision:  1,
					SnapID:    snapEntry.SnapStoreID,
					Type:      snapType,
					Publisher: snap.StoreAccount{ID: snapEntry.Account.AccountId, Username: snapEntry.Account.Username, DisplayName: snapEntry.Account.DisplayName},
				},
				Name:   snapEntry.Name,
				SnapID: snapEntry.SnapStoreID,
			})

			return results
		}()

		searchResult.Results = results
		return &searchResult, nil
	} else if err != nil {
		logrus.Error(err)
		return nil, err
	}

	return nil, errors.New("unknown error encountered in FindSnap")
}

func (h *Handler) GetSnapNames() (*responses.CatalogResults, error) {
   snaps, err := h.snaps.GetSnaps(&models.SnapFilter{})
   if err != nil {
      return nil, err
   }

   items := make([]responses.CatalogItem, len(snaps))
   for _, snap := range snaps {
      item := responses.CatalogItem{
         Name: snap.Name,
         Version: "none provided",  // TODO: implement version
         Summary: "none provided",  // TODO: implement summary
         Aliases: nil,              // TODO: implement aliases
         Apps: nil,                 // TODO: implement apps
         Title: "none provided",    // TODO: implement title
      }
      items = append(items, item)
   }

   result := responses.CatalogResults{
      Payload: responses.CatalogPayload{
         Items: items,
      },
   }

   return &result, nil
}

func (h *Handler) GetSections() (*responses.SectionResults, error) {
	sections, err := h.snaps.GetSections()
	if err == nil && sections != nil {
		results := responses.SectionResults{
			Payload: responses.Payload{
				Sections: []responses.Section{
					{Name: "general"},
				},
			},
		}

		return &results, nil
	}

	return nil, errors.New("unknown error")
}

func createAccountAssertion(signingDB *assertstest.SigningDB, keyId string, accountId string, storeAccountUsername string) *asserts.Account {
	trustedAcctHeaders := map[string]interface{}{
		"validation": "certified",
		"timestamp":  "2015-11-20T15:04:00Z",
		"account-id": accountId,
	}

	trustedAcct := assertstest.NewAccount(signingDB, storeAccountUsername, trustedAcctHeaders, keyId)
	return trustedAcct
}

func getTrustedAccount(accountID string, signingDB *assertstest.SigningDB, displayName string) *asserts.Account {
	trustedAcctHeaders := map[string]interface{}{
		"validation": "verified",
		"timestamp":  "2015-11-20T15:04:00Z",
	}

	if displayName != "" {
		trustedAcctHeaders["display-name"] = displayName
	}

	trustedAcctHeaders["account-id"] = accountID
	trustedAcct := assertstest.NewAccount(signingDB, accountID, trustedAcctHeaders, "")

	return trustedAcct
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
