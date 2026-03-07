// Package config handles configuration loading via Viper.
package config

import (
	"fmt"
	"os"

	"github.com/spf13/viper"
)

type MonitorConfig struct {
	Interval string `mapstructure:"interval"`
}

type DatabaseConfig struct {
	Path string `mapstructure:"path"`
}

type MetricsConfig struct {
	Enabled bool `mapstructure:"enabled"`
}

type ScanConfig struct {
	DefaultTimeout string `mapstructure:"default_timeout"`
}

// Config defines the shape of the YAML file
type Config struct {
	Monitor  MonitorConfig  `mapstructure:"monitor"`
	Database DatabaseConfig `mapstructure:"database"`
	Metrics  MetricsConfig  `mapstructure:"metrics"`
	Scan     ScanConfig     `mapstructure:"scan"`
}

var AppConfig Config

// Load reads ~/.netdiag.yaml and falls back to defaults if missing
func Load() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	viper.AddConfigPath(home)
	viper.SetConfigName(".netdiag")
	viper.SetConfigType("yaml")

	// Set defaults
	viper.SetDefault("monitor.interval", "5m")
	viper.SetDefault("database.path", "~/.netdiag.db")
	viper.SetDefault("metrics.enabled", false)
	viper.SetDefault("scan.default_timeout", "1s")

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return fmt.Errorf("failed to read config file: %w", err)
		}
	}

	if err := viper.Unmarshal(&AppConfig); err != nil {
		return fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return nil
}
