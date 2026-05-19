package reverse

import (
	"fmt"
	"io"
	"math"
	"sort"
	"strconv"
	"strings"
	"unicode"

	pdfreader "github.com/ledongthuc/pdf"
)

// Options configures PDF-to-Markdown extraction.
type Options struct {
	// PreservePages inserts an HTML comment before each page after the first one.
	PreservePages bool
}

// PageText contains the extracted text lines for one PDF page.
type PageText struct {
	Number int
	Lines  []Line
}

// Line contains positioned text chunks that share the same visual baseline.
type Line struct {
	Y        float64
	FontSize float64
	Chunks   []Chunk
}

// Chunk is a positioned piece of extracted PDF text.
type Chunk struct {
	Text     string
	X        float64
	Y        float64
	W        float64
	Font     string
	FontSize float64
}

// PDFFileToMarkdown extracts text from a PDF file and converts it to Markdown.
func PDFFileToMarkdown(path string, opts Options) (string, error) {
	file, reader, err := pdfreader.Open(path)
	if err != nil {
		return "", fmt.Errorf("open PDF: %w", err)
	}
	defer file.Close()

	pages := make([]PageText, 0, reader.NumPage())
	for pageNumber := 1; pageNumber <= reader.NumPage(); pageNumber++ {
		content := reader.Page(pageNumber).Content()
		pages = append(pages, PageText{
			Number: pageNumber,
			Lines:  groupTextIntoLines(content.Text),
		})
	}

	markdown := MarkdownFromPages(pages, opts)
	if strings.TrimSpace(markdown) == "" {
		plain, err := reader.GetPlainText()
		if err != nil {
			return "", fmt.Errorf("extract plain text fallback: %w", err)
		}
		data, err := io.ReadAll(plain)
		if err != nil {
			return "", fmt.Errorf("read plain text fallback: %w", err)
		}
		markdown = normalizePlainText(string(data))
	}
	return markdown, nil
}

// MarkdownFromPages converts positioned page text into best-effort Markdown.
func MarkdownFromPages(pages []PageText, opts Options) string {
	var out []string
	var paragraph []string

	flushParagraph := func() {
		if len(paragraph) == 0 {
			return
		}
		out = append(out, strings.Join(paragraph, " "), "")
		paragraph = nil
	}

	for pageIndex, page := range pages {
		if opts.PreservePages && pageIndex > 0 {
			flushParagraph()
			out = append(out, fmt.Sprintf("<!-- page %d -->", page.Number), "")
		}

		lines := page.Lines
		for i := 0; i < len(lines); {
			line := lines[i]
			text := lineText(line)
			if text == "" {
				flushParagraph()
				i++
				continue
			}

			if group, next := collectTable(lines, i); len(group) > 0 {
				flushParagraph()
				out = append(out, renderTable(group)...)
				out = append(out, "")
				i = next
				continue
			}

			if level := headingLevel(line); level > 0 {
				flushParagraph()
				out = append(out, strings.Repeat("#", level)+" "+escapeHeading(text), "")
				i++
				continue
			}

			if isCodeLine(line) {
				flushParagraph()
				var code []string
				for i < len(lines) && isCodeLine(lines[i]) {
					code = append(code, lineTextPreserveSpacing(lines[i]))
					i++
				}
				out = append(out, "```")
				out = append(out, code...)
				out = append(out, "```", "")
				continue
			}

			if item, kind, ok := listItemMarkdownKind(line); ok {
				flushParagraph()
				for ok {
					out = append(out, item)
					i++
					if i >= len(lines) {
						break
					}
					var nextKind string
					item, nextKind, ok = listItemMarkdownKind(lines[i])
					if ok && nextKind != kind {
						break
					}
				}
				out = append(out, "")
				continue
			}

			if isQuoteLine(line) {
				flushParagraph()
				for i < len(lines) && isQuoteLine(lines[i]) {
					out = append(out, "> "+lineText(lines[i]))
					i++
				}
				out = append(out, "")
				continue
			}

			paragraph = append(paragraph, text)
			if i+1 >= len(lines) || paragraphBreak(line, lines[i+1]) {
				flushParagraph()
			}
			i++
		}
	}
	flushParagraph()

	return strings.TrimSpace(collapseBlankLines(out)) + "\n"
}

