package docx

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
)

type paragraph struct {
	text  string
	style string
	list  bool
}

var orderedListStartRE = regexp.MustCompile(`^\s*\d+[\.)]\s+`)

// FileToMarkdown extracts readable text from a DOCX file and emits a small
// Markdown subset that can be passed to the renderer.
func FileToMarkdown(path string) (string, error) {
	reader, err := zip.OpenReader(path)
	if err != nil {
		return "", fmt.Errorf("open docx: %w", err)
	}
	defer reader.Close()

	for _, file := range reader.File {
		if file.Name != "word/document.xml" {
			continue
		}
		body, err := readZipFile(file)
		if err != nil {
			return "", err
		}
		paragraphs, err := parseDocumentXML(body)
		if err != nil {
			return "", err
		}
		return paragraphsToMarkdown(paragraphs), nil
	}

	return "", fmt.Errorf("docx is missing word/document.xml")
}

func readZipFile(file *zip.File) ([]byte, error) {
	reader, err := file.Open()
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", file.Name, err)
	}
	defer reader.Close()

	body, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", file.Name, err)
	}
	return body, nil
}

func parseDocumentXML(body []byte) ([]paragraph, error) {
	decoder := xml.NewDecoder(bytes.NewReader(body))
	var paragraphs []paragraph
	var current *paragraph
	var text strings.Builder
	inText := false

	// DOCX XML is namespaced, but encoding/xml exposes the local tag names. The
	// extractor only needs paragraph structure, paragraph style, list markers,
	// and text runs, so it streams tokens instead of building a full DOM.
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("parse word/document.xml: %w", err)
		}

		switch value := token.(type) {
		case xml.StartElement:
			switch value.Name.Local {
			case "p":
				current = &paragraph{}
				text.Reset()
			case "pStyle":
				if current != nil {
					current.style = attrValue(value.Attr, "val")
				}
			case "numPr":
				if current != nil {
					current.list = true
				}
			case "t":
				if current != nil {
					inText = true
				}
			case "tab":
				if current != nil {
					text.WriteByte('\t')
				}
			case "br", "cr":
				if current != nil {
					text.WriteByte('\n')
				}
			}
		case xml.CharData:
			if current != nil && inText {
				text.Write([]byte(value))
			}
		case xml.EndElement:
			switch value.Name.Local {
			case "t":
				inText = false
			case "p":
				if current != nil {
					current.text = normalizeText(text.String())
					paragraphs = append(paragraphs, *current)
				}
				current = nil
				inText = false
			}
		}
	}

	return paragraphs, nil
}

func attrValue(attrs []xml.Attr, localName string) string {
	for _, attr := range attrs {
		if attr.Name.Local == localName {
			return attr.Value
		}
	}
	return ""
}

func paragraphsToMarkdown(paragraphs []paragraph) string {
	var out strings.Builder
	for _, paragraph := range paragraphs {
		text := paragraph.text
		if text == "" {
			continue
		}

		// Markdown is the handoff format to the existing renderer, so each DOCX
		// paragraph is reduced to the closest supported block type.
		if level := headingLevel(paragraph.style); level > 0 {
			out.WriteString(strings.Repeat("#", level))
			out.WriteByte(' ')
			out.WriteString(escapeMarkdownBlockText(text))
			out.WriteString("\n\n")
			continue
		}

		if paragraph.list {
			out.WriteString("- ")
			out.WriteString(escapeMarkdownBlockText(text))
			out.WriteString("\n")
			continue
		}

		out.WriteString(escapeMarkdownBlockText(text))
		out.WriteString("\n\n")
	}
	return strings.TrimSpace(out.String()) + "\n"
}

func headingLevel(style string) int {
	style = strings.ToLower(strings.ReplaceAll(style, " ", ""))
	if style == "title" {
		return 1
	}
	if !strings.HasPrefix(style, "heading") {
		return 0
	}
	level, err := strconv.Atoi(strings.TrimPrefix(style, "heading"))
	if err != nil || level < 1 || level > 6 {
		return 0
	}
	return level
}

func normalizeText(text string) string {
	text = strings.ReplaceAll(text, "\t", " ")
	text = strings.ReplaceAll(text, "\n", " ")
	return strings.Join(strings.Fields(text), " ")
}

func escapeMarkdownBlockText(text string) string {
	// DOCX text is literal prose. Escape leading Markdown markers so a paragraph
	// like "# Not a heading" does not accidentally become structure.
	trimmed := strings.TrimLeft(text, " \t")
	prefix := text[:len(text)-len(trimmed)]
	if trimmed == "" {
		return text
	}
	if strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, ">") {
		return prefix + `\` + trimmed
	}
	if strings.HasPrefix(trimmed, "- ") || strings.HasPrefix(trimmed, "* ") || strings.HasPrefix(trimmed, "+ ") {
		return prefix + `\` + trimmed
	}
	if orderedListStartRE.MatchString(trimmed) {
		return prefix + `\` + trimmed
	}
	if isThematicBreakText(trimmed) {
		return prefix + `\` + trimmed
	}
	return text
}

func isThematicBreakText(text string) bool {
	if len(text) < 3 {
		return false
	}
	first := text[0]
	if first != '-' && first != '*' && first != '_' {
		return false
	}
	for i := 1; i < len(text); i++ {
		if text[i] != first {
			return false
		}
	}
	return true
}
