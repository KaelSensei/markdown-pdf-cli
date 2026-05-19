// Package markdown parses the subset of Markdown that mdpdf renders directly.
//
// The parser is intentionally small and predictable. It does not try to be a
// full CommonMark implementation; instead it extracts document blocks that the
// PDF renderer can lay out without relying on HTML, browsers, or external tools.
package markdown
