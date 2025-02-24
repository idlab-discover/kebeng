package config

import (
    "fmt"
    
    "github.com/spf13/viper"
)

type Config struct {
    DBHost string
    DBPort int
    DBUser int
    DBPassword string
    DBName string
}

// path: path in the container where the config file is
func LoadConfig(path string) (*Config, error) {
    viper.SetConfigName("config")
    viper.SetConfigType("yaml")
    viper.AddConfigPath(path)
    
    if err := viper.ReadInConfig(); err != nil {
        return nil, fmt.Errorf("failed to read config file: %v", err)
    }

    var config Config
    if err := viper.Unmarshal(&config); err != nil {
        return nil, fmt.Errorf("failed to unmarshal config: %v", err)
    }

    return &config, nil
}


