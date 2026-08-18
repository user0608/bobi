package configs

import (
	"errors"
	"strings"
)

type ConfigPath string

func ConfigPathFromArgs(args []string) (ConfigPath, error) {
	for i, rawArg := range args {
		arg := strings.TrimSpace(rawArg)

		if arg == "--config" || arg == "-c" {
			if i+1 >= len(args) {
				return "", errors.New("missing value for config flag")
			}

			path := strings.TrimSpace(args[i+1])
			if path == "" {
				return "", errors.New("empty config path")
			}

			return ConfigPath(path), nil
		}

		if value, ok := strings.CutPrefix(arg, "--config="); ok {
			path := strings.TrimSpace(value)
			if path == "" {
				return "", errors.New("empty config path")
			}

			return ConfigPath(path), nil
		}

		if value, ok := strings.CutPrefix(arg, "-c="); ok {
			path := strings.TrimSpace(value)
			if path == "" {
				return "", errors.New("empty config path")
			}

			return ConfigPath(path), nil
		}
	}

	return "", errors.New("--config flag not provided")
}
