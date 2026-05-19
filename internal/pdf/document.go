package pdf

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
)

// Size stores page dimensions in PDF points.
type Size struct {
	Width  float64
	Height float64
}

// RGB stores color channels as PDF-compatible values from 0 to 1.
type RGB struct {
	R float64
	G float64
	B float64
}

// Document accumulates pages and serializes them into a PDF byte stream.
type Document struct {
	size  Size
	title string
	pages []*Page
}

// Page records PDF drawing commands for a single page content stream.
type Page struct {
	size    Size
	content bytes.Buffer
}

// New creates a PDF document with a fixed page size and title metadata.
func New(size Size, title string) *Document {
	return &Document{size: size, title: title}
}

// Size returns the configured document page size.
func (d *Document) Size() Size {
	return d.size
}

// AddPage appends a blank page and returns it for drawing.
func (d *Document) AddPage() *Page {
	page := &Page{size: d.size}
	d.pages = append(d.pages, page)
	return page
}

// Text draws black text at the given page coordinates.
func (p *Page) Text(x, y, size float64, fontName string, text string) {
	p.TextColor(x, y, size, fontName, text, RGB{R: 0, G: 0, B: 0})
}

// TextColor draws colored text at the given page coordinates.
//
// Public drawing methods use top-left coordinates because that matches the
// renderer's layout model. The PDF stream stores bottom-left coordinates, so the
// Y value is flipped before writing the command.
func (p *Page) TextColor(x, y, size float64, fontName string, text string, color RGB) {
	if text == "" {
		return
	}
	fmt.Fprintf(&p.content, "%.3f %.3f %.3f rg BT /%s %.2f Tf %.2f %.2f Td %s Tj ET\n",
		color.R,
		color.G,
		color.B,
		fontName,
		size,
		x,
		p.size.Height-y,
		textLiteral(text),
	)
}

// Line draws a stroked line between two page coordinates.
func (p *Page) Line(x1, y1, x2, y2, width float64, color RGB) {
	fmt.Fprintf(&p.content, "%.3f %.3f %.3f RG %.2f w %.2f %.2f m %.2f %.2f l S\n",
		color.R, color.G, color.B,
		width,
		x1, p.size.Height-y1,
		x2, p.size.Height-y2,
	)
}

// FillRect draws a filled rectangle using top-left page coordinates.
func (p *Page) FillRect(x, y, width, height float64, color RGB) {
	fmt.Fprintf(&p.content, "%.3f %.3f %.3f rg %.2f %.2f %.2f %.2f re f\n",
		color.R, color.G, color.B,
		x, p.size.Height-y-height,
		width, height,
	)
}

// StrokeRect draws a rectangle outline using top-left page coordinates.
func (p *Page) StrokeRect(x, y, width, height, strokeWidth float64, color RGB) {
	fmt.Fprintf(&p.content, "%.3f %.3f %.3f RG %.2f w %.2f %.2f %.2f %.2f re S\n",
		color.R, color.G, color.B,
		strokeWidth,
		x, p.size.Height-y-height,
		width, height,
	)
}

