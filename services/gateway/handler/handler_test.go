package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	accountClient "github.com/idlab-discover/kebeng/services/account/client"
	"github.com/idlab-discover/kebeng/services/gateway/internal/config"
	"github.com/idlab-discover/kebeng/services/gateway/internal/message"
	storeClient "github.com/idlab-discover/kebeng/services/store/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Mock AccountClient
type MockAccountClient struct {
	mock.Mock
}

func (m *MockAccountClient) CreateAccount(displayName, username, email string) (*message.CreateAccountResponse, error) {
	args := m.Called(displayName, username, email)
	return args.Get(0).(*message.CreateAccountResponse), args.Error(1)
}

// Mock StoreClient
type MockStoreClient struct {
	mock.Mock
}

// func TestCreateAccount(t *testing.T) {
// 	gin.SetMode(gin.TestMode)

// 	accountClient := new(MockAccountClient)
// 	storeClient := new(MockStoreClient)
// 	config := &config.Config{}

// 	handler := NewHandler(accountClient, storeClient, config)

// 	router := gin.Default()
// 	handler.SetupEndpoints(router)

// 	t.Run("CreateAccount_Success", func(t *testing.T) {
// 		reqBody := message.CreateAccountRequest{
// 			DisplayName: "Test User",
// 			Username:    "testuser",
// 			Email:       "testuser@example.com",
// 		}
// 		body, _ := json.Marshal(reqBody)

// 		req, _ := http.NewRequest(http.MethodPost, "/createAccount", bytes.NewBuffer(body))
// 		req.Header.Set("Content-Type", "application/json")

// 		w := httptest.NewRecorder()
// 		router.ServeHTTP(w, req)

// 		assert.Equal(t, http.StatusOK, w.Code)
// 	})

// 	t.Run("CreateAccount_BadRequest", func(t *testing.T) {
// 		reqBody := map[string]interface{}{
// 			"invalidField": "invalidValue",
// 		}
// 		body, _ := json.Marshal(reqBody)

// 		req, _ := http.NewRequest(http.MethodPost, "/createAccount", bytes.NewBuffer(body))
// 		req.Header.Set("Content-Type", "application/json")

// 		w := httptest.NewRecorder()
// 		router.ServeHTTP(w, req)

// 		assert.Equal(t, http.StatusBadRequest, w.Code)
// 	})
// }

func TestRegisterSnapName(t *testing.T) {
	gin.SetMode(gin.TestMode)

	accountClient := &accountClient.AccountClient{}
	storeClient := new(MockStoreClient)
	config := &config.Config{}

	handler := NewHandler(accountClient, storeClient, config)

	router := gin.Default()
	handler.SetupEndpoints(router)

	t.Run("RegisterSnapName_Success", func(t *testing.T) {
		reqBody := message.RegisterSnapNameReq{
			SnapName:  "test-snap",
			IsPrivate: true,
			Store:     "test-store",
		}
		body, _ := json.Marshal(reqBody)

		req, _ := http.NewRequest(http.MethodPost, "/dev/api/register-name/", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
	})

	t.Run("RegisterSnapName_BadRequest", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"invalidField": "invalidValue",
		}
		body, _ := json.Marshal(reqBody)

		req, _ := http.NewRequest(http.MethodPost, "/dev/api/register-name/", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestRegisterSnapNameDispute(t *testing.T) {
	gin.SetMode(gin.TestMode)

	accountClient := &accountClient.AccountClient{}
	storeClient := &storeClient.StoreClient{}
	config := &config.Config{}

	handler := NewHandler(accountClient, storeClient, config)

	router := gin.Default()
	handler.SetupEndpoints(router)

	t.Run("RegisterSnapNameDispute_NotImplemented", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPost, "/dev/api/register-name-dispute/", nil)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotImplemented, w.Code)
	})
}

// func TestGenerateMacaroon(t *testing.T) {
// 	gin.SetMode(gin.TestMode)

// 	accountClient := &accountClient.AccountClient{}
// 	storeClient := &storeClient.StoreClient{}
// 	config := &config.Config{}

// 	handler := NewHandler(accountClient, storeClient, config)

// 	router := gin.Default()
// 	handler.SetupEndpoints(router)

// 	t.Run("GenerateMacaroon_Success", func(t *testing.T) {
// 		reqBody := message.GenerateMacaroonRequest{
// 			Packages: []message.Package{
// 				{Name: "test-package", SnapId: "test-snap-id"},
// 			},
// 		}
// 		body, _ := json.Marshal(reqBody)

// 		req, _ := http.NewRequest(http.MethodPost, "/dev/api/acl/", bytes.NewBuffer(body))
// 		req.Header.Set("Content-Type", "application/json")

// 		w := httptest.NewRecorder()
// 		router.ServeHTTP(w, req)

// 		assert.Equal(t, http.StatusOK, w.Code)
// 	})

// 	t.Run("GenerateMacaroon_BadRequest", func(t *testing.T) {
// 		reqBody := map[string]interface{}{
// 			"invalidField": "invalidValue",
// 		}
// 		body, _ := json.Marshal(reqBody)

// 		req, _ := http.NewRequest(http.MethodPost, "/dev/api/acl/", bytes.NewBuffer(body))
// 		req.Header.Set("Content-Type", "application/json")

// 		w := httptest.NewRecorder()
// 		router.ServeHTTP(w, req)

// 		assert.Equal(t, http.StatusBadRequest, w.Code)
// 	})
// }

func TestProcessSnapBuildAssertion(t *testing.T) {
	gin.SetMode(gin.TestMode)

	accountClient := &accountClient.AccountClient{}
	storeClient := &storeClient.StoreClient{}
	config := &config.Config{}

	handler := NewHandler(accountClient, storeClient, config)

	router := gin.Default()
	handler.SetupEndpoints(router)

	t.Run("ProcessSnapBuildAssertion_Success", func(t *testing.T) {
		reqBody := message.SnapBuildAssertionReq{
			Assertion: []byte("test-assertion"),
		}
		body, _ := json.Marshal(reqBody)

		req, _ := http.NewRequest(http.MethodPost, "/dev/api/snaps/test-snap-id/builds", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("ProcessSnapBuildAssertion_BadRequest", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"invalidField": "invalidValue",
		}
		body, _ := json.Marshal(reqBody)

		req, _ := http.NewRequest(http.MethodPost, "/dev/api/snaps/test-snap-id/builds", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}
