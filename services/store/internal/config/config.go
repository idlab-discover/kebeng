package config

import (
	"fmt"
	"os"

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

	TestMode          bool
	TestDataFilePath  string `mapstructure:"test_data_file_path" yaml:"test_data_file_path"`
	TestDataMinioPath string `mapstructure:"test_data_minio_path" yaml:"test_data_minio_path"`
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
