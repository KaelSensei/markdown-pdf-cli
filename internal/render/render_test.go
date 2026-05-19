package render

import (
	"bytes"
	"testing"
)

func TestMarkdownToPDFWithThemes(t *testing.T) {
	for _, tc := range []struct {
		theme  string
		scheme string
	}{
		{"modern", "light"},
		{"modern", "dark"},
		{"classic", "light"},
		{"elegant", "dark"},
	} {
		out, err := MarkdownToPDF(`# Title

Text with [a link](https://example.com).

- one
- two

| A | B |
| --- | --- |
| C | D |
`, Options{Title: "Test", PageSize: "a4", Margin: 56, Theme: tc.theme, ColorScheme: tc.scheme})
		if err != nil {
			t.Fatalf("%s/%s returned error: %v", tc.theme, tc.scheme, err)
		}
		if !bytes.HasPrefix(out, []byte("%PDF-1.4")) {
			t.Fatalf("%s/%s did not generate a PDF", tc.theme, tc.scheme)
		}
	}
}

func TestRejectsUnknownTheme(t *testing.T) {
	_, err := MarkdownToPDF("# Test", Options{Theme: "unknown"})
	if err == nil {
		t.Fatal("expected an error for unknown theme")
	}
}
