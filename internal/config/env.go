package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type EnvConfig struct {
	EnvFile string
}

func (cfg EnvConfig) GetConfigValue(key string) (string, bool) {
	return os.LookupEnv(key)
}

func MakeEnvConfig(envFile string) (cfg EnvConfig, err error) {
	err = godotenv.Load(envFile)

	if err != nil {
		return cfg, fmt.Errorf("failed to load env file %s - %w", envFile, err)
	}

	cfg.EnvFile = envFile

	return
}
