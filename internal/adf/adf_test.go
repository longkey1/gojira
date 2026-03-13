package adf

import (
	"encoding/json"
	"testing"
)

func toJSON(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}

func TestFromMarkdownHeading(t *testing.T) {
	tests := []struct {
		input string
		level int
		text  string
	}{
		{"# Heading 1", 1, "Heading 1"},
		{"## Heading 2", 2, "Heading 2"},
		{"### Heading 3", 3, "Heading 3"},
		{"###### Heading 6", 6, "Heading 6"},
	}

	for _, tt := range tests {
		doc := FromMarkdown(tt.input)
		content := doc["content"].([]any)
		if len(content) != 1 {
			t.Fatalf("expected 1 block, got %d", len(content))
		}
		block := content[0].(map[string]any)
		if block["type"] != "heading" {
			t.Errorf("expected heading, got %s", block["type"])
		}
		attrs := block["attrs"].(map[string]any)
		if attrs["level"] != tt.level {
			t.Errorf("expected level %d, got %v", tt.level, attrs["level"])
		}
		inlines := block["content"].([]any)
		textNode := inlines[0].(map[string]any)
		if textNode["text"] != tt.text {
			t.Errorf("expected text %q, got %q", tt.text, textNode["text"])
		}
	}
}

func TestFromMarkdownBulletList(t *testing.T) {
	input := "- Item 1\n- Item 2\n- Item 3"
	doc := FromMarkdown(input)
	content := doc["content"].([]any)
	if len(content) != 1 {
		t.Fatalf("expected 1 block, got %d", len(content))
	}
	block := content[0].(map[string]any)
	if block["type"] != "bulletList" {
		t.Errorf("expected bulletList, got %s", block["type"])
	}
	items := block["content"].([]any)
	if len(items) != 3 {
		t.Errorf("expected 3 items, got %d", len(items))
	}
}

func TestFromMarkdownBulletListAsterisk(t *testing.T) {
	input := "* Item A\n* Item B"
	doc := FromMarkdown(input)
	content := doc["content"].([]any)
	if len(content) != 1 {
		t.Fatalf("expected 1 block, got %d", len(content))
	}
	block := content[0].(map[string]any)
	if block["type"] != "bulletList" {
		t.Errorf("expected bulletList, got %s", block["type"])
	}
	items := block["content"].([]any)
	if len(items) != 2 {
		t.Errorf("expected 2 items, got %d", len(items))
	}
}

func TestFromMarkdownOrderedList(t *testing.T) {
	input := "1. First\n2. Second\n3. Third"
	doc := FromMarkdown(input)
	content := doc["content"].([]any)
	if len(content) != 1 {
		t.Fatalf("expected 1 block, got %d", len(content))
	}
	block := content[0].(map[string]any)
	if block["type"] != "orderedList" {
		t.Errorf("expected orderedList, got %s", block["type"])
	}
	items := block["content"].([]any)
	if len(items) != 3 {
		t.Errorf("expected 3 items, got %d", len(items))
	}
}

func TestFromMarkdownBlockquote(t *testing.T) {
	input := "> Quote line 1\n> Quote line 2"
	doc := FromMarkdown(input)
	content := doc["content"].([]any)
	if len(content) != 1 {
		t.Fatalf("expected 1 block, got %d", len(content))
	}
	block := content[0].(map[string]any)
	if block["type"] != "blockquote" {
		t.Errorf("expected blockquote, got %s", block["type"])
	}
	paragraphs := block["content"].([]any)
	if len(paragraphs) != 2 {
		t.Errorf("expected 2 paragraphs in blockquote, got %d", len(paragraphs))
	}
}

func TestFromMarkdownParagraph(t *testing.T) {
	input := "Just a plain paragraph"
	doc := FromMarkdown(input)
	content := doc["content"].([]any)
	if len(content) != 1 {
		t.Fatalf("expected 1 block, got %d", len(content))
	}
	block := content[0].(map[string]any)
	if block["type"] != "paragraph" {
		t.Errorf("expected paragraph, got %s", block["type"])
	}
}

