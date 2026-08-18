package configs

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/viper"
)

func LoadConfigFromCLIArgs() (*viper.Viper, error) {
	configPath, err := ConfigPathFromArgs(os.Args[1:])
	if err != nil {
		return nil, err
	}

	return LoadConfig(configPath)
}

func LoadConfig(configPath ConfigPath) (*viper.Viper, error) {
	filePath := strings.TrimSpace(string(configPath))
	if filePath == "" {
		return nil, errors.New("config path is empty")
	}

	v := viper.New()
	v.SetConfigFile(filePath)

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("read config %q: %w", filePath, err)
	}

	return v, nil
}
