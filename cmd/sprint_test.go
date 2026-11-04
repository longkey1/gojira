package cmd

import (
	"slices"
	"testing"
)

func TestParseSprintStates(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    []string
		wantErr bool
	}{
		{name: "empty means all states", input: "", want: nil},
		{name: "whitespace only", input: "  ", want: nil},
		{name: "single state", input: "active", want: []string{"active"}},
		{name: "multiple states with spaces", input: "active, closed ,future", want: []string{"active", "closed", "future"}},
		{name: "invalid state", input: "active,open", wantErr: true},
		{name: "trailing comma", input: "active,", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := parseSprintStates(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("parseSprintStates(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
			if !slices.Equal(got, tt.want) {
				t.Errorf("parseSprintStates(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestResolveBoardID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		flagSet      bool
		flagValue    int
		defaultBoard int
		want         int
		wantErr      bool
	}{
		{name: "flag wins over default", flagSet: true, flagValue: 1, defaultBoard: 7417, want: 1},
		{name: "default used when flag omitted", defaultBoard: 7417, want: 7417},
		{name: "neither set", wantErr: true},
		{name: "explicit zero flag", flagSet: true, flagValue: 0, defaultBoard: 7417, wantErr: true},
		{name: "negative flag", flagSet: true, flagValue: -1, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := resolveBoardID(tt.flagSet, tt.flagValue, tt.defaultBoard)
			if (err != nil) != tt.wantErr {
				t.Fatalf("resolveBoardID() error = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("resolveBoardID() = %d, want %d", got, tt.want)
			}
		})
	}
}
