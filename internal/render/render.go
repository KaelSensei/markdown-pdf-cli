package render

import (
	"fmt"
	"math"
	"strings"
	"unicode"

	"github.com/KaelSensei/markdown-pdf-cli/internal/markdown"
	"github.com/KaelSensei/markdown-pdf-cli/internal/pdf"
)

// Options configures one Markdown-to-PDF conversion.
type Options struct {
	// Title is written into PDF metadata. A default title is used when empty.
	Title string
	// PageSize accepts "a4" or "letter".
	PageSize string
	// Margin is measured in PDF points. When zero, a readable default is used.
	Margin float64
	// Theme accepts "modern", "classic", or "elegant".
	Theme string
	// ColorScheme accepts "light" or "dark".
	ColorScheme string
}

type theme struct {
	name                  string
	bodyFont              string
	boldFont              string
	italicFont            string
	monoFont              string
	headingFont           string
	bodySize              float64
	codeSize              float64
	tableSize             float64
	lineHeight            float64
	headingLineHeight     float64
	blockGap              float64
	topBarHeight          float64
	centerH1              bool
	headingSizes          map[int]float64
	background            pdf.RGB
	body                  pdf.RGB
	heading               pdf.RGB
	muted                 pdf.RGB
	accent                pdf.RGB
	rule                  pdf.RGB
	codeBackground        pdf.RGB
	quoteBackground       pdf.RGB
	tableHeaderBackground pdf.RGB
	tableBorder           pdf.RGB
	tableHeaderText       pdf.RGB
}

type renderer struct {
	doc          *pdf.Document
	page         *pdf.Page
	size         pdf.Size
	margin       float64
	contentWidth float64
	cursor       float64
	theme        theme
}

// MarkdownToPDF converts Markdown source text into a complete PDF file.
//
// The function is pure from the caller's perspective: it does not read files,
// write files, or call the network. All IO is handled by the CLI layer.
func MarkdownToPDF(source string, opts Options) ([]byte, error) {
	size, err := resolvePageSize(opts.PageSize)
	if err != nil {
		return nil, err
	}
	if opts.Margin == 0 {
		opts.Margin = 56
	}
	if opts.Margin < 24 {
		return nil, fmt.Errorf("margin must be at least 24 points")
	}
	if opts.Margin*2 >= size.Width || opts.Margin*2 >= size.Height {
		return nil, fmt.Errorf("margin is too large for the selected page size")
	}

	t, err := resolveTheme(opts.Theme, opts.ColorScheme)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(opts.Title) == "" {
		opts.Title = "Markdown PDF"
	}

	doc := pdf.New(size, opts.Title)
	r := renderer{
		doc:          doc,
		size:         size,
		margin:       opts.Margin,
		contentWidth: size.Width - opts.Margin*2,
		theme:        t,
	}
	r.newPage()

	parsed := markdown.Parse(source)
	for _, block := range parsed.Blocks {
		r.renderBlock(block)
	}

	return doc.Bytes(), nil
}

func resolvePageSize(name string) (pdf.Size, error) {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "", "a4":
		return pdf.Size{Width: 595.28, Height: 841.89}, nil
	case "letter":
		return pdf.Size{Width: 612, Height: 792}, nil
	default:
		return pdf.Size{}, fmt.Errorf("unsupported page size %q: use a4 or letter", name)
	}
}

func resolveTheme(name, scheme string) (theme, error) {
	name = strings.ToLower(strings.TrimSpace(name))
	if name == "" {
		name = "modern"
	}
	scheme = strings.ToLower(strings.TrimSpace(scheme))
	if scheme == "" {
		scheme = "light"
	}
	if scheme != "light" && scheme != "dark" {
		return theme{}, fmt.Errorf("unsupported color scheme %q: use light or dark", scheme)
	}

	var t theme
	switch name {
	case "modern":
		t = modernTheme()
	case "classic":
		t = classicTheme()
	case "elegant":
		t = elegantTheme()
	default:
		return theme{}, fmt.Errorf("unsupported theme %q: use modern, classic, or elegant", name)
	}
	if scheme == "dark" {
		applyDarkScheme(&t)
	}
	return t, nil
}

