package e2e_test

import (
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
