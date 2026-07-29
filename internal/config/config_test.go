package config

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// setupEnv isolates the test from the real environment: HOME points to an
// empty temp dir, XDG_CONFIG_HOME and the GOJIRA_* variables are cleared, and
// the package-level configFile is reset before and after the test.
func setupEnv(t *testing.T) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("GOJIRA_JIRA_EMAIL", "")
	t.Setenv("GOJIRA_JIRA_API_TOKEN", "")
	t.Setenv("GOJIRA_JIRA_BASE_URL", "")
	t.Setenv("GOJIRA_READ_ONLY", "")
	SetConfigFile("")
	t.Cleanup(func() { SetConfigFile("") })
}

// writeConfig writes content to a config.toml in a fresh temp dir and returns
// the file path.
func writeConfig(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}
	return path
}

func TestLoad(t *testing.T) {
	tests := []struct {
		name       string
		configTOML string // written to a temp config file when non-empty
		env        map[string]string
		want       Config
		wantErr    string // substring of the expected error, empty for success
	}{
		{
			name: "all values from config file",
			configTOML: `jira_base_url = "https://example.atlassian.net"
jira_email = "user@example.com"
jira_api_token = "token123"
`,
			want: Config{
				BaseURL:  "https://example.atlassian.net",
				Email:    "user@example.com",
				APIToken: "token123",
			},
		},
		{
			name: "all values from environment",
			env: map[string]string{
				"GOJIRA_JIRA_BASE_URL":  "https://env.atlassian.net",
				"GOJIRA_JIRA_EMAIL":     "env@example.com",
				"GOJIRA_JIRA_API_TOKEN": "envtoken",
			},
			want: Config{
				BaseURL:  "https://env.atlassian.net",
				Email:    "env@example.com",
				APIToken: "envtoken",
			},
		},
		{
			name: "environment overrides config file",
			configTOML: `jira_base_url = "https://file.atlassian.net"
jira_email = "file@example.com"
jira_api_token = "filetoken"
`,
			env: map[string]string{
				"GOJIRA_JIRA_EMAIL": "env@example.com",
			},
			want: Config{
				BaseURL:  "https://file.atlassian.net",
				Email:    "env@example.com",
				APIToken: "filetoken",
			},
		},
		{
			name: "values are env-expanded",
			configTOML: `jira_base_url = "https://example.atlassian.net"
jira_email = "user@example.com"
jira_api_token = "${TEST_GOJIRA_TOKEN}"
`,
			env: map[string]string{
				"TEST_GOJIRA_TOKEN": "expanded-secret",
			},
			want: Config{
				BaseURL:  "https://example.atlassian.net",
				Email:    "user@example.com",
				APIToken: "expanded-secret",
			},
		},
		{
			name: "read_only from config file",
			configTOML: `jira_base_url = "https://example.atlassian.net"
jira_email = "user@example.com"
jira_api_token = "token123"
read_only = true
`,
			want: Config{
				BaseURL:  "https://example.atlassian.net",
				Email:    "user@example.com",
				APIToken: "token123",
				ReadOnly: true,
			},
		},
		{
			name: "read_only from environment",
			env: map[string]string{
				"GOJIRA_JIRA_BASE_URL":  "https://env.atlassian.net",
				"GOJIRA_JIRA_EMAIL":     "env@example.com",
				"GOJIRA_JIRA_API_TOKEN": "envtoken",
				"GOJIRA_READ_ONLY": "true",
			},
			want: Config{
				BaseURL:  "https://env.atlassian.net",
				Email:    "env@example.com",
				APIToken: "envtoken",
				ReadOnly: true,
			},
		},
		{
			name: "missing email",
			configTOML: `jira_base_url = "https://example.atlassian.net"
jira_api_token = "token123"
`,
			wantErr: "email is required",
		},
		{
			name: "missing api token",
			configTOML: `jira_base_url = "https://example.atlassian.net"
jira_email = "user@example.com"
`,
			wantErr: "API token is required",
		},
		{
			name: "missing base URL",
			configTOML: `jira_email = "user@example.com"
jira_api_token = "token123"
`,
			wantErr: "base URL is required",
		},
		{
			name:    "no config file and no environment",
			wantErr: "email is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupEnv(t)
			for k, v := range tt.env {
				t.Setenv(k, v)
			}
			if tt.configTOML != "" {
				SetConfigFile(writeConfig(t, tt.configTOML))
			}

			got, err := Load()
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("Load() error = nil, want error containing %q", tt.wantErr)
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("Load() error = %q, want error containing %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Load() unexpected error: %v", err)
			}
			if *got != tt.want {
				t.Errorf("Load() = %+v, want %+v", *got, tt.want)
			}
		})
	}
}

