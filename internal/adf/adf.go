package adf

import (
	"fmt"
	"regexp"
	"strings"
)

// inlineRegex matches, in order of capture groups: inline code, bold, link(text, url).
var inlineRegex = regexp.MustCompile("`([^`]+)`|\\*\\*([^*]+)\\*\\*|\\[([^\\]]+)\\]\\(([^)]+)\\)")

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

	case "bulletList", "orderedList":
		return listToMarkdown(blockType == "orderedList", content)

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

	case "codeBlock":
		lang := ""
		if attrs, ok := block["attrs"].(map[string]any); ok {
			lang, _ = attrs["language"].(string)
		}
		var sb strings.Builder
		for _, n := range content {
			if node, ok := n.(map[string]any); ok {
				text, _ := node["text"].(string)
				sb.WriteString(text)
			}
		}
		return "```" + lang + "\n" + sb.String() + "\n```"

	case "table":
		return tableToMarkdown(content)

	case "rule":
		return "---"

	default:
		return inlinesToMarkdown(content)
	}
}

func tableToMarkdown(rows []any) string {
	var lines []string
	for i, r := range rows {
		row, ok := r.(map[string]any)
		if !ok {
			continue
		}
		cells, _ := row["content"].([]any)
		texts := make([]string, 0, len(cells))
		for _, c := range cells {
			cell, ok := c.(map[string]any)
			if !ok {
				continue
			}
			// Markdown tables cannot hold line breaks or raw pipes in a cell.
			text := strings.ReplaceAll(listItemText(cell), "\n", "<br>")
			text = strings.ReplaceAll(text, "|", `\|`)
			texts = append(texts, text)
		}
		lines = append(lines, "| "+strings.Join(texts, " | ")+" |")
		if i == 0 {
			seps := make([]string, len(texts))
			for j := range seps {
				seps[j] = "---"
			}
			lines = append(lines, "| "+strings.Join(seps, " | ")+" |")
		}
	}
	return strings.Join(lines, "\n")
}

