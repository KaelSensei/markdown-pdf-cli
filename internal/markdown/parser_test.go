package markdown

import "testing"

func TestParseCommonBlocks(t *testing.T) {
	doc := Parse(`# Title

Paragraph with [a link](https://example.com) and **strong** text.

- one
- two

` + "```go" + `
fmt.Println("hello")
` + "```" + `

> quoted text

| A | B |
| --- | --- |
| C | D |
`)

	if got, want := len(doc.Blocks), 6; got != want {
		t.Fatalf("block count = %d, want %d", got, want)
	}
	if doc.Blocks[0].Type != Heading || doc.Blocks[0].Text != "Title" {
		t.Fatalf("unexpected heading: %#v", doc.Blocks[0])
	}
	if doc.Blocks[1].Text != "Paragraph with a link (https://example.com) and strong text." {
		t.Fatalf("unexpected paragraph text: %q", doc.Blocks[1].Text)
	}
	if len(doc.Blocks[2].Items) != 2 {
		t.Fatalf("list item count = %d, want 2", len(doc.Blocks[2].Items))
	}
	if doc.Blocks[3].Type != CodeBlock || doc.Blocks[3].Lines[0] != `fmt.Println("hello")` {
		t.Fatalf("unexpected code block: %#v", doc.Blocks[3])
	}
	if doc.Blocks[5].Type != Table || len(doc.Blocks[5].Rows) != 1 {
		t.Fatalf("unexpected table: %#v", doc.Blocks[5])
	}
}

func TestSkipsFrontMatter(t *testing.T) {
	doc := Parse(`---
title: Private Metadata
---

# Public Title
`)

	if got, want := len(doc.Blocks), 1; got != want {
		t.Fatalf("block count = %d, want %d", got, want)
	}
	if doc.Blocks[0].Text != "Public Title" {
		t.Fatalf("heading text = %q, want Public Title", doc.Blocks[0].Text)
	}
}

func TestPlainInlineImages(t *testing.T) {
	got := PlainInline(`See ![diagram](arch.png) and ` + "`code`" + `.`)
	want := "See [image: diagram] and code."
	if got != want {
		t.Fatalf("PlainInline() = %q, want %q", got, want)
	}
}
