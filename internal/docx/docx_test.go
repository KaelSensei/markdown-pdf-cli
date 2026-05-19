package docx

import (
	"archive/zip"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFileToMarkdownExtractsHeadingsParagraphsAndLists(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sample.docx")
	writeTestDocx(t, path, `<?xml version="1.0" encoding="UTF-8"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:body>
    <w:p>
      <w:pPr><w:pStyle w:val="Heading1"/></w:pPr>
      <w:r><w:t>Project Notes</w:t></w:r>
    </w:p>
    <w:p>
      <w:r><w:t>Hello from DOCX.</w:t></w:r>
      <w:r><w:tab/></w:r>
      <w:r><w:t>Tabbed text.</w:t></w:r>
    </w:p>
    <w:p>
      <w:pPr><w:numPr><w:ilvl w:val="0"/></w:numPr></w:pPr>
      <w:r><w:t>First item</w:t></w:r>
    </w:p>
  </w:body>
</w:document>`)

	got, err := FileToMarkdown(path)
	if err != nil {
		t.Fatalf("FileToMarkdown returned error: %v", err)
	}

	for _, want := range []string{
		"# Project Notes",
		"Hello from DOCX. Tabbed text.",
		"- First item",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected Markdown to contain %q\n\nMarkdown:\n%s", want, got)
		}
	}
}

func TestFileToMarkdownEscapesAccidentalMarkdown(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sample.docx")
	writeTestDocx(t, path, `<?xml version="1.0" encoding="UTF-8"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:body>
    <w:p><w:r><w:t># Not a heading</w:t></w:r></w:p>
    <w:p><w:r><w:t>1. Not a list</w:t></w:r></w:p>
  </w:body>
</w:document>`)

	got, err := FileToMarkdown(path)
	if err != nil {
		t.Fatalf("FileToMarkdown returned error: %v", err)
	}
	if !strings.Contains(got, `\# Not a heading`) {
		t.Fatalf("expected heading marker to be escaped\n\nMarkdown:\n%s", got)
	}
	if !strings.Contains(got, `\1. Not a list`) {
		t.Fatalf("expected ordered list marker to be escaped\n\nMarkdown:\n%s", got)
	}
}

func writeTestDocx(t *testing.T, path string, documentXML string) {
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
