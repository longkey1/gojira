package adf

import (
	"fmt"
	"regexp"
	"strings"
)

var linkRegex = regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`)

// ToMarkdown converts an ADF document to a Markdown string.
func ToMarkdown(doc map[string]any) string {
	content, ok := doc["content"].([]any)
	if !ok {
		return ""
	}

	var parts []string
	for _, block := range content {
		b, ok := block.(map[string]any)
		if !ok {
			continue
		}
		parts = append(parts, blockToMarkdown(b))
	}
	return strings.Join(parts, "\n")
}

func blockToMarkdown(block map[string]any) string {
	blockType, _ := block["type"].(string)
	content, _ := block["content"].([]any)

	switch blockType {
	case "heading":
		level := 1
		if attrs, ok := block["attrs"].(map[string]any); ok {
			if l, ok := attrs["level"].(float64); ok {
				level = int(l)
			} else if l, ok := attrs["level"].(int); ok {
				level = l
			}
		}
		return strings.Repeat("#", level) + " " + inlinesToMarkdown(content)

	case "paragraph":
		return inlinesToMarkdown(content)

	case "bulletList":
		var lines []string
		for _, item := range content {
			li, ok := item.(map[string]any)
			if !ok {
				continue
			}
			lines = append(lines, "- "+listItemText(li))
		}
		return strings.Join(lines, "\n")

	case "orderedList":
		var lines []string
		for i, item := range content {
			li, ok := item.(map[string]any)
			if !ok {
				continue
			}
			lines = append(lines, fmt.Sprintf("%d. %s", i+1, listItemText(li)))
		}
		return strings.Join(lines, "\n")

	case "blockquote":
		var lines []string
		for _, child := range content {
			c, ok := child.(map[string]any)
			if !ok {
				continue
			}
			lines = append(lines, "> "+blockToMarkdown(c))
		}
		return strings.Join(lines, "\n")

	default:
		return inlinesToMarkdown(content)
	}
}

func listItemText(li map[string]any) string {
	content, ok := li["content"].([]any)
	if !ok {
		return ""
	}
	var parts []string
	for _, child := range content {
		c, ok := child.(map[string]any)
		if !ok {
			continue
		}
		parts = append(parts, blockToMarkdown(c))
	}
	return strings.Join(parts, "\n")
}

func inlinesToMarkdown(nodes []any) string {
	var sb strings.Builder
	for _, node := range nodes {
		n, ok := node.(map[string]any)
		if !ok {
			continue
		}
		text, _ := n["text"].(string)
		if hasLinkMark(n) {
			href := getLinkHref(n)
			sb.WriteString("[" + text + "](" + href + ")")
		} else {
			sb.WriteString(text)
		}
	}
	return sb.String()
}

func hasLinkMark(node map[string]any) bool {
	marks, ok := node["marks"].([]any)
	if !ok {
		return false
	}
	for _, m := range marks {
		mark, ok := m.(map[string]any)
		if !ok {
			continue
		}
		if mark["type"] == "link" {
			return true
		}
	}
	return false
}

func getLinkHref(node map[string]any) string {
	marks, ok := node["marks"].([]any)
	if !ok {
		return ""
	}
	for _, m := range marks {
		mark, ok := m.(map[string]any)
		if !ok {
			continue
		}
		if mark["type"] == "link" {
			if attrs, ok := mark["attrs"].(map[string]any); ok {
				href, _ := attrs["href"].(string)
				return href
			}
		}
	}
	return ""
}

// FromMarkdown converts a Markdown string to an ADF (Atlassian Document Format) document.
func FromMarkdown(md string) map[string]any {
	lines := strings.Split(md, "\n")
	content := []any{}

	i := 0
	for i < len(lines) {
		line := lines[i]

		// Skip empty lines
		if strings.TrimSpace(line) == "" {
			i++
			continue
		}

		// Heading
		if level, text, ok := parseHeading(line); ok {
			content = append(content, map[string]any{
				"type": "heading",
				"attrs": map[string]any{
					"level": level,
				},
				"content": ParseInline(text),
			})
			i++
			continue
		}

		// Bullet list
		if text, ok := parseBulletItem(line); ok {
			items := []any{bulletListItem(text)}
			i++
			for i < len(lines) {
				if t, ok := parseBulletItem(lines[i]); ok {
					items = append(items, bulletListItem(t))
					i++
				} else {
					break
				}
			}
			content = append(content, map[string]any{
				"type":    "bulletList",
				"content": items,
			})
			continue
		}

		// Ordered list
		if text, ok := parseOrderedItem(line); ok {
			items := []any{orderedListItem(text)}
			i++
			for i < len(lines) {
				if t, ok := parseOrderedItem(lines[i]); ok {
					items = append(items, orderedListItem(t))
					i++
				} else {
					break
				}
			}
			content = append(content, map[string]any{
				"type":    "orderedList",
				"content": items,
			})
			continue
		}

		// Blockquote
		if text, ok := parseBlockquote(line); ok {
			texts := []string{text}
			i++
			for i < len(lines) {
				if t, ok := parseBlockquote(lines[i]); ok {
					texts = append(texts, t)
					i++
				} else {
					break
				}
			}
			quoteContent := []any{}
			for _, t := range texts {
				quoteContent = append(quoteContent, map[string]any{
					"type":    "paragraph",
					"content": ParseInline(t),
				})
			}
			content = append(content, map[string]any{
				"type":    "blockquote",
				"content": quoteContent,
			})
			continue
		}

		// Paragraph (default)
		content = append(content, map[string]any{
			"type":    "paragraph",
			"content": ParseInline(line),
		})
		i++
	}

	return map[string]any{
		"type":    "doc",
		"version": 1,
		"content": content,
	}
}

// ParseInline parses inline elements (links) from text and returns ADF inline nodes.
func ParseInline(text string) []any {
	var nodes []any

	matches := linkRegex.FindAllStringSubmatchIndex(text, -1)
	if len(matches) == 0 {
		return []any{textNode(text)}
	}

	lastEnd := 0
	for _, match := range matches {
		// Text before the link
		if match[0] > lastEnd {
			nodes = append(nodes, textNode(text[lastEnd:match[0]]))
		}
		// Link text and URL
		linkText := text[match[2]:match[3]]
		linkURL := text[match[4]:match[5]]
		nodes = append(nodes, map[string]any{
			"type": "text",
			"text": linkText,
			"marks": []any{
				map[string]any{
					"type": "link",
					"attrs": map[string]any{
						"href": linkURL,
					},
				},
			},
		})
		lastEnd = match[1]
	}
	// Text after the last link
	if lastEnd < len(text) {
		nodes = append(nodes, textNode(text[lastEnd:]))
	}

	return nodes
}

func textNode(text string) map[string]any {
	return map[string]any{
		"type": "text",
		"text": text,
	}
}

func parseHeading(line string) (int, string, bool) {
	trimmed := line
	level := 0
	for level < 6 && level < len(trimmed) && trimmed[level] == '#' {
		level++
	}
	if level == 0 || level >= len(trimmed) || trimmed[level] != ' ' {
		return 0, "", false
	}
	return level, strings.TrimSpace(trimmed[level+1:]), true
}

func parseBulletItem(line string) (string, bool) {
	trimmed := strings.TrimLeft(line, " \t")
	if strings.HasPrefix(trimmed, "- ") {
		return strings.TrimSpace(trimmed[2:]), true
	}
	if strings.HasPrefix(trimmed, "* ") {
		return strings.TrimSpace(trimmed[2:]), true
	}
	return "", false
}

func parseOrderedItem(line string) (string, bool) {
	trimmed := strings.TrimLeft(line, " \t")
	for i, c := range trimmed {
		if c >= '0' && c <= '9' {
			continue
		}
		if c == '.' && i > 0 && i+1 < len(trimmed) && trimmed[i+1] == ' ' {
			return strings.TrimSpace(trimmed[i+2:]), true
		}
		break
	}
	return "", false
}

func parseBlockquote(line string) (string, bool) {
	trimmed := strings.TrimLeft(line, " \t")
	if strings.HasPrefix(trimmed, "> ") {
		return strings.TrimSpace(trimmed[2:]), true
	}
	if trimmed == ">" {
		return "", true
	}
	return "", false
}

func bulletListItem(text string) map[string]any {
	return map[string]any{
		"type": "listItem",
		"content": []any{
			map[string]any{
				"type":    "paragraph",
				"content": ParseInline(text),
			},
		},
	}
}

func orderedListItem(text string) map[string]any {
	return map[string]any{
		"type": "listItem",
		"content": []any{
			map[string]any{
				"type":    "paragraph",
				"content": ParseInline(text),
			},
		},
	}
}