func modernTheme() theme {
	return theme{
		name:              "modern",
		bodyFont:          "F1",
		boldFont:          "F2",
		italicFont:        "F5",
		monoFont:          "F3",
		headingFont:       "F2",
		bodySize:          10.8,
		codeSize:          9.2,
		tableSize:         9.4,
		lineHeight:        1.42,
		headingLineHeight: 1.2,
		blockGap:          8,
		topBarHeight:      5,
		headingSizes: map[int]float64{
			1: 27, 2: 20, 3: 16, 4: 13.5, 5: 12, 6: 11,
		},
		background:            rgb(1, 1, 1),
		body:                  rgb(0.13, 0.15, 0.18),
		heading:               rgb(0.05, 0.08, 0.12),
		muted:                 rgb(0.38, 0.43, 0.50),
		accent:                rgb(0.04, 0.36, 0.72),
		rule:                  rgb(0.82, 0.86, 0.90),
		codeBackground:        rgb(0.95, 0.97, 0.99),
		quoteBackground:       rgb(0.96, 0.98, 1.00),
		tableHeaderBackground: rgb(0.91, 0.95, 0.99),
		tableBorder:           rgb(0.78, 0.84, 0.91),
		tableHeaderText:       rgb(0.06, 0.12, 0.20),
	}
}

func classicTheme() theme {
	return theme{
		name:              "classic",
		bodyFont:          "F6",
		boldFont:          "F7",
		italicFont:        "F8",
		monoFont:          "F3",
		headingFont:       "F7",
		bodySize:          11.2,
		codeSize:          9.2,
		tableSize:         9.6,
		lineHeight:        1.48,
		headingLineHeight: 1.18,
		blockGap:          9,
		headingSizes: map[int]float64{
			1: 28, 2: 21, 3: 16.5, 4: 14, 5: 12.3, 6: 11.2,
		},
		background:            rgb(1.00, 0.995, 0.975),
		body:                  rgb(0.12, 0.10, 0.08),
		heading:               rgb(0.10, 0.07, 0.05),
		muted:                 rgb(0.43, 0.38, 0.32),
		accent:                rgb(0.48, 0.14, 0.12),
		rule:                  rgb(0.76, 0.69, 0.60),
		codeBackground:        rgb(0.965, 0.95, 0.925),
		quoteBackground:       rgb(0.972, 0.955, 0.928),
		tableHeaderBackground: rgb(0.90, 0.86, 0.79),
		tableBorder:           rgb(0.72, 0.65, 0.56),
		tableHeaderText:       rgb(0.12, 0.08, 0.05),
	}
}

func elegantTheme() theme {
	return theme{
		name:              "elegant",
		bodyFont:          "F6",
		boldFont:          "F7",
		italicFont:        "F8",
		monoFont:          "F3",
		headingFont:       "F7",
		bodySize:          11,
		codeSize:          9.1,
		tableSize:         9.4,
		lineHeight:        1.5,
		headingLineHeight: 1.16,
		blockGap:          10,
		topBarHeight:      7,
		centerH1:          true,
		headingSizes: map[int]float64{
			1: 30, 2: 21, 3: 16, 4: 13.5, 5: 12, 6: 11,
		},
		background:            rgb(0.995, 0.995, 0.985),
		body:                  rgb(0.11, 0.13, 0.12),
		heading:               rgb(0.05, 0.12, 0.10),
		muted:                 rgb(0.39, 0.43, 0.39),
		accent:                rgb(0.58, 0.42, 0.16),
		rule:                  rgb(0.78, 0.72, 0.60),
		codeBackground:        rgb(0.95, 0.955, 0.94),
		quoteBackground:       rgb(0.96, 0.95, 0.92),
		tableHeaderBackground: rgb(0.88, 0.84, 0.74),
		tableBorder:           rgb(0.72, 0.68, 0.57),
		tableHeaderText:       rgb(0.07, 0.12, 0.10),
	}
}

