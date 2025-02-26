package config

import (
    "fmt"
    "os"
    
    "github.com/spf13/viper"
    "github.com/sirupsen/logrus"
)

var (
    config *Config
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
    config = cfg
    if err := viper.Unmarshal(cfg); err != nil {
        return nil, fmt.Errorf("failed to unmarshal config: %v", err)
    }
    
    return cfg, nil
}

func GetAccountServiceAddress() string {
    if config == nil {
        logrus.Warn("config is nil, using default address")
        return "account_service:8080"
    }

    return fmt.Sprintf("%s:%d", config.GRPCHost, config.GRPCPort)
}