// Bytes serializes the document into a complete PDF 1.4 file.
func (d *Document) Bytes() []byte {
	if len(d.pages) == 0 {
		d.AddPage()
	}

	// Object 1 is the catalog and object 2 is the page tree. Their bodies depend
	// on page IDs, so placeholders keep numbering stable while pages are built.
	objects := []string{"", ""}
	addObject := func(body string) int {
		objects = append(objects, body)
		return len(objects)
	}

	fonts := map[string]string{
		"F1": "Helvetica",
		"F2": "Helvetica-Bold",
		"F3": "Courier",
		"F4": "Courier-Bold",
		"F5": "Helvetica-Oblique",
		"F6": "Times-Roman",
		"F7": "Times-Bold",
		"F8": "Times-Italic",
	}
	fontNames := make([]string, 0, len(fonts))
	for name := range fonts {
		fontNames = append(fontNames, name)
	}
	sort.Strings(fontNames)

	fontIDs := make(map[string]int, len(fontNames))
	for _, name := range fontNames {
		fontIDs[name] = addObject(fmt.Sprintf(
			"<< /Type /Font /Subtype /Type1 /BaseFont /%s /Encoding /WinAnsiEncoding >>",
			fonts[name],
		))
	}

	var pageIDs []int
	for _, page := range d.pages {
		content := page.content.String()
		contentID := addObject(fmt.Sprintf("<< /Length %d >>\nstream\n%s\nendstream", len([]byte(content)), content))

		fontResource := strings.Builder{}
		for _, name := range fontNames {
			fmt.Fprintf(&fontResource, "/%s %d 0 R ", name, fontIDs[name])
		}

		pageID := addObject(fmt.Sprintf(
			"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 %.2f %.2f] /Resources << /Font << %s>> >> /Contents %d 0 R >>",
			d.size.Width,
			d.size.Height,
			fontResource.String(),
			contentID,
		))
		pageIDs = append(pageIDs, pageID)
	}

	kids := strings.Builder{}
	for _, id := range pageIDs {
		fmt.Fprintf(&kids, "%d 0 R ", id)
	}
	objects[0] = "<< /Type /Catalog /Pages 2 0 R >>"
	objects[1] = fmt.Sprintf("<< /Type /Pages /Kids [ %s] /Count %d >>", kids.String(), len(pageIDs))

	infoID := addObject(fmt.Sprintf("<< /Title %s /Producer %s >>",
		textLiteral(d.title),
		textLiteral("mdpdf"),
	))

	var out bytes.Buffer
	out.WriteString("%PDF-1.4\n%\xE2\xE3\xCF\xD3\n")

	// The xref table needs byte offsets for every object, so offsets are captured
	// immediately before each object is written.
	offsets := make([]int, len(objects)+1)
	for i, object := range objects {
		id := i + 1
		offsets[id] = out.Len()
		fmt.Fprintf(&out, "%d 0 obj\n%s\nendobj\n", id, object)
	}

	xrefOffset := out.Len()
	fmt.Fprintf(&out, "xref\n0 %d\n", len(objects)+1)
	out.WriteString("0000000000 65535 f \n")
	for id := 1; id <= len(objects); id++ {
		fmt.Fprintf(&out, "%010d 00000 n \n", offsets[id])
	}
	fmt.Fprintf(&out, "trailer\n<< /Size %d /Root 1 0 R /Info %d 0 R >>\nstartxref\n%d\n%%%%EOF\n",
		len(objects)+1,
		infoID,
		xrefOffset,
	)

	return out.Bytes()
}

func textLiteral(text string) string {
	encoded := encodeWinAnsi(text)
	var out bytes.Buffer
	out.WriteByte('(')
	for _, b := range encoded {
		switch b {
		case '\\', '(', ')':
			out.WriteByte('\\')
			out.WriteByte(b)
		case '\n':
			out.WriteString(`\n`)
		case '\r':
			out.WriteString(`\r`)
		case '\t':
			out.WriteByte(' ')
		default:
			if b < 0x20 || b == 0x7f {
				fmt.Fprintf(&out, `\%03o`, b)
				continue
			}
			out.WriteByte(b)
		}
	}
	out.WriteByte(')')
	return out.String()
}

func encodeWinAnsi(text string) []byte {
	var out []byte
	for _, r := range text {
		switch {
		case r >= 0x20 && r <= 0x7e:
			out = append(out, byte(r))
		case r >= 0xa0 && r <= 0xff:
			out = append(out, byte(r))
		case r == '\t' || r == '\n' || r == '\r':
			out = append(out, byte(r))
		default:
			if b, ok := winAnsiSpecials[r]; ok {
				out = append(out, b)
			} else {
				out = append(out, '?')
			}
		}
	}
	return out
}

// winAnsiSpecials maps printable Unicode punctuation to WinAnsi bytes supported
// by built-in PDF fonts. Unknown runes fall back to '?' until embedded Unicode
// fonts are added.
var winAnsiSpecials = map[rune]byte{
	'€': 0x80,
	'‚': 0x82,
	'ƒ': 0x83,
	'„': 0x84,
	'…': 0x85,
	'†': 0x86,
	'‡': 0x87,
	'ˆ': 0x88,
	'‰': 0x89,
	'Š': 0x8a,
	'‹': 0x8b,
	'Œ': 0x8c,
	'Ž': 0x8e,
	'‘': 0x91,
	'’': 0x92,
	'“': 0x93,
	'”': 0x94,
	'•': 0x95,
	'–': 0x96,
	'—': 0x97,
	'˜': 0x98,
	'™': 0x99,
	'š': 0x9a,
	'›': 0x9b,
	'œ': 0x9c,
	'ž': 0x9e,
	'Ÿ': 0x9f,
}
