package main

import (
	"reflect"
	"testing"
)

func TestNormalizeArgsAllowsFlagsAfterInput(t *testing.T) {
	got := normalizeArgs([]string{"input.md", "-o", "out.pdf", "-theme", "elegant"})
	want := []string{"-o", "out.pdf", "-theme", "elegant", "input.md"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("normalizeArgs() = %#v, want %#v", got, want)
	}
}

func TestDefaultOutputPath(t *testing.T) {
	got := defaultOutputPath("docs/readme.md")
	want := "docs/readme.pdf"
	if got != want {
		t.Fatalf("defaultOutputPath() = %q, want %q", got, want)
	}
}
