// Package config handles configuration loading via Viper.
package config

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/viper"
)

// ScanConfig holds defaults for the scan command.
type ScanConfig struct {
	DefaultTimeout string `mapstructure:"default_timeout"`
}

// Config defines the shape of ~/.netdiag.yaml.
//
// Keys are added here only once a command reads them; an inert key in the
// config file is worse than no key, because it silently does nothing.
type Config struct {
	Scan ScanConfig `mapstructure:"scan"`
}

// AppConfig is the loaded configuration. Populated by Load.
var AppConfig Config

// Load reads ~/.netdiag.yaml and falls back to defaults if missing.
func Load() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	viper.AddConfigPath(home)
	viper.SetConfigName(".netdiag")
	viper.SetConfigType("yaml")

	viper.SetDefault("scan.default_timeout", "1s")

	viper.SetEnvPrefix("NETDIAG")
	// Nested keys are dotted, environment variables are not: without this,
	// NETDIAG_SCAN_DEFAULT_TIMEOUT never reaches scan.default_timeout.
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		var notFound viper.ConfigFileNotFoundError
		if !errors.As(err, &notFound) {
			return fmt.Errorf("failed to read config file: %w", err)
		}
	}

	if err := viper.Unmarshal(&AppConfig); err != nil {
		return fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return nil
}
