package render

import (
	"bytes"
	"image"
	"image/color"
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

func TestPlainTextToPDF(t *testing.T) {
	out, err := PlainTextToPDF("First paragraph.\n\nSecond paragraph.", Options{Title: "Text"})
	if err != nil {
		t.Fatalf("PlainTextToPDF returned error: %v", err)
	}
	if !bytes.HasPrefix(out, []byte("%PDF-1.4")) {
		t.Fatal("plain text conversion did not generate a PDF")
	}
	if !bytes.Contains(out, []byte("First paragraph")) {
		t.Fatal("plain text content missing from PDF stream")
	}
}

func TestImageToPDF(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 4, 2))
	for y := 0; y < 2; y++ {
		for x := 0; x < 4; x++ {
			img.Set(x, y, color.RGBA{R: 40, G: 100, B: 180, A: 255})
		}
	}

	out, err := ImageToPDF(img, Options{Title: "Image"})
	if err != nil {
		t.Fatalf("ImageToPDF returned error: %v", err)
	}
	if !bytes.HasPrefix(out, []byte("%PDF-1.4")) {
		t.Fatal("image conversion did not generate a PDF")
	}
	if !bytes.Contains(out, []byte("/Subtype /Image")) {
		t.Fatal("image resource missing from PDF")
	}
}
