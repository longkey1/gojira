package version

import (
	"fmt"
	"runtime"
	"testing"
)

// setVersionVars overrides the package-level build variables for a test and
// restores them on cleanup.
func setVersionVars(t *testing.T, version, commit, buildTime string) {
	t.Helper()
	origVersion, origCommit, origBuildTime := Version, CommitSHA, BuildTime
	t.Cleanup(func() {
		Version, CommitSHA, BuildTime = origVersion, origCommit, origBuildTime
	})
	Version, CommitSHA, BuildTime = version, commit, buildTime
}

func TestInfo(t *testing.T) {
	tests := []struct {
		name      string
		version   string
		commit    string
		buildTime string
	}{
		{
			name:      "default dev values",
			version:   "dev",
			commit:    "unknown",
			buildTime: "unknown",
		},
		{
			name:      "release values injected via ldflags",
			version:   "v1.2.3",
			commit:    "abc1234",
			buildTime: "2026-07-13T00:00:00Z",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setVersionVars(t, tt.version, tt.commit, tt.buildTime)

			want := fmt.Sprintf("Version: %s\nCommit: %s\nBuild Time: %s\nGo Version: %s",
				tt.version, tt.commit, tt.buildTime, runtime.Version())
			if got := Info(); got != want {
				t.Errorf("Info() = %q, want %q", got, want)
			}
		})
	}
}

func TestShort(t *testing.T) {
	tests := []struct {
		name    string
		version string
	}{
		{name: "dev version", version: "dev"},
		{name: "release version", version: "v1.2.3"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setVersionVars(t, tt.version, "unknown", "unknown")

			if got := Short(); got != tt.version {
				t.Errorf("Short() = %q, want %q", got, tt.version)
			}
		})
	}
}
