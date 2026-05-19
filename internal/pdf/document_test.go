package pdf

import (
	"bytes"
	"testing"
)

func TestDocumentBytesCreatesPDF(t *testing.T) {
	doc := New(Size{Width: 200, Height: 200}, "Test")
	page := doc.AddPage()
	page.Text(20, 40, 12, "F1", "Hello PDF")

	out := doc.Bytes()
	if !bytes.HasPrefix(out, []byte("%PDF-1.4")) {
		t.Fatalf("PDF header missing")
	}
	if !bytes.Contains(out, []byte("/Type /Catalog")) {
		t.Fatalf("catalog object missing")
	}
	if !bytes.Contains(out, []byte("xref")) {
		t.Fatalf("xref table missing")
	}
}

func TestTextLiteralEscapesPDFSyntax(t *testing.T) {
	got := textLiteral(`hello) Tj ET /Evil << /JS (app.alert\(1\)) >> BT (` + "\nnext")

	if bytes.Contains([]byte(got), []byte("hello) Tj")) {
		t.Fatalf("text literal did not escape closing parenthesis: %q", got)
	}
	if !bytes.Contains([]byte(got), []byte(`hello\) Tj`)) {
		t.Fatalf("text literal should escape closing parenthesis: %q", got)
	}
	if !bytes.Contains([]byte(got), []byte(`app.alert\\\(1\\\)`)) {
		t.Fatalf("text literal should escape backslashes and parentheses: %q", got)
	}
	if !bytes.Contains([]byte(got), []byte(`\nnext`)) {
		t.Fatalf("text literal should encode newlines: %q", got)
	}
}

func TestWinAnsiEncodingKeepsLatinCharacters(t *testing.T) {
	got := encodeWinAnsi("Déjà vu — 10€")
	if !bytes.Contains(got, []byte{0xe9}) {
		t.Fatalf("expected e acute in WinAnsi output: %#v", got)
	}
	if !bytes.Contains(got, []byte{0x97}) {
		t.Fatalf("expected em dash in WinAnsi output: %#v", got)
	}
	if !bytes.Contains(got, []byte{0x80}) {
		t.Fatalf("expected euro in WinAnsi output: %#v", got)
	}
}
