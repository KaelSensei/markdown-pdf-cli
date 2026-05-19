package e2e_test

import (
	"archive/zip"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestCLIRoundTripsMarkdownPDFMarkdown(t *testing.T) {
	root := repositoryRoot(t)
	tmp := t.TempDir()

	inputPath := filepath.Join(tmp, "input.md")
	pdfPath := filepath.Join(tmp, "input.pdf")
	reversedPath := filepath.Join(tmp, "input.reverse.md")

	input := `# E2E Sample

This document exercises the public CLI.

- offline
- reversible

` + "```go" + `
fmt.Println("hello")
` + "```" + `

| Area | Status |
| --- | --- |
| Text | Supported |
`
	if err := os.WriteFile(inputPath, []byte(input), 0o644); err != nil {
		t.Fatalf("write input Markdown: %v", err)
	}

	run(t, root, "go", "run", "./cmd/mdpdf", inputPath, "-o", pdfPath, "-quiet")
	run(t, root, "go", "run", "./cmd/mdpdf", "reverse", pdfPath, "-o", reversedPath, "-quiet")

	output, err := os.ReadFile(reversedPath)
	if err != nil {
		t.Fatalf("read reversed Markdown: %v", err)
	}
	markdown := string(output)

	assertContains(t, markdown, "# E2E Sample")
	assertContains(t, markdown, "This document exercises the public CLI.")
	assertContains(t, markdown, "- offline")
	assertContains(t, markdown, "- reversible")
	assertContains(t, markdown, "fmt.Println(\"hello\")")
	assertContains(t, markdown, "| Area | Status |")
}

func TestCLIConvertsAdditionalInputFormatsToPDF(t *testing.T) {
	root := repositoryRoot(t)
	tmp := t.TempDir()

	txtPath := filepath.Join(tmp, "notes.txt")
	docxPath := filepath.Join(tmp, "notes.docx")
	pngPath := filepath.Join(tmp, "diagram.png")
	jpgPath := filepath.Join(tmp, "photo.jpg")

	if err := os.WriteFile(txtPath, []byte("Plain text input.\n\nSecond paragraph."), 0o644); err != nil {
		t.Fatalf("write text input: %v", err)
	}
	writeDocx(t, docxPath, `<?xml version="1.0" encoding="UTF-8"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:body>
    <w:p>
      <w:pPr><w:pStyle w:val="Heading1"/></w:pPr>
      <w:r><w:t>DOCX Title</w:t></w:r>
    </w:p>
    <w:p><w:r><w:t>DOCX paragraph.</w:t></w:r></w:p>
  </w:body>
</w:document>`)
	writePNG(t, pngPath)
	writeJPEG(t, jpgPath)

	for _, inputPath := range []string{txtPath, docxPath, pngPath, jpgPath} {
		outputPath := filepath.Join(tmp, strings.TrimSuffix(filepath.Base(inputPath), filepath.Ext(inputPath))+".pdf")
		run(t, root, "go", "run", "./cmd/mdpdf", inputPath, "-o", outputPath, "-quiet")
		assertPDF(t, outputPath)
	}
}

func run(t *testing.T, dir string, name string, args ...string) {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%s %s failed: %v\n%s", name, strings.Join(args, " "), err, output)
	}
}

func repositoryRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve test file path")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

func assertContains(t *testing.T, got, want string) {
	t.Helper()
	if !strings.Contains(got, want) {
		t.Fatalf("expected output to contain %q\n\nOutput:\n%s", want, got)
	}
}

func assertPDF(t *testing.T, path string) {
	t.Helper()
	output, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read PDF %s: %v", path, err)
	}
	if !strings.HasPrefix(string(output), "%PDF-1.4") {
		t.Fatalf("%s is not a PDF", path)
	}
}

func writeDocx(t *testing.T, path string, documentXML string) {
	t.Helper()
	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("create docx: %v", err)
	}
	defer file.Close()

	zipWriter := zip.NewWriter(file)
	writer, err := zipWriter.Create("word/document.xml")
	if err != nil {
		t.Fatalf("create document.xml: %v", err)
	}
	if _, err := writer.Write([]byte(documentXML)); err != nil {
		t.Fatalf("write document.xml: %v", err)
	}
	if err := zipWriter.Close(); err != nil {
		t.Fatalf("close docx: %v", err)
	}
}

func writePNG(t *testing.T, path string) {
	t.Helper()
	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("create png: %v", err)
	}
	defer file.Close()
	if err := png.Encode(file, testImage()); err != nil {
		t.Fatalf("encode png: %v", err)
	}
}

func writeJPEG(t *testing.T, path string) {
	t.Helper()
	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("create jpeg: %v", err)
	}
	defer file.Close()
	if err := jpeg.Encode(file, testImage(), &jpeg.Options{Quality: 85}); err != nil {
		t.Fatalf("encode jpeg: %v", err)
	}
}

func testImage() image.Image {
	img := image.NewRGBA(image.Rect(0, 0, 24, 12))
	for y := 0; y < 12; y++ {
		for x := 0; x < 24; x++ {
			img.Set(x, y, color.RGBA{R: uint8(10 * x), G: uint8(18 * y), B: 160, A: 255})
		}
	}
	return img
}
