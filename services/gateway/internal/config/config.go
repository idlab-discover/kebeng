package config

import (
	"fmt"
	"os"
	"strings"

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

	StoreUrl string `mapstructure:"store_url" yaml:"store_url"`

	TestMode           bool   `mapstructure:"test_mode" yaml:"test_mode"`
	TestDataFolderPath string `mapstructure:"test_data_folder_path" yaml:"test_data_folder_path"`
	StoreIP string `mapstructure:"store_ip" yaml:"store_ip"`
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
		return nil, fmt.Errorf("config validation failed: %v", err)
	}

	return cfg, nil
}

func (c *Config) checkConfig() error {
	var errs []string

	// Account Service config
	if c.AccountServiceHost == "" {
		errs = append(errs, "account_service_host is required")
	}
	if c.AccountServicePort <= 0 {
		errs = append(errs, "account_service_port must be a positive integer")
	}

	// Store Service config
	if c.StoreServiceHost == "" {
		errs = append(errs, "store_service_host is required")
	}
	if c.StoreServicePort <= 0 {
		errs = append(errs, "store_service_port must be a positive integer")
	}

	// Assertion Service config
	if c.AssertionServiceHost == "" {
		errs = append(errs, "assertion_service_host is required")
	}
	if c.AssertionServicePort <= 0 {
		errs = append(errs, "assertion_service_port must be a positive integer")
	}

	// Macaroon config
	if c.MacaroonConfig == nil {
		errs = append(errs, "macaroon config is required")
	} else {
		if c.MacaroonConfig.RootKey == "" {
			errs = append(errs, "macaroon.root_key is required")
		}
		if c.MacaroonConfig.RootId == "" {
			errs = append(errs, "macaroon.root_id is required")
		}
		if c.MacaroonConfig.RootLocation == "" {
			errs = append(errs, "macaroon.root_location is required")
		}
		if c.MacaroonConfig.DischargeKey == "" {
			errs = append(errs, "macaroon.discharge_key is required")
		}
		if c.MacaroonConfig.ThirdPartyCaveatId == "" {
			errs = append(errs, "macaroon.third_party_caveat_id is required")
		}
		if c.MacaroonConfig.ThirdPartyLocation == "" {
			errs = append(errs, "macaroon.third_party_location is required")
		}
	}

	// Store URL config
	if c.StoreUrl == "" {
		errs = append(errs, "store_url is required")
	}

	// Test mode settings
	if c.TestMode {
		if c.TestDataFolderPath == "" {
			errs = append(errs, "test_data_folder_path is required in test mode")
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("%s", strings.Join(errs, "; "))
	}
	return nil
}
