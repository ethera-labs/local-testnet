package configs

import (
	_ "embed"
	"fmt"
	"strings"
	"sync"

	"github.com/spf13/viper"
)

var (
	//go:embed config.example.yaml
	defaultConfigYAML string

	defaultConfigOnce sync.Once
	defaultConfig     Config
	defaultConfigErr  error
)

// DefaultConfig returns the parsed configuration from the embedded config.example.yaml.
func DefaultConfig() (Config, error) {
	defaultConfigOnce.Do(func() {
		v := viper.New()
		v.SetConfigType("yaml")
		if err := v.ReadConfig(strings.NewReader(defaultConfigYAML)); err != nil {
			defaultConfigErr = fmt.Errorf("failed to read embedded config.example.yaml: %w", err)
			return
		}

		if err := v.Unmarshal(&defaultConfig); err != nil {
			defaultConfigErr = fmt.Errorf("failed to decode embedded config.example.yaml: %w", err)
			return
		}
	})

	if defaultConfigErr != nil {
		return Config{}, defaultConfigErr
	}

	return defaultConfig, nil
}

// MustDefaultConfig returns embedded defaults or panics if they cannot be loaded.
func MustDefaultConfig() Config {
	cfg, err := DefaultConfig()
	if err != nil {
		panic(err)
	}
	return cfg
}
