package jsonutil

import (
	"bytes"
	"fmt"
)

// SanitizeJSON fixes common JSON spec (RFC 8259) violations:
//   - Raw control characters (U+0000 through U+001F) inside string values are escaped
//   - Invalid escape sequences (e.g., \T) are fixed by escaping the backslash
func SanitizeJSON(data []byte) []byte {
	var buf bytes.Buffer
	buf.Grow(len(data))

	inString := false
	escaped := false

	for _, b := range data {
		if escaped {
			escaped = false
			switch b {
			case '"', '\\', '/', 'b', 'f', 'n', 'r', 't', 'u':
				buf.WriteByte(b)
			default:
				// Invalid escape like \T → \\T (escape the backslash)
				buf.WriteByte('\\')
				buf.WriteByte(b)
			}
			continue
		}

		if inString {
			switch {
			case b == '\\':
				escaped = true
				buf.WriteByte(b)
			case b == '"':
				inString = false
				buf.WriteByte(b)
			case b < 0x20:
				switch b {
				case '\t':
					buf.WriteString(`\t`)
				case '\n':
					buf.WriteString(`\n`)
				case '\r':
					buf.WriteString(`\r`)
				default:
					fmt.Fprintf(&buf, `\u%04x`, b)
				}
			default:
				buf.WriteByte(b)
			}
		} else {
			if b == '"' {
				inString = true
			}
			buf.WriteByte(b)
		}
	}

	return buf.Bytes()
}
