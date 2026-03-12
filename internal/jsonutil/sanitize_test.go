package jsonutil

import (
	"testing"
)

func TestSanitizeJSON(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "valid JSON unchanged",
			input: `{"key": "value"}`,
			want:  `{"key": "value"}`,
		},
		{
			name:  "properly escaped sequences unchanged",
			input: `{"text": "line1\\nline2"}`,
			want:  `{"text": "line1\\nline2"}`,
		},
		{
			name:  "raw newline in string",
			input: "{\"text\": \"line1\nline2\"}",
			want:  `{"text": "line1\nline2"}`,
		},
		{
			name:  "raw carriage return and newline in string",
			input: "{\"text\": \"line1\r\nline2\"}",
			want:  `{"text": "line1\r\nline2"}`,
		},
		{
			name:  "raw tab in string",
			input: "{\"text\": \"col1\tcol2\"}",
			want:  `{"text": "col1\tcol2"}`,
		},
		{
			name:  "invalid escape sequence backslash-T",
			input: `{"text": "Mercari\Transaction\Api"}`,
			want:  `{"text": "Mercari\\Transaction\\Api"}`,
		},
		{
			name:  "invalid escape sequence backslash-S",
			input: `{"sql": "SELECT\nFROM"}`,
			want:  `{"sql": "SELECT\nFROM"}`,
		},
		{
			name:  "mixed valid and invalid escapes",
			input: `{"text": "a\nb\Tc\rd"}`,
			want:  `{"text": "a\nb\\Tc\rd"}`,
		},
		{
			name:  "null byte in string",
			input: "{\"text\": \"a\x00b\"}",
			want:  `{"text": "a\u0000b"}`,
		},
		{
			name:  "control characters outside strings unchanged",
			input: "{\n  \"key\": \"value\"\n}",
			want:  "{\n  \"key\": \"value\"\n}",
		},
		{
			name:  "escaped quote in string",
			input: `{"text": "say \"hello\""}`,
			want:  `{"text": "say \"hello\""}`,
		},
		{
			name:  "escaped backslash followed by n",
			input: `{"text": "path\\name"}`,
			want:  `{"text": "path\\name"}`,
		},
		{
			name:  "unicode escape unchanged",
			input: `{"text": "\u0041"}`,
			want:  `{"text": "\u0041"}`,
		},
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
		{
			name:  "nested objects with raw control chars",
			input: "{\"outer\": {\"inner\": \"has\nnewline\"}}",
			want:  `{"outer": {"inner": "has\nnewline"}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := string(SanitizeJSON([]byte(tt.input)))
			if got != tt.want {
				t.Errorf("SanitizeJSON()\ngot:  %q\nwant: %q", got, tt.want)
			}
		})
	}
}
