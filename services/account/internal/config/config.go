package config

import (
    "fmt"
    "os"

    "github.com/spf13/viper"
    "github.com/sirupsen/logrus"
    "github.com/google/uuid"
)

type MacaroonConfig struct {
    RootKey string `mapstructure:"macaroon_root_key" yaml:"macaroon_root_key"`
    RootId uuid.UUID `mapstructure:"macaroon_root_id" yaml:"macaroon_root_id"`
    RootLocation string `mapstructure:"macaroon_root_location" yaml:"macaroon_root_location"`
    
    // MacaroonDischargeKey is a key used for third-party caveats in the macaroon (secret shared key between service and third party)
    DischargeKey string `mapstructure:"macaroon_discharge_key" yaml:"macaroon_discharge_key"`
    ThirdPartyCaveatId uuid.UUID `mapstructure:"macaroon_third_party_caveat_id" yaml:"macaroon_third_party_caveat_id"`
    ThirdPartyLocation string `mapstructure:"macaroon_third_party_location" yaml:"macaroon_third_party_location"`
}

type Config struct {
    DBHost         string `mapstructure:"db_host" yaml:"db_host"`
    DBPort         int    `mapstructure:"db_port" yaml:"db_port"`
    DBUser         string `mapstructure:"db_user" yaml:"db_user"`
    DBPassword     string `mapstructure:"db_password" yaml:"db_password"`
    DBName         string `mapstructure:"db_name" yaml:"db_name"`
    GRPCHost       string `mapstructure:"grpc_host" yaml:"grpc_host"`
    GRPCPort       int    `mapstructure:"grpc_port" yaml:"grpc_port"`
    MigrationPath  string `mapstructure:"migration_path" yaml:"migration_path"`
    MacaroonConfig *MacaroonConfig `mapstructure:"macaroon" yaml:"macaroon"`
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

    if cfg.MacaroonConfig == nil {
        return nil, fmt.Errorf("macaroon config is not set")
    }

    if len(cfg.MacaroonConfig.DischargeKey) == 0 {
        return nil, fmt.Errorf("macaroon discharge key is not set")
    }
    
    return cfg, nil
}

func GetAccountServiceAddress(host string, port int) string {
    if host == "" || port == 0 {
        logrus.Warn("host or port is not set, using default address account_service:8080")
        return "account_service:8080"
    }

    return fmt.Sprintf("%s:%d", host, port)
}


