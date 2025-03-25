package config

import (
	"fmt"
	"os"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

type MacaroonConfig struct {
	RootKey            string `mapstructure:"root_key" yaml:"root_key"`
	RootId             string `mapstructure:"root_id" yaml:"root_id"`
	RootLocation       string `mapstructure:"root_location" yaml:"root_location"`
	DischargeKey       string `mapstructure:"discharge_key" yaml:"discharge_key"`
	ThirdPartyCaveatId string `mapstructure:"third_party_caveat_id" yaml:"third_party_caveat_id"`
	ThirdPartyLocation string `mapstructure:"third_party_location" yaml:"third_party_location"`
}

type Config struct {
	DebugMode          bool   `mapstructure:"debug_mode" yaml:"debug_mode"`
	AccountServiceHost string `mapstructure:"account_service_host" yaml:"account_service_host"`
	AccountServicePort int    `mapstructure:"account_service_port" yaml:"account_service_port"`

	StoreServiceHost string `mapstructure:"store_service_host" yaml:"store_service_host"`
	StoreServicePort int    `mapstructure:"store_service_port" yaml:"store_service_port"`

	AssertionServiceHost string `mapstructure:"assertion_service_host" yaml:"assertion_service_host"`
	AssertionServicePort int    `mapstructure:"assertion_service_port" yaml:"assertion_service_port"`

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
		return nil, fmt.Errorf("macaroon config is required")
	}

	logrus.Infof("loaded config: %+v, macaroonConfig: %+v", cfg, cfg.MacaroonConfig)

	if cfg.MacaroonConfig.DischargeKey == "" {
		return nil, fmt.Errorf("discharge key is required")
	}

	if cfg.MacaroonConfig.RootKey == "" {
		return nil, fmt.Errorf("root key is required")
	}

	return cfg, nil
}