// listToMarkdown renders list items; every block after an item's first is indented by the marker width
// so that nested lists stay inside their parent item when parsed back.
func listToMarkdown(ordered bool, items []any) string {
	var lines []string
	n := 0
	for _, item := range items {
		li, ok := item.(map[string]any)
		if !ok {
			continue
		}
		n++
		marker := "- "
		if ordered {
			marker = fmt.Sprintf("%d. ", n)
		}
		pad := strings.Repeat(" ", len(marker))

		children, _ := li["content"].([]any)
		var parts []string
		for _, child := range children {
			if c, ok := child.(map[string]any); ok {
				parts = append(parts, blockToMarkdown(c))
			}
		}
		itemLines := strings.Split(strings.Join(parts, "\n"), "\n")
		lines = append(lines, marker+itemLines[0])
		for _, l := range itemLines[1:] {
			lines = append(lines, pad+l)
		}
	}
	return strings.Join(lines, "\n")
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
		marks, _ := n["marks"].([]any)
		var isCode, isStrong bool
		href, isLink := "", false
		for _, m := range marks {
			mark, ok := m.(map[string]any)
			if !ok {
				continue
			}
			switch mark["type"] {
			case "code":
				isCode = true
			case "strong":
				isStrong = true
			case "link":
				isLink = true
				if attrs, ok := mark["attrs"].(map[string]any); ok {
					href, _ = attrs["href"].(string)
				}
			}
		}
		if isCode {
			text = "`" + text + "`"
		}
		if isStrong {
			text = "**" + text + "**"
		}
		if isLink {
			text = "[" + text + "](" + href + ")"
		}
		sb.WriteString(text)
	}
	return sb.String()
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

		// Fenced code block
		if lang, ok := parseCodeFence(line); ok {
			i++
			var codeLines []string
			for i < len(lines) && !isCodeFenceEnd(lines[i]) {
				codeLines = append(codeLines, lines[i])
				i++
			}
			i++ // closing fence (or end of input for an unterminated block)
			block := map[string]any{"type": "codeBlock"}
			if lang != "" {
				block["attrs"] = map[string]any{"language": lang}
			}
			if code := strings.Join(codeLines, "\n"); code != "" {
				block["content"] = []any{textNode(code)}
			}
			content = append(content, block)
			continue
		}

		// Table: header row followed by a separator row
		if i+1 < len(lines) && isTableRow(line) && isTableSeparator(lines[i+1]) {
			rows := []any{tableRow(splitTableRow(line), "tableHeader")}
			i += 2
			for i < len(lines) && isTableRow(lines[i]) {
				rows = append(rows, tableRow(splitTableRow(lines[i]), "tableCell"))
				i++
			}
			content = append(content, map[string]any{
				"type":    "table",
				"content": rows,
			})
			continue
		}

		// Thematic break. Checked before lists so "- - -" / "* * *" are not parsed as list items.
		if isThematicBreak(line) {
			content = append(content, map[string]any{"type": "rule"})
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

		// Bullet or ordered list, possibly nested
		if _, ok := parseListItem(line); ok {
			var list map[string]any
			list, i = parseList(lines, i)
			content = append(content, list)
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

// ParseInline parses inline elements (inline code, bold, links) from text and returns ADF inline nodes.
func ParseInline(text string) []any {
	var nodes []any

	matches := inlineRegex.FindAllStringSubmatchIndex(text, -1)
	if len(matches) == 0 {
		return []any{textNode(text)}
	}

	lastEnd := 0
	for _, match := range matches {
		if match[0] > lastEnd {
			nodes = append(nodes, textNode(text[lastEnd:match[0]]))
		}
		switch {
		case match[2] >= 0: // `code`
			nodes = append(nodes, markedTextNode(text[match[2]:match[3]], map[string]any{"type": "code"}))
		case match[4] >= 0: // **bold**
			nodes = append(nodes, markedTextNode(text[match[4]:match[5]], map[string]any{"type": "strong"}))
		default: // [text](url)
			nodes = append(nodes, markedTextNode(text[match[6]:match[7]], map[string]any{
				"type":  "link",
				"attrs": map[string]any{"href": text[match[8]:match[9]]},
			}))
		}
		lastEnd = match[1]
	}
	if lastEnd < len(text) {
		nodes = append(nodes, textNode(text[lastEnd:]))
	}

	return nodes
}

func markedTextNode(text string, mark map[string]any) map[string]any {
	return map[string]any{
		"type":  "text",
		"text":  text,
		"marks": []any{mark},
	}
}

func parseCodeFence(line string) (string, bool) {
	trimmed := strings.TrimSpace(line)
	if !strings.HasPrefix(trimmed, "```") {
		return "", false
	}
	return strings.TrimSpace(trimmed[3:]), true
}

func isCodeFenceEnd(line string) bool {
	return strings.TrimSpace(line) == "```"
}

func isTableRow(line string) bool {
	trimmed := strings.TrimSpace(line)
	return len(trimmed) >= 2 && strings.HasPrefix(trimmed, "|") && strings.HasSuffix(trimmed, "|")
}

func isTableSeparator(line string) bool {
	if !isTableRow(line) {
		return false
	}
	for _, cell := range splitTableRow(line) {
		cell = strings.Trim(cell, ":")
		if len(cell) == 0 || strings.Trim(cell, "-") != "" {
			return false
		}
	}
	return true
}

// splitTableRow splits a Markdown table row into trimmed cells, honoring `\|` escapes.
func splitTableRow(line string) []string {
	trimmed := strings.TrimSpace(line)
	trimmed = strings.TrimSuffix(strings.TrimPrefix(trimmed, "|"), "|")

	var cells []string
	var sb strings.Builder
	for i := 0; i < len(trimmed); i++ {
		switch {
		case trimmed[i] == '\\' && i+1 < len(trimmed) && trimmed[i+1] == '|':
			sb.WriteByte('|')
			i++
		case trimmed[i] == '|':
			cells = append(cells, strings.TrimSpace(sb.String()))
			sb.Reset()
		default:
			sb.WriteByte(trimmed[i])
		}
	}
	return append(cells, strings.TrimSpace(sb.String()))
}

func tableRow(cells []string, cellType string) map[string]any {
	items := make([]any, 0, len(cells))
	for _, c := range cells {
		items = append(items, map[string]any{
			"type": cellType,
			"content": []any{
				map[string]any{
					"type":    "paragraph",
					"content": ParseInline(c),
				},
			},
		})
	}
	return map[string]any{
		"type":    "tableRow",
		"content": items,
	}
}

func textNode(text string) map[string]any {
	return map[string]any{
		"type": "text",
		"text": text,
	}
}

// isThematicBreak reports whether line is three or more of the same '-', '*' or '_',
// optionally separated by spaces or tabs.
func isThematicBreak(line string) bool {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return false
	}
	marker := trimmed[0]
	if marker != '-' && marker != '*' && marker != '_' {
		return false
	}
	count := 0
	for i := 0; i < len(trimmed); i++ {
		switch trimmed[i] {
		case marker:
			count++
		case ' ', '\t':
		default:
			return false
		}
	}
	return count >= 3
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

type listItemLine struct {
	indent  int // leading whitespace width in columns, tabs expanded to the next multiple of 4
	ordered bool
	text    string
}

func parseListItem(line string) (listItemLine, bool) {
	indent, pos := 0, 0
	for pos < len(line) && (line[pos] == ' ' || line[pos] == '\t') {
		if line[pos] == '\t' {
			indent += 4 - indent%4
		} else {
			indent++
		}
		pos++
	}
	rest := line[pos:]

	if strings.HasPrefix(rest, "- ") || strings.HasPrefix(rest, "* ") {
		return listItemLine{indent: indent, text: strings.TrimSpace(rest[2:])}, true
	}
	for j, c := range rest {
		if c >= '0' && c <= '9' {
			continue
		}
		if c == '.' && j > 0 && j+1 < len(rest) && rest[j+1] == ' ' {
			return listItemLine{indent: indent, ordered: true, text: strings.TrimSpace(rest[j+2:])}, true
		}
		break
	}
	return listItemLine{}, false
}

// parseList parses a list starting at lines[i] and returns it with the index of the first unconsumed line.
// Items indented deeper than the list's own items form a child list of the preceding item; any deeper
// indent is accepted (not only the parent's marker width) so that 2-space children under "1." still nest.
// The list ends at a non-item line, a shallower item, or a same-level item of the other list type.
func parseList(lines []string, i int) (map[string]any, int) {
	first, _ := parseListItem(lines[i])
	items := []any{}
	for i < len(lines) {
		if isThematicBreak(lines[i]) {
			break
		}
		item, ok := parseListItem(lines[i])
		if !ok || item.indent < first.indent {
			break
		}
		if item.indent > first.indent {
			var child map[string]any
			child, i = parseList(lines, i)
			last := items[len(items)-1].(map[string]any)
			last["content"] = append(last["content"].([]any), child)
			continue
		}
		if item.ordered != first.ordered {
			break
		}
		items = append(items, listItem(item.text))
		i++
	}

	listType := "bulletList"
	if first.ordered {
		listType = "orderedList"
	}
	return map[string]any{
		"type":    listType,
		"content": items,
	}, i
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

func listItem(text string) map[string]any {
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