func TestFromMarkdownLink(t *testing.T) {
	input := "See [example](https://example.com) for details"
	doc := FromMarkdown(input)
	content := doc["content"].([]any)
	block := content[0].(map[string]any)
	inlines := block["content"].([]any)

	if len(inlines) != 3 {
		t.Fatalf("expected 3 inline nodes, got %d: %s", len(inlines), toJSON(inlines))
	}

	// "See "
	node0 := inlines[0].(map[string]any)
	if node0["text"] != "See " {
		t.Errorf("expected 'See ', got %q", node0["text"])
	}

	// link
	node1 := inlines[1].(map[string]any)
	if node1["text"] != "example" {
		t.Errorf("expected 'example', got %q", node1["text"])
	}
	marks := node1["marks"].([]any)
	mark := marks[0].(map[string]any)
	attrs := mark["attrs"].(map[string]any)
	if attrs["href"] != "https://example.com" {
		t.Errorf("expected href 'https://example.com', got %q", attrs["href"])
	}

	// " for details"
	node2 := inlines[2].(map[string]any)
	if node2["text"] != " for details" {
		t.Errorf("expected ' for details', got %q", node2["text"])
	}
}

func TestFromMarkdownComplex(t *testing.T) {
	input := "## Overview\n- Item 1\n- Item 2\n> Quote\nPlain text"
	doc := FromMarkdown(input)
	content := doc["content"].([]any)

	if len(content) != 4 {
		t.Fatalf("expected 4 blocks, got %d", len(content))
	}

	types := []string{"heading", "bulletList", "blockquote", "paragraph"}
	for i, expected := range types {
		block := content[i].(map[string]any)
		if block["type"] != expected {
			t.Errorf("block %d: expected %s, got %s", i, expected, block["type"])
		}
	}
}

func TestFromMarkdownEmptyLines(t *testing.T) {
	input := "Line 1\n\n\nLine 2"
	doc := FromMarkdown(input)
	content := doc["content"].([]any)
	if len(content) != 2 {
		t.Fatalf("expected 2 blocks, got %d", len(content))
	}
}

func TestFromMarkdownDocStructure(t *testing.T) {
	doc := FromMarkdown("Hello")
	if doc["type"] != "doc" {
		t.Errorf("expected type 'doc', got %v", doc["type"])
	}
	if doc["version"] != 1 {
		t.Errorf("expected version 1, got %v", doc["version"])
	}
}

func TestParseInlinePlainText(t *testing.T) {
	nodes := ParseInline("plain text")
	if len(nodes) != 1 {
		t.Fatalf("expected 1 node, got %d", len(nodes))
	}
	node := nodes[0].(map[string]any)
	if node["text"] != "plain text" {
		t.Errorf("expected 'plain text', got %q", node["text"])
	}
}

func TestToMarkdownRoundtrip(t *testing.T) {
	input := "## Overview\n- Item 1\n- Item 2\n> Quote\nPlain text"
	doc := FromMarkdown(input)
	result := ToMarkdown(doc)
	if result != input {
		t.Errorf("roundtrip mismatch:\nexpected: %q\ngot:      %q", input, result)
	}
}

func TestToMarkdownHeading(t *testing.T) {
	doc := FromMarkdown("### Title")
	result := ToMarkdown(doc)
	if result != "### Title" {
		t.Errorf("expected '### Title', got %q", result)
	}
}

func TestToMarkdownOrderedList(t *testing.T) {
	doc := FromMarkdown("1. A\n2. B")
	result := ToMarkdown(doc)
	if result != "1. A\n2. B" {
		t.Errorf("expected '1. A\\n2. B', got %q", result)
	}
}

func TestToMarkdownLink(t *testing.T) {
	doc := FromMarkdown("See [link](https://example.com)")
	result := ToMarkdown(doc)
	if result != "See [link](https://example.com)" {
		t.Errorf("expected 'See [link](https://example.com)', got %q", result)
	}
}

func TestToMarkdownBlockquote(t *testing.T) {
	doc := FromMarkdown("> line 1\n> line 2")
	result := ToMarkdown(doc)
	if result != "> line 1\n> line 2" {
		t.Errorf("expected '> line 1\\n> line 2', got %q", result)
	}
}

func TestParseInlineMultipleLinks(t *testing.T) {
	nodes := ParseInline("[a](http://a.com) and [b](http://b.com)")
	if len(nodes) != 3 {
		t.Fatalf("expected 3 nodes, got %d: %s", len(nodes), toJSON(nodes))
	}
	node0 := nodes[0].(map[string]any)
	if node0["text"] != "a" {
		t.Errorf("expected 'a', got %q", node0["text"])
	}
	node1 := nodes[1].(map[string]any)
	if node1["text"] != " and " {
		t.Errorf("expected ' and ', got %q", node1["text"])
	}
	node2 := nodes[2].(map[string]any)
	if node2["text"] != "b" {
		t.Errorf("expected 'b', got %q", node2["text"])
	}
}
