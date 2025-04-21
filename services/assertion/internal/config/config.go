package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/google/uuid"
	"github.com/idlab-discover/kebeng/common/crypto"
	"github.com/sirupsen/logrus"
	"github.com/snapcore/snapd/asserts"
	"github.com/spf13/viper"
)

type Config struct {
	DBHost        string `mapstructure:"db_host" yaml:"db_host"`
	DBPort        int    `mapstructure:"db_port" yaml:"db_port"`
	DBUser        string `mapstructure:"db_user" yaml:"db_user"`
	DBPassword    string `mapstructure:"db_password" yaml:"db_password"`
	DBName        string `mapstructure:"db_name" yaml:"db_name"`
	GRPCHost      string `mapstructure:"grpc_host" yaml:"grpc_host"`
	GRPCPort      int    `mapstructure:"grpc_port" yaml:"grpc_port"`
	MigrationPath string `mapstructure:"migration_path" yaml:"migration_path"`

	RootKey             asserts.PrivateKey
	RootKeyPath         string `mapstructure:"root_key_path" yaml:"root_key_path"`
	rootAccountIDString string `mapstructure:"root_account_id" yaml:"root_account_id"`
	RootAccountID       uuid.UUID

	AuthorityID string `mapstructure:"authority_id" yaml:"authority_id"`
	StoreName   string `mapstructure:"store_name" yaml:"store_name"`

	TestMode bool
}

func LoadConfig() (*Config, error) {
	configPath := os.Getenv("CONFIG_FILE_PATH")
	if configPath == "" {
		return nil, fmt.Errorf("CONFIG_FILE_PATH is not set")
	}

	logrus.Infof("loading config from %s", configPath)
	viper.SetConfigFile(configPath)
	viper.SetConfigType("yaml")

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %v", err)
	}

	cfg := &Config{}
	if err := viper.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %v", err)
	}

	testMode := os.Getenv("TEST_MODE")
	if testMode == "1" {
		logrus.Info("running in test mode")
		cfg.TestMode = true
	}

	if err := cfg.checkConfig(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %v", err)
	}

	// Load the root key from the file.
	rootKey, cerr := crypto.GetPrivateKeyFromPEMFile(cfg.RootKeyPath)
	if cerr != nil {
		return nil, fmt.Errorf("failed to load root key from %s: %v", cfg.RootKeyPath, cerr)
	}
	if rootKey == nil {
		return nil, fmt.Errorf("failed to load root key from %s", cfg.RootKeyPath)
	}
	cfg.RootKey = rootKey

	// Parse the root account ID from string to UUID.
	rootAccountID, err := uuid.Parse(cfg.rootAccountIDString)
	if err != nil {
		return nil, fmt.Errorf("failed to parse root account ID %s: %v", cfg.rootAccountIDString, err)
	}
	cfg.RootAccountID = rootAccountID

	logrus.Infof("loaded config: %+v", cfg)

	return cfg, nil
}

func GetAssertionServiceAddress(host string, port int) string {
	if host == "" || port == 0 {
		logrus.Warn("host or port is not set, using default address assertion_service:8083")
		return "assertion_service:8083"
	}

	return fmt.Sprintf("%s:%d", host, port)
}

func (c *Config) checkConfig() error {
	var errs []string

	// Check DB config.
	if c.DBHost == "" {
		errs = append(errs, "DBHost is required")
	}
	if c.DBPort <= 0 {
		errs = append(errs, "DBPort must be a positive integer")
	}
	if c.DBUser == "" {
		errs = append(errs, "DBUser is required")
	}
	if c.DBPassword == "" {
		errs = append(errs, "DBPassword is required")
	}
	if c.DBName == "" {
		errs = append(errs, "DBName is required")
	}

	// Check gRPC config.
	if c.GRPCHost == "" {
		errs = append(errs, "GRPCHost is required")
	}
	if c.GRPCPort <= 0 {
		errs = append(errs, "GRPCPort must be a positive integer")
	}

	// Check migration path.
	if c.MigrationPath == "" {
		errs = append(errs, "MigrationPath is required")
	}

	// Check key paths.
	if c.RootKeyPath == "" {
		errs = append(errs, "RootKeyPath is required")
	}

	// Check authority ID and store name.
	if c.AuthorityID == "" {
		errs = append(errs, "AuthorityID is required")
	}
	if c.StoreName == "" {
		errs = append(errs, "StoreName is required")
	}
	if c.rootAccountIDString == "" {
		errs = append(errs, "RootAccountID is required")
	}
	if _, err := uuid.Parse(c.rootAccountIDString); err != nil {
		errs = append(errs, "RootAccountID must be a valid UUID")
	}

	if len(errs) > 0 {
		return fmt.Errorf("%s", strings.Join(errs, "; "))
	}
	return nil
}
