package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

type Config struct {
	DBHost         string `mapstructure:"db_host" yaml:"db_host"`
	DBPort         int    `mapstructure:"db_port" yaml:"db_port"`
	DBUser         string `mapstructure:"db_user" yaml:"db_user"`
	DBPassword     string `mapstructure:"db_password" yaml:"db_password"`
	DBName         string `mapstructure:"db_name" yaml:"db_name"`
	GRPCHost       string `mapstructure:"grpc_host" yaml:"grpc_host"`
	GRPCPort       int    `mapstructure:"grpc_port" yaml:"grpc_port"`
	MigrationPath  string `mapstructure:"migration_path" yaml:"migration_path"`
	RootKeyPath    string `mapstructure:"root_key_path" yaml:"root_key_path"`
	GenericKeyPath string `mapstructure:"generic_key_path" yaml:"generic_key_path"`
	MinioAccessKey string `mapstructure:"minio_access_key" yaml:"minio_access_key"`
	MinioSecretKey string `mapstructure:"minio_secret_key" yaml:"minio_secret"`
	MinioHost      string `mapstructure:"minio_host" yaml:"minio_host"`
	MinioSecure    bool   `mapstructure:"minio_secure" yaml:"minio_secure"`

	Monitoring bool `mapstructure:"monitoring" yaml:"monitoring"`

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

	logrus.Infof("loaded config: %+v", cfg)

	return cfg, nil
}

func GetStoreServiceAddress(host string, port int) string {
	if host == "" || port == 0 {
		logrus.Warn("host or port is not set, using default address store_service:8081")
		return "store_service:8081"
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
	if c.GenericKeyPath == "" {
		errs = append(errs, "GenericKeyPath is required")
	}

	// Check Minio settings.
	if c.MinioAccessKey == "" {
		errs = append(errs, "MinioAccessKey is required")
	}
	if c.MinioSecretKey == "" {
		errs = append(errs, "MinioSecretKey is required")
	}
	if c.MinioHost == "" {
		errs = append(errs, "MinioHost is required")
	}

	if len(errs) > 0 {
		return fmt.Errorf("%s", strings.Join(errs, "; "))
	}
	return nil
}
