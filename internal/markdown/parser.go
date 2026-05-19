package markdown

import (
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

// BlockType identifies the semantic kind of a parsed Markdown block.
type BlockType int

const (
	// Paragraph is a run of inline text with no stronger block semantics.
	Paragraph BlockType = iota
	// Heading is an ATX heading, from level 1 through 6.
	Heading
	// List is an ordered or unordered list.
	List
	// CodeBlock is a fenced code block.
	CodeBlock
	// Blockquote is one or more quoted lines.
	Blockquote
	// ThematicBreak is a horizontal rule.
	ThematicBreak
	// Table is a simple pipe-delimited table.
	Table
)

// Document is the parser output consumed by the renderer.
type Document struct {
	Blocks []Block
}

// Block holds the fields needed by each supported block type.
//
// The parser keeps this structure flat to avoid over-modeling a deliberately
// small Markdown subset. Callers should read the fields that match Type.
type Block struct {
	Type   BlockType
	Level  int
	Text   string
	Lines  []string
	Items  []ListItem
	Header []string
	Rows   [][]string
}

// ListItem is one rendered item in an ordered or unordered list.
type ListItem struct {
	Text    string
	Level   int
	Ordered bool
	Number  int
}

var (
	imageRE    = regexp.MustCompile(`!\[([^\]]*)\]\(([^)]*)\)`)
	linkRE     = regexp.MustCompile(`\[([^\]]+)\]\(([^)]*)\)`)
	codeRE     = regexp.MustCompile("`([^`]*)`")
	htmlTagRE  = regexp.MustCompile(`<[^>]+>`)
	autolinkRE = regexp.MustCompile(`<((?:https?|mailto):[^>]+)>`)
)

// Parse converts Markdown source text into a sequence of renderable blocks.
//
// It normalizes line endings, skips YAML-style front matter, and recognizes the
// Markdown features supported by V1 of the CLI.
func Parse(input string) Document {
	input = strings.ReplaceAll(input, "\r\n", "\n")
	input = strings.ReplaceAll(input, "\r", "\n")
	lines := strings.Split(input, "\n")
	lines = trimTrailingEmptyLines(lines)

	var blocks []Block
	for i := 0; i < len(lines); {
		line := trimRight(lines[i])
		if strings.TrimSpace(line) == "" {
			i++
			continue
		}
		// Treat a top-of-file --- block as front matter, not as a visible rule.
		if i == 0 && strings.TrimSpace(line) == "---" {
			if next := findFrontMatterEnd(lines); next > 0 {
				i = next + 1
				continue
			}
		}
		if marker, ok := fenceMarker(line); ok {
			block, next := parseCodeBlock(lines, i, marker)
			blocks = append(blocks, block)
			i = next
			continue
		}
		if level, text, ok := parseHeading(line); ok {
			blocks = append(blocks, Block{Type: Heading, Level: level, Text: PlainInline(text)})
			i++
			continue
		}
		if isThematicBreak(line) {
			blocks = append(blocks, Block{Type: ThematicBreak})
			i++
			continue
		}
		if isTableStart(lines, i) {
			block, next := parseTable(lines, i)
			blocks = append(blocks, block)
			i = next
			continue
		}
		if _, ok := parseListMarker(line); ok {
			block, next := parseList(lines, i)
			blocks = append(blocks, block)
			i = next
			continue
		}
		if isQuoteLine(line) {
			block, next := parseBlockquote(lines, i)
			blocks = append(blocks, block)
			i = next
			continue
		}

		block, next := parseParagraph(lines, i)
		blocks = append(blocks, block)
		i = next
	}

	return Document{Blocks: blocks}
}

// PlainInline reduces inline Markdown to text that can be drawn in the PDF.
//
// V1 does not preserve inline styling spans. Links are expanded to
// "label (url)" and images are represented as "[image: alt text]" so the output
// remains readable and fully offline.
func PlainInline(text string) string {
	text = imageRE.ReplaceAllString(text, "[image: $1]")
	text = linkRE.ReplaceAllString(text, "$1 ($2)")
	text = autolinkRE.ReplaceAllString(text, "$1")
	text = codeRE.ReplaceAllString(text, "$1")
	text = htmlTagRE.ReplaceAllString(text, "")

	replacer := strings.NewReplacer(
		"**", "",
		"__", "",
		"~~", "",
		"*", "",
		"_", "",
		"\\[", "[",
		"\\]", "]",
		"\\(", "(",
		"\\)", ")",
		"\\`", "`",
		"\\*", "*",
		"\\_", "_",
	)
	text = replacer.Replace(text)
	return strings.Join(strings.Fields(text), " ")
}

func parseCodeBlock(lines []string, start int, marker string) (Block, int) {
	var body []string
	i := start + 1
	for i < len(lines) {
		line := trimRight(lines[i])
		if strings.HasPrefix(strings.TrimSpace(line), marker) {
			return Block{Type: CodeBlock, Lines: body}, i + 1
		}
		body = append(body, trimRight(line))
		i++
	}
	return Block{Type: CodeBlock, Lines: body}, i
}

func parseHeading(line string) (int, string, bool) {
	trimmed := strings.TrimSpace(line)
	level := 0
	for level < len(trimmed) && trimmed[level] == '#' {
		level++
	}
	if level == 0 || level > 6 {
		return 0, "", false
	}
	if level < len(trimmed) && !unicode.IsSpace(rune(trimmed[level])) {
		return 0, "", false
	}
	text := strings.TrimSpace(trimmed[level:])
	for strings.HasSuffix(text, "#") {
		text = strings.TrimSpace(strings.TrimSuffix(text, "#"))
	}
	return level, text, true
}

func parseParagraph(lines []string, start int) (Block, int) {
	var parts []string
	i := start
	for i < len(lines) {
		line := trimRight(lines[i])
		if strings.TrimSpace(line) == "" || isBlockStart(lines, i) {
			break
		}
		parts = append(parts, strings.TrimSpace(line))
		i++
	}
	return Block{Type: Paragraph, Text: PlainInline(strings.Join(parts, " "))}, i
}

func parseList(lines []string, start int) (Block, int) {
	var items []ListItem
	i := start
	for i < len(lines) {
		item, ok := parseListMarker(trimRight(lines[i]))
		if !ok {
			break
		}
		i++
		var parts []string
		if item.Text != "" {
			parts = append(parts, item.Text)
		}
		for i < len(lines) {
			line := trimRight(lines[i])
			if strings.TrimSpace(line) == "" {
				break
			}
			if _, ok := parseListMarker(line); ok || isBlockStart(lines, i) {
				break
			}
			parts = append(parts, strings.TrimSpace(line))
			i++
		}
		item.Text = PlainInline(strings.Join(parts, " "))
		items = append(items, item)
	}
	return Block{Type: List, Items: items}, i
}

func parseBlockquote(lines []string, start int) (Block, int) {
	var quoteLines []string
	i := start
	for i < len(lines) {
		line := trimRight(lines[i])
		if strings.TrimSpace(line) == "" {
			quoteLines = append(quoteLines, "")
			i++
			continue
		}
		if !isQuoteLine(line) {
			break
		}
		trimmed := strings.TrimSpace(line)
		trimmed = strings.TrimPrefix(trimmed, ">")
		trimmed = strings.TrimSpace(trimmed)
		quoteLines = append(quoteLines, PlainInline(trimmed))
		i++
	}
	return Block{Type: Blockquote, Lines: quoteLines}, i
}

func parseTable(lines []string, start int) (Block, int) {
	header := normalizeCells(splitPipeRow(lines[start]))
	i := start + 2
	var rows [][]string
	for i < len(lines) {
		line := trimRight(lines[i])
		if strings.TrimSpace(line) == "" || !strings.Contains(line, "|") {
			break
		}
		rows = append(rows, normalizeCells(splitPipeRow(line)))
		i++
	}
	return Block{Type: Table, Header: header, Rows: rows}, i
}

func isBlockStart(lines []string, index int) bool {
	line := trimRight(lines[index])
	if strings.TrimSpace(line) == "" {
		return true
	}
	if _, ok := fenceMarker(line); ok {
		return true
	}
	if _, _, ok := parseHeading(line); ok {
		return true
	}
	if isThematicBreak(line) || isQuoteLine(line) || isTableStart(lines, index) {
		return true
	}
	_, ok := parseListMarker(line)
	return ok
}

func parseListMarker(line string) (ListItem, bool) {
	indent, trimmed := splitIndent(line)
	if len(trimmed) >= 2 && strings.ContainsRune("-*+", rune(trimmed[0])) && unicode.IsSpace(rune(trimmed[1])) {
		return ListItem{Text: strings.TrimSpace(trimmed[2:]), Level: indent / 2}, true
	}
	digitEnd := 0
	for digitEnd < len(trimmed) && trimmed[digitEnd] >= '0' && trimmed[digitEnd] <= '9' {
		digitEnd++
	}
	if digitEnd == 0 || digitEnd+1 >= len(trimmed) {
		return ListItem{}, false
	}
	if trimmed[digitEnd] != '.' && trimmed[digitEnd] != ')' {
		return ListItem{}, false
	}
	if !unicode.IsSpace(rune(trimmed[digitEnd+1])) {
		return ListItem{}, false
	}
	number, _ := strconv.Atoi(trimmed[:digitEnd])
	return ListItem{
		Text:    strings.TrimSpace(trimmed[digitEnd+2:]),
		Level:   indent / 2,
		Ordered: true,
		Number:  number,
	}, true
}

func splitIndent(line string) (int, string) {
	indent := 0
	for len(line) > 0 {
		switch line[0] {
		case ' ':
			indent++
			line = line[1:]
		case '\t':
			indent += 4
			line = line[1:]
		default:
			return indent, line
		}
	}
	return indent, line
}

func fenceMarker(line string) (string, bool) {
	trimmed := strings.TrimSpace(line)
	if strings.HasPrefix(trimmed, "```") {
		return repeatedPrefix(trimmed, '`'), true
	}
	if strings.HasPrefix(trimmed, "~~~") {
		return repeatedPrefix(trimmed, '~'), true
	}
	return "", false
}

func repeatedPrefix(text string, char byte) string {
	n := 0
	for n < len(text) && text[n] == char {
		n++
	}
	return strings.Repeat(string(char), n)
}

func isQuoteLine(line string) bool {
	return strings.HasPrefix(strings.TrimSpace(line), ">")
}

func isThematicBreak(line string) bool {
	trimmed := strings.ReplaceAll(strings.TrimSpace(line), " ", "")
	if len(trimmed) < 3 {
		return false
	}
	first := trimmed[0]
	if first != '-' && first != '*' && first != '_' {
		return false
	}
	for i := 1; i < len(trimmed); i++ {
		if trimmed[i] != first {
			return false
		}
	}
	return true
}

func isTableStart(lines []string, index int) bool {
	if index+1 >= len(lines) || !strings.Contains(lines[index], "|") {
		return false
	}
	cells := splitPipeRow(lines[index+1])
	if len(cells) == 0 {
		return false
	}
	for _, cell := range cells {
		cell = strings.TrimSpace(cell)
		if len(cell) < 3 {
			return false
		}
		if strings.HasPrefix(cell, ":") {
			cell = cell[1:]
		}
		if strings.HasSuffix(cell, ":") {
			cell = cell[:len(cell)-1]
		}
		if len(cell) < 3 {
			return false
		}
		for _, r := range cell {
			if r != '-' {
				return false
			}
		}
	}
	return true
}

func splitPipeRow(line string) []string {
	line = strings.TrimSpace(line)
	if strings.HasPrefix(line, "|") {
		line = line[1:]
	}
	if strings.HasSuffix(line, "|") {
		line = line[:len(line)-1]
	}
	var cells []string
	var current strings.Builder
	escaped := false
	for _, r := range line {
		// Markdown tables allow escaped pipes inside cell text.
		if escaped {
			current.WriteRune(r)
			escaped = false
			continue
		}
		if r == '\\' {
			escaped = true
			continue
		}
		if r == '|' {
			cells = append(cells, current.String())
			current.Reset()
			continue
		}
		current.WriteRune(r)
	}
	cells = append(cells, current.String())
	return cells
}

func normalizeCells(cells []string) []string {
	out := make([]string, 0, len(cells))
	for _, cell := range cells {
		out = append(out, PlainInline(cell))
	}
	return out
}

func findFrontMatterEnd(lines []string) int {
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			return i
		}
	}
	return -1
}

func trimTrailingEmptyLines(lines []string) []string {
	for len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}

func trimRight(line string) string {
	return strings.TrimRight(line, " \t")
}
