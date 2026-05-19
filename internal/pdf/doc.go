// Package pdf writes a compact PDF document using only the Go standard library.
//
// It exposes drawing primitives used by the renderer and serializes the final
// document as PDF objects, page streams, font resources, an xref table, and
// trailer metadata. The package uses built-in Type 1 PDF fonts so generated
// files remain self-contained without bundling font files.
package pdf
