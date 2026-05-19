// Package reverse converts PDF text extraction results into best-effort
// Markdown.
//
// PDF files generally store positioned text, fonts, and drawing commands rather
// than semantic document structure. This package therefore reconstructs Markdown
// with heuristics instead of promising a perfect inverse of Markdown-to-PDF.
package reverse
