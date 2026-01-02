package config

import (
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

type Config struct {
	BaseURL     string  `yaml:"base_url"`
	Token       string  `yaml:"token"`
	ModelName   string  `yaml:"model_name"`
	Temperature float64 `yaml:"temperature"`
}

func LoadConfig() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	if err := viper.ReadInConfig(); err != nil {
		log.Error().Err(err).Msg("Error reading config file")
		return nil, errors.New("Error reading config file: " + err.Error())
	}
	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		log.Error().Err(err).Msg("Error unmarshalling config file")
		return nil, errors.New("Error unmarshalling config file: " + err.Error())
	}
	return &config, nil
}
