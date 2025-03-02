package config

import (
    "os"
    "fmt"

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
    DebugMode bool `mapstructure:"debug_mode" yaml:"debug_mode"`
    AccountServiceHost string `mapstructure:"account_service_host" yaml:"account_service_host"`
    AccountServicePort int    `mapstructure:"account_service_port" yaml:"account_service_port"`
    
    StoreServiceHost string `mapstructure:"store_service_host" yaml:"store_service_host"`
    StoreServicePort int    `mapstructure:"store_service_port" yaml:"store_service_port"`
    
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

    return cfg, nil
}
