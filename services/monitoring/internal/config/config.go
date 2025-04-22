package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

type Config struct {
	StoreUrl string `mapstructure:"store_url" yaml:"store_url"`
	Macaroon string `mapstructure:"macaroon" yaml:"macaroon"`
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

	if err := cfg.checkConfig(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %v", err)
	}

	logrus.Infof("loaded config: %+v", cfg)

	return cfg, nil
}

func (c *Config) checkConfig() error {
	var errs []string

	// Check DB config.
	if c.StoreUrl == "" {
		errs = append(errs, "store_url is required")
	}

	if c.Macaroon == "" {
		errs = append(errs, "macaroon is required")
	}

	if len(errs) > 0 {
		return fmt.Errorf("%s", strings.Join(errs, "; "))
	}
	return nil
}