func applyDarkScheme(t *theme) {
	switch t.name {
	case "classic":
		t.background = rgb(0.075, 0.067, 0.058)
		t.body = rgb(0.88, 0.84, 0.76)
		t.heading = rgb(0.98, 0.93, 0.82)
		t.muted = rgb(0.68, 0.62, 0.52)
		t.accent = rgb(0.78, 0.36, 0.28)
		t.rule = rgb(0.36, 0.31, 0.25)
		t.codeBackground = rgb(0.13, 0.115, 0.10)
		t.quoteBackground = rgb(0.12, 0.105, 0.09)
		t.tableHeaderBackground = rgb(0.20, 0.16, 0.13)
		t.tableBorder = rgb(0.35, 0.30, 0.24)
		t.tableHeaderText = rgb(0.98, 0.91, 0.78)
	case "elegant":
		t.background = rgb(0.055, 0.070, 0.066)
		t.body = rgb(0.87, 0.90, 0.84)
		t.heading = rgb(0.96, 0.95, 0.86)
		t.muted = rgb(0.67, 0.71, 0.64)
		t.accent = rgb(0.75, 0.58, 0.27)
		t.rule = rgb(0.30, 0.33, 0.28)
		t.codeBackground = rgb(0.09, 0.105, 0.10)
		t.quoteBackground = rgb(0.085, 0.105, 0.098)
		t.tableHeaderBackground = rgb(0.16, 0.18, 0.15)
		t.tableBorder = rgb(0.30, 0.34, 0.29)
		t.tableHeaderText = rgb(0.96, 0.91, 0.75)
	default:
		t.background = rgb(0.075, 0.085, 0.105)
		t.body = rgb(0.86, 0.89, 0.93)
		t.heading = rgb(0.97, 0.985, 1.00)
		t.muted = rgb(0.62, 0.68, 0.76)
		t.accent = rgb(0.40, 0.68, 0.98)
		t.rule = rgb(0.26, 0.31, 0.38)
		t.codeBackground = rgb(0.11, 0.13, 0.16)
		t.quoteBackground = rgb(0.10, 0.125, 0.16)
		t.tableHeaderBackground = rgb(0.14, 0.18, 0.24)
		t.tableBorder = rgb(0.28, 0.34, 0.43)
		t.tableHeaderText = rgb(0.94, 0.97, 1.00)
	}
}

func (r *renderer) newPage() {
	r.page = r.doc.AddPage()
	r.page.FillRect(0, 0, r.size.Width, r.size.Height, r.theme.background)
	if r.theme.topBarHeight > 0 {
		r.page.FillRect(0, 0, r.size.Width, r.theme.topBarHeight, r.theme.accent)
	}
	r.cursor = r.margin
}

func (r *renderer) renderBlock(block markdown.Block) {
	switch block.Type {
	case markdown.Heading:
		r.renderHeading(block)
	case markdown.Paragraph:
		r.renderParagraph(block.Text)
	case markdown.List:
		r.renderList(block.Items)
	case markdown.CodeBlock:
		r.renderCode(block.Lines)
	case markdown.Blockquote:
		r.renderQuote(block.Lines)
	case markdown.ThematicBreak:
		r.renderRule()
	case markdown.Table:
		r.renderTable(block)
	}
}

func (r *renderer) renderHeading(block markdown.Block) {
	text := strings.TrimSpace(block.Text)
	if text == "" {
		return
	}
	level := block.Level
	if level < 1 || level > 6 {
		level = 6
	}
	size := r.theme.headingSizes[level]
	lineHeight := size * r.theme.headingLineHeight
	before := 8.0
	if level == 1 {
		before = 0
	} else if level >= 4 {
		before = 6
	}
	r.addSpace(before)

	lines := wrapText(text, r.contentWidth, r.theme.headingFont, size)
	for _, line := range lines {
		r.ensure(lineHeight)
		x := r.margin
		if r.theme.centerH1 && level == 1 {
			x = r.margin + math.Max(0, (r.contentWidth-textWidth(line, r.theme.headingFont, size))/2)
		}
		r.drawTextLine(x, r.cursor, size, lineHeight, r.theme.headingFont, line, r.theme.heading)
		r.cursor += lineHeight
	}

	if level == 1 {
		r.cursor += 6
		r.ensure(8)
		lineWidth := 128.0
		x1 := r.margin
		if r.theme.centerH1 {
			x1 = r.margin + (r.contentWidth-lineWidth)/2
		}
		r.page.Line(x1, r.cursor, x1+lineWidth, r.cursor, 1.2, r.theme.accent)
		r.cursor += 14
		return
	}
	r.cursor += 4
}

func (r *renderer) renderParagraph(text string) {
	text = strings.TrimSpace(text)
	if text == "" {
		return
	}
	lines := wrapText(text, r.contentWidth, r.theme.bodyFont, r.theme.bodySize)
	r.drawWrappedLines(lines, r.margin, r.theme.bodySize, r.theme.bodySize*r.theme.lineHeight, r.theme.bodyFont, r.theme.body)
	r.cursor += r.theme.blockGap
}