func groupTextIntoLines(texts []pdfreader.Text) []Line {
	chunks := make([]Chunk, 0, len(texts))
	for _, text := range texts {
		s := rawPDFText(text.S)
		if s == "" {
			continue
		}
		chunks = append(chunks, Chunk{
			Text:     s,
			X:        text.X,
			Y:        text.Y,
			W:        text.W,
			Font:     text.Font,
			FontSize: text.FontSize,
		})
	}
	sort.Slice(chunks, func(i, j int) bool {
		if math.Abs(chunks[i].Y-chunks[j].Y) > lineTolerance(chunks[i], chunks[j]) {
			return chunks[i].Y > chunks[j].Y
		}
		return chunks[i].X < chunks[j].X
	})

	var lines []Line
	for _, chunk := range chunks {
		if len(lines) == 0 {
			lines = append(lines, newLine(chunk))
			continue
		}
		last := &lines[len(lines)-1]
		if math.Abs(last.Y-chunk.Y) <= lineToleranceFromLine(*last, chunk) {
			last.Chunks = append(last.Chunks, chunk)
			if chunk.FontSize > last.FontSize {
				last.FontSize = chunk.FontSize
			}
			continue
		}
		lines = append(lines, newLine(chunk))
	}

	for i := range lines {
		sort.Slice(lines[i].Chunks, func(a, b int) bool {
			return lines[i].Chunks[a].X < lines[i].Chunks[b].X
		})
		lines[i].Chunks = mergeAdjacentChunks(lines[i].Chunks)
	}
	return lines
}

func newLine(chunk Chunk) Line {
	return Line{
		Y:        chunk.Y,
		FontSize: chunk.FontSize,
		Chunks:   []Chunk{chunk},
	}
}

func collectTable(lines []Line, start int) ([][]string, int) {
	if !isTableCandidate(lines[start]) {
		return nil, start
	}

	var rows [][]string
	i := start
	for i < len(lines) && isTableCandidate(lines[i]) {
		rows = append(rows, tableCells(lines[i]))
		i++
	}
	if len(rows) < 2 {
		return nil, start
	}
	return rows, i
}

func renderTable(rows [][]string) []string {
	cols := 0
	for _, row := range rows {
		if len(row) > cols {
			cols = len(row)
		}
	}
	if cols == 0 {
		return nil
	}

	var out []string
	out = append(out, renderTableRow(rows[0], cols))
	separator := make([]string, cols)
	for i := range separator {
		separator[i] = "---"
	}
	out = append(out, renderTableRow(separator, cols))
	for _, row := range rows[1:] {
		out = append(out, renderTableRow(row, cols))
	}
	return out
}

func renderTableRow(row []string, cols int) string {
	cells := make([]string, cols)
	for i := 0; i < cols; i++ {
		if i < len(row) {
			cells[i] = escapeTableCell(row[i])
		}
	}
	return "| " + strings.Join(cells, " | ") + " |"
}

func tableCells(line Line) []string {
	cells := make([]string, 0, len(line.Chunks))
	for _, chunk := range line.Chunks {
		text := cleanText(chunk.Text)
		if text != "" {
			cells = append(cells, text)
		}
	}
	return cells
}

func mergeAdjacentChunks(chunks []Chunk) []Chunk {
	if len(chunks) <= 1 {
		return chunks
	}

	var merged []Chunk
	current := chunks[0]
	for _, next := range chunks[1:] {
		if shouldStartNewChunk(current, next) {
			if text := cleanText(current.Text); text != "" {
				current.Text = text
				merged = append(merged, current)
			}
			current = next
			continue
		}
		current.Text = joinAdjacentText(current.Text, next.Text, current, next)
		current.W = math.Max(current.W, (next.X-current.X)+next.W)
		if next.FontSize > current.FontSize {
			current.FontSize = next.FontSize
		}
	}
	if text := cleanText(current.Text); text != "" {
		current.Text = text
		merged = append(merged, current)
	}
	return merged
}

func shouldStartNewChunk(current, next Chunk) bool {
	if current.Font != next.Font && cleanText(current.Text) != "" && cleanText(next.Text) != "" {
		return true
	}
	gap := next.X - (current.X + chunkWidth(current))
	threshold := math.Max(current.FontSize, next.FontSize) * 1.35
	return gap > threshold
}

func joinAdjacentText(left, right string, current, next Chunk) string {
	if strings.TrimSpace(right) == "" {
		if strings.HasSuffix(left, " ") {
			return left
		}
		return left + " "
	}
	if strings.TrimSpace(left) == "" {
		return right
	}
	gap := next.X - (current.X + chunkWidth(current))
	if gap > math.Max(current.FontSize, next.FontSize)*0.45 && !strings.HasSuffix(left, " ") {
		return left + " " + right
	}
	return left + right
}

func isTableCandidate(line Line) bool {
	if len(line.Chunks) < 2 || isCodeLine(line) || headingLevel(line) > 0 || isQuoteLine(line) {
		return false
	}
	_, isList := listItemMarkdown(line)
	return !isList
}

func headingLevel(line Line) int {
	if len(line.Chunks) == 0 || len(line.Chunks) > 1 {
		return 0
	}
	if !allChunksMatch(line, isBoldFont) {
		return 0
	}

	size := line.FontSize
	switch {
	case size >= 24:
		return 1
	case size >= 18:
		return 2
	case size >= 15:
		return 3
	case size >= 13:
		return 4
	default:
		return 0
	}
}