func TestLoadExplicitConfigFileErrors(t *testing.T) {
	tests := []struct {
		name  string
		setup func(t *testing.T) string // returns the path passed to SetConfigFile
	}{
		{
			name: "explicit config file does not exist",
			setup: func(t *testing.T) string {
				return filepath.Join(t.TempDir(), "missing.toml")
			},
		},
		{
			name: "invalid TOML in config file",
			setup: func(t *testing.T) string {
				return writeConfig(t, "this is not = [valid toml")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupEnv(t)
			SetConfigFile(tt.setup(t))

			if _, err := Load(); err == nil {
				t.Fatal("Load() error = nil, want error")
			}
			if _, err := Get("jira_email"); err == nil {
				t.Fatal("Get() error = nil, want error")
			}
		})
	}
}

func TestLoadFromDefaultXDGPath(t *testing.T) {
	setupEnv(t)

	xdg := t.TempDir()
	dir := filepath.Join(xdg, "gojira")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("failed to create config dir: %v", err)
	}
	content := `jira_base_url = "https://xdg.atlassian.net"
jira_email = "xdg@example.com"
jira_api_token = "xdgtoken"
`
	if err := os.WriteFile(filepath.Join(dir, "config.toml"), []byte(content), 0o600); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}
	t.Setenv("XDG_CONFIG_HOME", xdg)

	got, err := Load()
	if err != nil {
		t.Fatalf("Load() unexpected error: %v", err)
	}
	want := Config{
		BaseURL:  "https://xdg.atlassian.net",
		Email:    "xdg@example.com",
		APIToken: "xdgtoken",
	}
	if *got != want {
		t.Errorf("Load() = %+v, want %+v", *got, want)
	}
}

func TestGet(t *testing.T) {
	tests := []struct {
		name       string
		key        string
		configTOML string
		env        map[string]string
		want       string
		wantErr    string
	}{
		{
			name:       "known key from config file",
			key:        "jira_base_url",
			configTOML: `jira_base_url = "https://example.atlassian.net"`,
			want:       "https://example.atlassian.net",
		},
		{
			name: "known key from environment",
			key:  "jira_email",
			env:  map[string]string{"GOJIRA_JIRA_EMAIL": "env@example.com"},
			want: "env@example.com",
		},
		{
			name:       "value is env-expanded",
			key:        "jira_api_token",
			configTOML: `jira_api_token = "$TEST_GOJIRA_TOKEN"`,
			env:        map[string]string{"TEST_GOJIRA_TOKEN": "secret"},
			want:       "secret",
		},
		{
			name: "unset key returns empty string without error",
			key:  "jira_api_token",
			want: "",
		},
		{
			name:    "unknown key",
			key:     "nonexistent",
			wantErr: `unknown config key "nonexistent"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupEnv(t)
			for k, v := range tt.env {
				t.Setenv(k, v)
			}
			if tt.configTOML != "" {
				SetConfigFile(writeConfig(t, tt.configTOML))
			}

			got, err := Get(tt.key)
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("Get(%q) error = nil, want error containing %q", tt.key, tt.wantErr)
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("Get(%q) error = %q, want error containing %q", tt.key, err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Get(%q) unexpected error: %v", tt.key, err)
			}
			if got != tt.want {
				t.Errorf("Get(%q) = %q, want %q", tt.key, got, tt.want)
			}
		})
	}
}

func TestReadOnly(t *testing.T) {
	tests := []struct {
		name       string
		configTOML string
		env        map[string]string
		want       bool
	}{
		{
			name: "unset defaults to false",
			want: false,
		},
		{
			name:       "enabled via config file",
			configTOML: `read_only = true`,
			want:       true,
		},
		{
			name: "enabled via environment",
			env:  map[string]string{"GOJIRA_READ_ONLY": "true"},
			want: true,
		},
		{
			name:       "environment overrides config file",
			configTOML: `read_only = true`,
			env:        map[string]string{"GOJIRA_READ_ONLY": "false"},
			want:       false,
		},
		{
			name: "works without connection settings",
			env:  map[string]string{"GOJIRA_READ_ONLY": "1"},
			want: true,
		},
		{
			name: "legacy JIRA_READ_ONLY is ignored",
			env:  map[string]string{"JIRA_READ_ONLY": "true"},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupEnv(t)
			for k, v := range tt.env {
				t.Setenv(k, v)
			}
			if tt.configTOML != "" {
				SetConfigFile(writeConfig(t, tt.configTOML))
			}

			got, err := ReadOnly()
			if err != nil {
				t.Fatalf("ReadOnly() unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("ReadOnly() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestKeys(t *testing.T) {
	want := []string{"jira_base_url", "jira_email", "jira_api_token", "read_only"}

	got := Keys()
	if !slices.Equal(got, want) {
		t.Fatalf("Keys() = %v, want %v", got, want)
	}

	// Mutating the returned slice must not affect subsequent calls.
	got[0] = "mutated"
	if again := Keys(); !slices.Equal(again, want) {
		t.Errorf("Keys() after mutation = %v, want %v", again, want)
	}
}
