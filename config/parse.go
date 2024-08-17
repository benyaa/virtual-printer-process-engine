package config

import (
	"gopkg.in/yaml.v3"
	"os"
)

func ParseConfig(location string) (Config, error) {
	data, err := os.ReadFile(location)
	if err != nil {
		return Config{}, err
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return Config{}, err
	}

	return config, nil
}
