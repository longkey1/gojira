package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	BaseURL  string
	Email    string
	APIToken string
	ReadOnly bool
}

// configFile is set by the root command from the --config flag.
var configFile string

// SetConfigFile registers an explicit config file path to be used by Load.
// An empty string falls back to the default search paths.
func SetConfigFile(path string) {
	configFile = path
}

// keys are the recognized config keys, used by Load and Get.
var keys = []string{"jira_base_url", "jira_email", "jira_api_token", "read_only"}

// Keys returns the recognized config keys in a stable order.
func Keys() []string {
	return append([]string(nil), keys...)
}

// newViper builds a viper instance wired to the config file (or default search
// paths) and the GOJIRA_-prefixed environment variables (prefix + config key,
// e.g. GOJIRA_JIRA_EMAIL for jira_email).
func newViper() (*viper.Viper, error) {
	v := viper.New()
	v.SetConfigType("toml")

	if configFile != "" {
		v.SetConfigFile(configFile)
	} else {
		v.SetConfigName("config")
		for _, dir := range defaultConfigPaths() {
			v.AddConfigPath(dir)
		}
	}

	if err := v.ReadInConfig(); err != nil {
		var notFound viper.ConfigFileNotFoundError
		if !errors.As(err, &notFound) {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
		// When the user explicitly specified --config, a missing file is an error.
		if configFile != "" {
			return nil, fmt.Errorf("failed to read config file %q: %w", configFile, err)
		}
	}

	v.SetEnvPrefix("GOJIRA")
	v.AutomaticEnv()

	return v, nil
}

// Get returns the resolved value for a single config key without requiring the
// other keys to be set. Returns an error for an unknown key.
func Get(key string) (string, error) {
	if !slices.Contains(keys, key) {
		return "", fmt.Errorf("unknown config key %q (valid keys: %s)", key, strings.Join(keys, ", "))
	}

	v, err := newViper()
	if err != nil {
		return "", err
	}

	return os.ExpandEnv(v.GetString(key)), nil
}

func Load() (*Config, error) {
	v, err := newViper()
	if err != nil {
		return nil, err
	}

	email := os.ExpandEnv(v.GetString("jira_email"))
	if email == "" {
		return nil, errors.New("JIRA email is required (set GOJIRA_JIRA_EMAIL or 'jira_email' in config file)")
	}

	apiToken := os.ExpandEnv(v.GetString("jira_api_token"))
	if apiToken == "" {
		return nil, errors.New("JIRA API token is required (set GOJIRA_JIRA_API_TOKEN or 'jira_api_token' in config file)")
	}

	baseURL := os.ExpandEnv(v.GetString("jira_base_url"))
	if baseURL == "" {
		return nil, errors.New("JIRA base URL is required (set GOJIRA_JIRA_BASE_URL or 'jira_base_url' in config file)")
	}

	return &Config{
		BaseURL:  baseURL,
		Email:    email,
		APIToken: apiToken,
		ReadOnly: v.GetBool("read_only"),
	}, nil
}

// ReadOnly reports whether read-only mode is enabled (GOJIRA_READ_ONLY env var
// or read_only config key). Unlike Load, it does not require the connection
// settings to be present, so write commands can be rejected before any
// credential check.
func ReadOnly() (bool, error) {
	v, err := newViper()
	if err != nil {
		return false, err
	}
	return v.GetBool("read_only"), nil
}

// defaultConfigPaths returns the directories searched for config.toml when no
// explicit --config is given, in priority order.
func defaultConfigPaths() []string {
	var paths []string
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		paths = append(paths, filepath.Join(xdg, "gojira"))
	}
	if home, err := os.UserHomeDir(); err == nil {
		paths = append(paths, filepath.Join(home, ".config", "gojira"))
	}
	return paths
}