func (r *renderer) renderList(items []markdown.ListItem) {
	if len(items) == 0 {
		return
	}
	lineHeight := r.theme.bodySize * r.theme.lineHeight
	for index, item := range items {
		indent := math.Min(float64(item.Level)*18, r.contentWidth/3)
		markerWidth := 24.0
		marker := "•"
		if item.Ordered {
			number := item.Number
			if number == 0 {
				number = index + 1
			}
			marker = fmt.Sprintf("%d.", number)
		}
		textX := r.margin + indent + markerWidth
		maxWidth := r.contentWidth - indent - markerWidth
		lines := wrapText(item.Text, maxWidth, r.theme.bodyFont, r.theme.bodySize)
		if len(lines) == 0 {
			lines = []string{""}
		}
		for lineIndex, line := range lines {
			r.ensure(lineHeight)
			if lineIndex == 0 {
				r.drawTextLine(r.margin+indent, r.cursor, r.theme.bodySize, lineHeight, r.theme.bodyFont, marker, r.theme.accent)
			}
			r.drawTextLine(textX, r.cursor, r.theme.bodySize, lineHeight, r.theme.bodyFont, line, r.theme.body)
			r.cursor += lineHeight
		}
		r.cursor += 2
	}
	r.cursor += r.theme.blockGap - 2
}

func (r *renderer) renderCode(lines []string) {
	if len(lines) == 0 {
		return
	}
	r.addSpace(3)
	lineHeight := r.theme.codeSize * 1.52
	padX := 9.0
	for _, raw := range lines {
		wrapped := wrapCodeLine(raw, r.contentWidth-padX*2, r.theme.monoFont, r.theme.codeSize)
		if len(wrapped) == 0 {
			wrapped = []string{""}
		}
		for _, line := range wrapped {
			r.ensure(lineHeight)
			r.page.FillRect(r.margin, r.cursor, r.contentWidth, lineHeight, r.theme.codeBackground)
			r.page.FillRect(r.margin, r.cursor, 3, lineHeight, r.theme.accent)
			r.drawTextLine(r.margin+padX, r.cursor, r.theme.codeSize, lineHeight, r.theme.monoFont, line, r.theme.body)
			r.cursor += lineHeight
		}
	}
	r.cursor += r.theme.blockGap
}

func (r *renderer) renderQuote(lines []string) {
	if len(lines) == 0 {
		return
	}
	r.addSpace(3)
	lineHeight := r.theme.bodySize * r.theme.lineHeight
	x := r.margin + 14
	maxWidth := r.contentWidth - 22
	for _, raw := range lines {
		wrapped := wrapText(raw, maxWidth, r.theme.italicFont, r.theme.bodySize)
		if len(wrapped) == 0 {
			wrapped = []string{""}
		}
		for _, line := range wrapped {
			r.ensure(lineHeight)
			r.page.FillRect(r.margin, r.cursor, r.contentWidth, lineHeight, r.theme.quoteBackground)
			r.page.FillRect(r.margin, r.cursor, 4, lineHeight, r.theme.accent)
			r.drawTextLine(x, r.cursor, r.theme.bodySize, lineHeight, r.theme.italicFont, line, r.theme.muted)
			r.cursor += lineHeight
		}
	}
	r.cursor += r.theme.blockGap
}

func (r *renderer) renderRule() {
	r.addSpace(6)
	r.ensure(12)
	r.page.Line(r.margin, r.cursor+4, r.margin+r.contentWidth, r.cursor+4, 0.8, r.theme.rule)
	r.cursor += 18
}

func (r *renderer) renderTable(block markdown.Block) {
	cols := len(block.Header)
	for _, row := range block.Rows {
		if len(row) > cols {
			cols = len(row)
		}
	}
	if cols == 0 {
		return
	}

	r.addSpace(4)
	colWidth := r.contentWidth / float64(cols)
	lineHeight := r.theme.tableSize * 1.35
	rows := make([][]string, 0, len(block.Rows)+1)
	rows = append(rows, block.Header)
	rows = append(rows, block.Rows...)

	for rowIndex, row := range rows {
		isHeader := rowIndex == 0
		font := r.theme.bodyFont
		color := r.theme.body
		if isHeader {
			font = r.theme.boldFont
			color = r.theme.tableHeaderText
		}

		cellLines := make([][]string, cols)
		maxLines := 1
		for col := 0; col < cols; col++ {
			text := ""
			if col < len(row) {
				text = row[col]
			}
			lines := wrapText(text, colWidth-10, font, r.theme.tableSize)
			if len(lines) == 0 {
				lines = []string{""}
			}
			cellLines[col] = lines
			if len(lines) > maxLines {
				maxLines = len(lines)
			}
		}

		// Tables are paginated at row boundaries. Individual rows can grow taller
		// when cell text wraps, but a row is never split across two pages.
		rowHeight := float64(maxLines)*lineHeight + 8
		r.ensure(rowHeight)
		y := r.cursor
		if isHeader {
			r.page.FillRect(r.margin, y, r.contentWidth, rowHeight, r.theme.tableHeaderBackground)
		}
		for col := 0; col < cols; col++ {
			x := r.margin + float64(col)*colWidth
			r.page.StrokeRect(x, y, colWidth, rowHeight, 0.4, r.theme.tableBorder)
			for lineIndex, line := range cellLines[col] {
				lineY := y + 4 + float64(lineIndex)*lineHeight
				r.drawTextLine(x+5, lineY, r.theme.tableSize, lineHeight, font, line, color)
			}
		}
		r.cursor += rowHeight
	}
	r.cursor += r.theme.blockGap
}

