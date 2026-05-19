# Architecture

`mdpdf` is built as a small local conversion pipeline:

```text
Markdown file -> CLI -> Markdown parser -> Renderer -> PDF writer -> PDF file
PDF file -> CLI -> PDF extractor -> Reverse renderer -> Markdown file
```

The main design constraint is offline conversion. The program does not call a
web service, start a browser, invoke a large document toolchain, or use an LLM at
conversion time.

## Packages

### `cmd/mdpdf`

The CLI package owns user-facing behavior:

- Parse command-line flags.
- Read the Markdown source file.
- Choose default output paths and PDF titles.
- Call the renderer.
- Write the generated PDF bytes.
- Dispatch reverse conversion through `mdpdf reverse`.

The CLI intentionally stays thin so the conversion core can later be reused by a
web UI, desktop wrapper, or batch processor.

### `internal/markdown`

The Markdown package parses the V1-supported Markdown subset into block data.

Supported blocks include headings, paragraphs, lists, fenced code blocks,
blockquotes, thematic breaks, simple pipe tables, links, and image alt text. It
does not attempt full CommonMark compatibility. This keeps behavior predictable
and avoids pulling in a large parsing dependency before the project needs it.

### `internal/render`

The renderer turns parsed blocks into page layout decisions:

- Page size and margins.
- Theme and color scheme selection.
- Text wrapping.
- Pagination.
- Block-specific layout for headings, code blocks, quotes, tables, and lists.

The renderer uses top-left coordinates because that model is easier to reason
about for document layout. The PDF writer handles conversion to PDF's native
bottom-left coordinate system.

### `internal/pdf`

The PDF package writes the final PDF file directly with standard-library Go.

It creates PDF objects, page streams, built-in font resources, metadata, an xref
table, and trailer data. It uses built-in Type 1 fonts and WinAnsi encoding, so
generated files do not need bundled fonts.

### `internal/reverse`

The reverse package extracts positioned PDF text and converts it into
best-effort Markdown.

It groups extracted text into visual lines, merges adjacent glyph chunks,
detects common document patterns, and emits readable Markdown. Detection is
heuristic because PDF files do not reliably preserve source-level structure.

The package currently infers:

- Headings from larger bold text.
- Paragraphs from line spacing.
- Lists from bullet or numbered prefixes.
- Code blocks from monospace fonts.
- Blockquotes from italic text.
- Simple tables from repeated aligned text chunks.

## Styling Model

Themes are plain Go data structures. A theme defines fonts, text sizes,
spacing, page accents, and colors for light mode. Dark mode is applied as a
palette transform over the selected theme.

Current themes:

- `modern`
- `classic`
- `elegant`

Current color schemes:

- `light`
- `dark`

This keeps the rendering layer deterministic and makes new themes easy to add
without changing parser or PDF serialization code.

## Markdown To PDF Scope

The V1 converter prioritizes clean, readable technical documents. It is not a
browser engine and does not support arbitrary HTML/CSS, JavaScript, remote
images, or custom font loading.

This tradeoff is intentional: the output is simpler, but the binary remains
portable and the conversion stays fully local.

## PDF To Markdown

PDF-to-Markdown is possible, but it is not a true inverse of Markdown-to-PDF.
A PDF usually stores positioned text and drawing commands, not the original
semantic document structure.

The V2 reverse mode is therefore treated as best-effort extraction:

- Extract text from pages.
- Infer headings from font size and position when available.
- Infer paragraphs from spacing.
- Infer lists from bullets, numbering, and indentation.
- Infer tables only when text alignment is clear enough.

The reverse pipeline stays in a separate package and command mode so it does not
complicate the Markdown-to-PDF renderer.

## Future Extension Points

- Embedded TrueType fonts for broader Unicode support.
- A richer Markdown parser if full CommonMark compatibility becomes important.
- Optional syntax highlighting for fenced code blocks.
- Table of contents generation.
- A batch conversion mode.
- A web UI built on top of the existing conversion core.
