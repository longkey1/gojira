package cmd

import (
	"strings"
	"testing"

	"github.com/longkey1/gojira/internal/config"
	"github.com/spf13/cobra"
)

func TestIsWriteCommand(t *testing.T) {
	t.Parallel()

	writeCommands := map[string]bool{"create": true, "update": true, "add": true, "delete": true}

	tests := []struct {
		name string
		path []string // command chain from root to leaf
		want bool
	}{
		{name: "create command", path: []string{"create"}, want: true},
		{name: "update command", path: []string{"update"}, want: true},
		{name: "comment add subcommand", path: []string{"comment", "add"}, want: true},
		{name: "comment delete subcommand", path: []string{"comment", "delete"}, want: true},
		{name: "comment list subcommand", path: []string{"comment", "list"}, want: false},
		{name: "get command", path: []string{"get"}, want: false},
		{name: "list command", path: []string{"list"}, want: false},
		{name: "root only", path: nil, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			root := &cobra.Command{Use: "gojira"}
			leaf := root
			for _, name := range tt.path {
				child := &cobra.Command{Use: name}
				if writeCommands[name] {
					child.Annotations = map[string]string{writeAnnotation: "true"}
				}
				leaf.AddCommand(child)
				leaf = child
			}

			if got := isWriteCommand(leaf); got != tt.want {
				t.Errorf("isWriteCommand(%v) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

// TestWriteCommandAnnotations verifies that the actual registered commands
// carry the write annotation exactly where expected.
func TestWriteCommandAnnotations(t *testing.T) {
	tests := []struct {
		path []string // command chain from root
		want bool
	}{
		{path: []string{"create"}, want: true},
		{path: []string{"update"}, want: true},
		{path: []string{"comment", "add"}, want: true},
		{path: []string{"comment", "update"}, want: true},
		{path: []string{"comment", "delete"}, want: true},
		{path: []string{"comment", "list"}, want: false},
		{path: []string{"get"}, want: false},
		{path: []string{"list"}, want: false},
		{path: []string{"fields"}, want: false},
		{path: []string{"merge"}, want: false},
		{path: []string{"config"}, want: false},
		{path: []string{"version"}, want: false},
	}

	for _, tt := range tests {
		t.Run(strings.Join(tt.path, " "), func(t *testing.T) {
			cmd := rootCmd
			for _, name := range tt.path {
				found := false
				for _, c := range cmd.Commands() {
					if c.Name() == name {
						cmd = c
						found = true
						break
					}
				}
				if !found {
					t.Fatalf("command %q not found under %q", name, cmd.Name())
				}
			}

			if got := isWriteCommand(cmd); got != tt.want {
				t.Errorf("isWriteCommand(%v) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestCheckReadOnly(t *testing.T) {
	tests := []struct {
		name     string
		readOnly string // value for GOJIRA_READ_ONLY, empty to leave unset
		wantErr  bool
	}{
		{name: "read-only disabled", readOnly: "", wantErr: false},
		{name: "read-only enabled", readOnly: "true", wantErr: true},
		{name: "read-only explicitly false", readOnly: "false", wantErr: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Isolate from the real environment and any user config file.
			t.Setenv("HOME", t.TempDir())
			t.Setenv("XDG_CONFIG_HOME", "")
			t.Setenv("GOJIRA_READ_ONLY", tt.readOnly)
			config.SetConfigFile("")
			t.Cleanup(func() { config.SetConfigFile("") })

			err := checkReadOnly()
			if tt.wantErr {
				if err == nil {
					t.Fatal("checkReadOnly() error = nil, want error")
				}
				if !strings.Contains(err.Error(), "read-only mode is enabled") {
					t.Fatalf("checkReadOnly() error = %q, want read-only error", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("checkReadOnly() unexpected error: %v", err)
			}
		})
	}
}
