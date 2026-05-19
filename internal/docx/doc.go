// Package docx extracts text from DOCX files for the PDF renderer.
//
// DOCX is a ZIP package of XML files. This package intentionally reads only
// word/document.xml and maps common paragraph structure into a small Markdown
// subset. It is not a full Word layout engine.
package docx
