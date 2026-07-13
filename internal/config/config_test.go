package config

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// setupEnv isolates the test from the real environment: HOME points to an
// empty temp dir, XDG_CONFIG_HOME and the JIRA_* variables are cleared, and
// the package-level configFile is reset before and after the test.
func setupEnv(t *testing.T) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("JIRA_EMAIL", "")
	t.Setenv("JIRA_API_TOKEN", "")
	t.Setenv("JIRA_BASE_URL", "")
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
			configTOML: `base_url = "https://example.atlassian.net"
email = "user@example.com"
api_token = "token123"
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
				"JIRA_BASE_URL":  "https://env.atlassian.net",
				"JIRA_EMAIL":     "env@example.com",
				"JIRA_API_TOKEN": "envtoken",
			},
			want: Config{
				BaseURL:  "https://env.atlassian.net",
				Email:    "env@example.com",
				APIToken: "envtoken",
			},
		},
		{
			name: "environment overrides config file",
			configTOML: `base_url = "https://file.atlassian.net"
email = "file@example.com"
api_token = "filetoken"
`,
			env: map[string]string{
				"JIRA_EMAIL": "env@example.com",
			},
			want: Config{
				BaseURL:  "https://file.atlassian.net",
				Email:    "env@example.com",
				APIToken: "filetoken",
			},
		},
		{
			name: "values are env-expanded",
			configTOML: `base_url = "https://example.atlassian.net"
email = "user@example.com"
api_token = "${TEST_GOJIRA_TOKEN}"
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
			name: "missing email",
			configTOML: `base_url = "https://example.atlassian.net"
api_token = "token123"
`,
			wantErr: "email is required",
		},
		{
			name: "missing api token",
			configTOML: `base_url = "https://example.atlassian.net"
email = "user@example.com"
`,
			wantErr: "API token is required",
		},
		{
			name: "missing base URL",
			configTOML: `email = "user@example.com"
api_token = "token123"
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
			if _, err := Get("email"); err == nil {
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
	content := `base_url = "https://xdg.atlassian.net"
email = "xdg@example.com"
api_token = "xdgtoken"
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
			key:        "base_url",
			configTOML: `base_url = "https://example.atlassian.net"`,
			want:       "https://example.atlassian.net",
		},
		{
			name: "known key from environment",
			key:  "email",
			env:  map[string]string{"JIRA_EMAIL": "env@example.com"},
			want: "env@example.com",
		},
		{
			name:       "value is env-expanded",
			key:        "api_token",
			configTOML: `api_token = "$TEST_GOJIRA_TOKEN"`,
			env:        map[string]string{"TEST_GOJIRA_TOKEN": "secret"},
			want:       "secret",
		},
		{
			name: "unset key returns empty string without error",
			key:  "api_token",
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

func TestKeys(t *testing.T) {
	want := []string{"base_url", "email", "api_token"}

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