func isCodeLine(line Line) bool {
	return len(line.Chunks) > 0 && allChunksMatch(line, isMonoFont)
}

func isQuoteLine(line Line) bool {
	return len(line.Chunks) > 0 && allChunksMatch(line, isItalicFont)
}

func listItemMarkdown(line Line) (string, bool) {
	item, _, ok := listItemMarkdownKind(line)
	return item, ok
}

func listItemMarkdownKind(line Line) (string, string, bool) {
	text := lineText(line)
	if text == "" {
		return "", "", false
	}
	fields := strings.Fields(text)
	if len(fields) == 0 {
		return "", "", false
	}
	first := fields[0]
	rest := strings.TrimSpace(strings.TrimPrefix(text, first))
	if isBullet(first) && rest != "" {
		return "- " + rest, "ul", true
	}
	if number, ok := orderedMarker(first); ok && rest != "" {
		return strconv.Itoa(number) + ". " + rest, "ol", true
	}
	return "", "", false
}

func paragraphBreak(current, next Line) bool {
	if lineText(next) == "" || headingLevel(next) > 0 || isCodeLine(next) || isQuoteLine(next) {
		return true
	}
	if _, ok := listItemMarkdown(next); ok {
		return true
	}
	if isTableCandidate(next) {
		return true
	}
	gap := math.Abs(current.Y - next.Y)
	size := math.Max(current.FontSize, next.FontSize)
	return gap > size*1.8
}

func lineText(line Line) string {
	parts := make([]string, 0, len(line.Chunks))
	for _, chunk := range line.Chunks {
		if text := cleanText(chunk.Text); text != "" {
			parts = append(parts, text)
		}
	}
	return strings.Join(parts, " ")
}

func lineTextPreserveSpacing(line Line) string {
	return strings.TrimRight(lineText(line), " ")
}

func allChunksMatch(line Line, match func(string) bool) bool {
	for _, chunk := range line.Chunks {
		if !match(chunk.Font) {
			return false
		}
	}
	return true
}

func isBoldFont(font string) bool {
	return strings.Contains(strings.ToLower(font), "bold")
}

func isItalicFont(font string) bool {
	font = strings.ToLower(font)
	return strings.Contains(font, "italic") || strings.Contains(font, "oblique")
}

func isMonoFont(font string) bool {
	font = strings.ToLower(font)
	return strings.Contains(font, "courier") || strings.Contains(font, "mono")
}

func isBullet(marker string) bool {
	switch strings.TrimSpace(marker) {
	case "•", "·", "●", "▪", "-", "*", "+", "\u0095":
		return true
	default:
		return false
	}
}

func orderedMarker(marker string) (int, bool) {
	marker = strings.TrimSpace(marker)
	if len(marker) < 2 {
		return 0, false
	}
	last := marker[len(marker)-1]
	if last != '.' && last != ')' {
		return 0, false
	}
	number, err := strconv.Atoi(marker[:len(marker)-1])
	return number, err == nil
}

func escapeHeading(text string) string {
	return strings.TrimLeft(text, "# ")
}

func escapeTableCell(text string) string {
	return strings.ReplaceAll(text, "|", `\|`)
}

func cleanText(text string) string {
	text = strings.Map(func(r rune) rune {
		if r == '\u00a0' {
			return ' '
		}
		if unicode.IsControl(r) && r != '\n' && r != '\t' {
			return -1
		}
		return r
	}, text)
	return strings.Join(strings.Fields(text), " ")
}

func rawPDFText(text string) string {
	return strings.Map(func(r rune) rune {
		if r == '\u00a0' {
			return ' '
		}
		if unicode.IsControl(r) && r != '\n' && r != '\t' {
			return -1
		}
		return r
	}, text)
}

func normalizePlainText(text string) string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	var lines []string
	for _, line := range strings.Split(text, "\n") {
		line = cleanText(line)
		if line != "" {
			lines = append(lines, line)
		}
	}
	return strings.TrimSpace(strings.Join(lines, "\n\n")) + "\n"
}

func collapseBlankLines(lines []string) string {
	var out []string
	blank := false
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			if !blank && len(out) > 0 {
				out = append(out, "")
			}
			blank = true
			continue
		}
		out = append(out, line)
		blank = false
	}
	return strings.Join(out, "\n")
}

func lineTolerance(a, b Chunk) float64 {
	return math.Max(1.5, math.Max(a.FontSize, b.FontSize)*0.25)
}

func lineToleranceFromLine(line Line, chunk Chunk) float64 {
	return math.Max(1.5, math.Max(line.FontSize, chunk.FontSize)*0.25)
}

func chunkWidth(chunk Chunk) float64 {
	if chunk.W > 0 {
		return chunk.W
	}
	return float64(len([]rune(chunk.Text))) * chunk.FontSize * 0.5
}
