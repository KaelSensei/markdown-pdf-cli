package reverse

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/KaelSensei/markdown-pdf-cli/internal/render"
)

func TestMarkdownFromPagesInfersCommonBlocks(t *testing.T) {
	got := MarkdownFromPages([]PageText{
		{
			Number: 1,
			Lines: []Line{
				{Y: 760, FontSize: 28, Chunks: []Chunk{{Text: "Project Notes", X: 56, Font: "Helvetica-Bold", FontSize: 28}}},
				{Y: 720, FontSize: 11, Chunks: []Chunk{{Text: "This is an extracted paragraph", X: 56, Font: "Helvetica", FontSize: 11}}},
				{Y: 704, FontSize: 11, Chunks: []Chunk{{Text: "that continues on the next line.", X: 56, Font: "Helvetica", FontSize: 11}}},
				{Y: 668, FontSize: 11, Chunks: []Chunk{{Text: "•", X: 56, Font: "Helvetica", FontSize: 11}, {Text: "offline conversion", X: 80, Font: "Helvetica", FontSize: 11}}},
				{Y: 648, FontSize: 9, Chunks: []Chunk{{Text: "fmt.Println(\"hello\")", X: 56, Font: "Courier", FontSize: 9}}},
				{Y: 628, FontSize: 11, Chunks: []Chunk{{Text: "quoted text", X: 70, Font: "Helvetica-Oblique", FontSize: 11}}},
				{Y: 598, FontSize: 9, Chunks: []Chunk{{Text: "Name", X: 56, Font: "Helvetica-Bold", FontSize: 9}, {Text: "Value", X: 160, Font: "Helvetica-Bold", FontSize: 9}}},
				{Y: 580, FontSize: 9, Chunks: []Chunk{{Text: "Theme", X: 56, Font: "Helvetica", FontSize: 9}, {Text: "modern", X: 160, Font: "Helvetica", FontSize: 9}}},
			},
		},
	}, Options{})

	assertContains(t, got, "# Project Notes")
	assertContains(t, got, "This is an extracted paragraph that continues on the next line.")
	assertContains(t, got, "- offline conversion")
	assertContains(t, got, "```")
	assertContains(t, got, "fmt.Println(\"hello\")")
	assertContains(t, got, "> quoted text")
	assertContains(t, got, "| Name | Value |")
	assertContains(t, got, "| Theme | modern |")
}

func TestPDFFileToMarkdownRoundTripFromGeneratedPDF(t *testing.T) {
	tmp := t.TempDir()
	pdfPath := filepath.Join(tmp, "sample.pdf")

	pdfBytes, err := render.MarkdownToPDF(`# Sample Title

This document can be reversed into readable Markdown.

- offline
- best effort

`+"```go"+`
fmt.Println("hello")
`+"```"+`

| Area | Status |
| --- | --- |
| Text | Supported |
`, render.Options{Title: "Sample", Theme: "modern", ColorScheme: "light"})
	if err != nil {
		t.Fatalf("render Markdown to PDF: %v", err)
	}
	if err := os.WriteFile(pdfPath, pdfBytes, 0o644); err != nil {
		t.Fatalf("write generated PDF: %v", err)
	}

	got, err := PDFFileToMarkdown(pdfPath, Options{})
	if err != nil {
		t.Fatalf("reverse generated PDF: %v", err)
	}

	assertContains(t, got, "# Sample Title")
	assertContains(t, got, "This document can be reversed into readable Markdown.")
	assertContains(t, got, "- offline")
	assertContains(t, got, "- best effort")
	assertContains(t, got, "fmt.Println(\"hello\")")
	assertContains(t, got, "| Area | Status |")
}

func assertContains(t *testing.T, got, want string) {
	t.Helper()
	if !strings.Contains(got, want) {
		t.Fatalf("expected Markdown to contain %q\n\nMarkdown:\n%s", want, got)
	}
}