func (r *renderer) drawWrappedLines(lines []string, x, size, lineHeight float64, font string, color pdf.RGB) {
	for _, line := range lines {
		r.ensure(lineHeight)
		r.drawTextLine(x, r.cursor, size, lineHeight, font, line, color)
		r.cursor += lineHeight
	}
}

func (r *renderer) drawTextLine(x, y, size, lineHeight float64, font, text string, color pdf.RGB) {
	// Page layout tracks the top of a line box. PDF text drawing expects a
	// baseline, so the text is nudged down into the box before drawing.
	baseline := y + size + (lineHeight-size)*0.45
	r.page.TextColor(x, baseline, size, font, text, color)
}

func (r *renderer) addSpace(amount float64) {
	if amount <= 0 || r.cursor <= r.margin {
		return
	}
	r.cursor += amount
	if r.cursor > r.size.Height-r.margin {
		r.newPage()
	}
}

func (r *renderer) ensure(height float64) {
	if r.cursor+height <= r.size.Height-r.margin {
		return
	}
	if r.cursor > r.margin {
		r.newPage()
	}
}

func wrapText(text string, maxWidth float64, font string, size float64) []string {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}
	words := strings.Fields(text)
	var lines []string
	current := ""
	for _, word := range words {
		for _, part := range splitLongToken(word, maxWidth, font, size) {
			if current == "" {
				current = part
				continue
			}
			candidate := current + " " + part
			if textWidth(candidate, font, size) <= maxWidth {
				current = candidate
				continue
			}
			lines = append(lines, current)
			current = part
		}
	}
	if current != "" {
		lines = append(lines, current)
	}
	return lines
}

func wrapCodeLine(text string, maxWidth float64, font string, size float64) []string {
	text = strings.ReplaceAll(text, "\t", "    ")
	if text == "" {
		return []string{""}
	}
	var lines []string
	var current strings.Builder
	for _, r := range text {
		candidate := current.String() + string(r)
		if current.Len() > 0 && textWidth(candidate, font, size) > maxWidth {
			lines = append(lines, current.String())
			current.Reset()
		}
		current.WriteRune(r)
	}
	if current.Len() > 0 {
		lines = append(lines, current.String())
	}
	return lines
}

func splitLongToken(token string, maxWidth float64, font string, size float64) []string {
	if textWidth(token, font, size) <= maxWidth {
		return []string{token}
	}
	var parts []string
	var current strings.Builder
	for _, r := range token {
		candidate := current.String() + string(r)
		if current.Len() > 0 && textWidth(candidate, font, size) > maxWidth {
			parts = append(parts, current.String())
			current.Reset()
		}
		current.WriteRune(r)
	}
	if current.Len() > 0 {
		parts = append(parts, current.String())
	}
	return parts
}

func textWidth(text string, font string, size float64) float64 {
	width := 0.0
	for _, r := range text {
		width += glyphWidth(r, font) * size
	}
	return width
}

func glyphWidth(r rune, font string) float64 {
	// This approximation keeps wrapping deterministic without a font metrics
	// dependency. It is tuned for built-in Helvetica, Times, and Courier output.
	if strings.HasPrefix(font, "F3") || strings.HasPrefix(font, "F4") {
		return 0.60
	}
	switch {
	case r == ' ':
		return 0.27
	case r == '\t':
		return 1.08
	case strings.ContainsRune(".,:;!|'`", r):
		return 0.24
	case strings.ContainsRune("()[]{}\"-_/\\", r):
		return 0.34
	case strings.ContainsRune("ilI", r):
		return 0.25
	case strings.ContainsRune("mwMW", r):
		return 0.82
	case unicode.IsUpper(r):
		return 0.66
	case unicode.IsDigit(r):
		return 0.52
	case r > 255:
		return 0.88
	default:
		return 0.50
	}
}

func rgb(r, g, b float64) pdf.RGB {
	return pdf.RGB{R: r, G: g, B: b}
}
