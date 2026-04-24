package config

import (
	"os"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

// createTempConfigFile writes the provided content to a temporary file and returns its name.
// It also registers a cleanup function to remove the temporary file.
func createTempConfigFile(t *testing.T, content string) string {
	tmpFile, err := os.CreateTemp("", "testconfig-*.yaml")
	assert.NoError(t, err, "expected to create temp file without error")

	_, err = tmpFile.WriteString(content)
	assert.NoError(t, err, "expected to write to temp file without error")

	err = tmpFile.Close()
	assert.NoError(t, err, "expected to close temp file without error")

	t.Cleanup(func() { os.Remove(tmpFile.Name()) })
	return tmpFile.Name()
}

func TestLoadConfig_NoEnv(t *testing.T) {
	// Unset CONFIG_FILE_PATH to simulate missing environment variable.
	os.Unsetenv("CONFIG_FILE_PATH")
	cfg, err := LoadConfig()
	assert.Nil(t, cfg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "CONFIG_FILE_PATH is not set")
}

func TestLoadConfig_InvalidYAML(t *testing.T) {
	invalidYAML := "this: is: not: valid: yaml: -"
	tmpFile := createTempConfigFile(t, invalidYAML)
	os.Setenv("CONFIG_FILE_PATH", tmpFile)
	defer os.Unsetenv("CONFIG_FILE_PATH")

	// Reset viper to avoid contamination from previous tests.
	viper.Reset()

	cfg, err := LoadConfig()
	assert.Nil(t, cfg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read config file")
}

func TestLoadConfig_MissingMacaroon(t *testing.T) {
	// YAML config without macaroon block.
	content := `
debug_mode: false
account_service_host: "localhost"
account_service_port: 8080
store_service_host: "localhost"
store_service_port: 8081
assertion_service_host: "localhost"
assertion_service_port: 8082
store_url: "https://store.example.com"
`
	tmpFile := createTempConfigFile(t, content)
	os.Setenv("CONFIG_FILE_PATH", tmpFile)
	defer os.Unsetenv("CONFIG_FILE_PATH")
	viper.Reset()

	cfg, err := LoadConfig()
	assert.Nil(t, cfg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "macaroon config is required")
}

func TestLoadConfig_MissingDischargeKey(t *testing.T) {
	// YAML config where macaroon.discharge_key is empty.
	content := `
debug_mode: false
account_service_host: "localhost"
account_service_port: 8080
store_service_host: "localhost"
store_service_port: 8081
assertion_service_host: "localhost"
assertion_service_port: 8082
macaroon:
  root_key: "some-root-key"
  root_id: "some-root-id"
  root_location: "some-root-location"
  discharge_key: ""
  third_party_caveat_id: "some-third-party-caveat-id"
  third_party_location: "some-third-party-location"
store_url: "https://store.example.com"
`
	tmpFile := createTempConfigFile(t, content)
	os.Setenv("CONFIG_FILE_PATH", tmpFile)
	defer os.Unsetenv("CONFIG_FILE_PATH")
	viper.Reset()

	cfg, err := LoadConfig()
	assert.Nil(t, cfg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "macaroon.discharge_key is required")
}

func TestLoadConfig_MissingRootKey(t *testing.T) {
	// YAML config where macaroon.root_key is empty.
	content := `
debug_mode: false
account_service_host: "localhost"
account_service_port: 8080
store_service_host: "localhost"
store_service_port: 8081
assertion_service_host: "localhost"
assertion_service_port: 8082
macaroon:
  root_key: ""
  root_id: "some-root-id"
  root_location: "some-root-location"
  discharge_key: "some-discharge-key"
  third_party_caveat_id: "some-third-party-caveat-id"
  third_party_location: "some-third-party-location"
store_url: "https://store.example.com"
`
	tmpFile := createTempConfigFile(t, content)
	os.Setenv("CONFIG_FILE_PATH", tmpFile)
	defer os.Unsetenv("CONFIG_FILE_PATH")
	viper.Reset()

	cfg, err := LoadConfig()
	assert.Nil(t, cfg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "macaroon.root_key is required")
}

func TestLoadConfig_MissingStoreUrl(t *testing.T) {
	// YAML config with an empty store_url.
	content := `
debug_mode: false
account_service_host: "localhost"
account_service_port: 8080
store_service_host: "localhost"
store_service_port: 8081
assertion_service_host: "localhost"
assertion_service_port: 8082
macaroon:
  root_key: "some-root-key"
  root_id: "some-root-id"
  root_location: "some-root-location"
  discharge_key: "some-discharge-key"
  third_party_caveat_id: "some-third-party-caveat-id"
  third_party_location: "some-third-party-location"
store_url: ""
`
	tmpFile := createTempConfigFile(t, content)
	os.Setenv("CONFIG_FILE_PATH", tmpFile)
	defer os.Unsetenv("CONFIG_FILE_PATH")
	viper.Reset()

	cfg, err := LoadConfig()
	assert.Nil(t, cfg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "store_url is required")
}

func TestLoadConfig_MissingCohortSigningKey(t *testing.T) {
	content := `
debug_mode: false
account_service_host: "localhost"
account_service_port: 8080
store_service_host: "localhost"
store_service_port: 8081
assertion_service_host: "localhost"
assertion_service_port: 8082
macaroon:
  root_key: "some-root-key"
  root_id: "some-root-id"
  root_location: "some-root-location"
  discharge_key: "some-discharge-key"
  third_party_caveat_id: "some-third-party-caveat-id"
  third_party_location: "some-third-party-location"
store_url: "https://store.example.com"
store_name: "example-store"
`
	tmpFile := createTempConfigFile(t, content)
	os.Setenv("CONFIG_FILE_PATH", tmpFile)
	defer os.Unsetenv("CONFIG_FILE_PATH")
	viper.Reset()

	cfg, err := LoadConfig()
	assert.Nil(t, cfg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cohort_signing_key is required")
}

func TestLoadConfig_Success(t *testing.T) {
	content := `
debug_mode: false
account_service_host: "localhost"
account_service_port: 8080
store_service_host: "localhost"
store_service_port: 8081
assertion_service_host: "localhost"
assertion_service_port: 8082
macaroon:
  root_key: "some-root-key"
  root_id: "some-root-id"
  root_location: "some-root-location"
  discharge_key: "some-discharge-key"
  third_party_caveat_id: "some-third-party-caveat-id"
  third_party_location: "some-third-party-location"
store_url: "https://store.example.com"
store_name: "example-store"
cohort_signing_key: "example_key"
`
	tmpFile := createTempConfigFile(t, content)
	os.Setenv("CONFIG_FILE_PATH", tmpFile)
	defer os.Unsetenv("CONFIG_FILE_PATH")
	viper.Reset()

	cfg, err := LoadConfig()
	assert.NoError(t, err)
	assert.NotNil(t, cfg)
	// Verify a few fields.
	assert.Equal(t, false, cfg.DebugMode)
	assert.Equal(t, "localhost", cfg.AccountServiceHost)
	assert.Equal(t, 8080, cfg.AccountServicePort)
	// Check macaroon config.
	assert.NotNil(t, cfg.MacaroonConfig)
	assert.Equal(t, "some-root-key", cfg.MacaroonConfig.RootKey)
	assert.Equal(t, "some-discharge-key", cfg.MacaroonConfig.DischargeKey)
	assert.Equal(t, "https://store.example.com", cfg.StoreUrl)
	assert.Equal(t, "example-store", cfg.StoreName)
	assert.Equal(t, "example_key", cfg.CohortSigningKey)
}
