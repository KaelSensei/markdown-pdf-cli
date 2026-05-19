# Markdown PDF CLI

[![CI](https://github.com/KaelSensei/markdown-pdf-cli/actions/workflows/ci.yml/badge.svg)](https://github.com/KaelSensei/markdown-pdf-cli/actions/workflows/ci.yml)

`mdpdf` is a small offline Markdown-to-PDF converter for the terminal.

The V1 goal is intentionally narrow: generate readable PDF documents from common Markdown without a browser, a web service, LaTeX, Pandoc, or any LLM call. The converter is written in Go and currently uses only the Go standard library.

## Features

- Converts local Markdown files to local PDF files.
- Runs fully offline at conversion time.
- Uses no third-party Go dependencies.
- Supports headings, paragraphs, unordered and ordered lists, fenced code blocks, blockquotes, horizontal rules, links, images as alt text, and simple pipe tables.
- Supports A4 and Letter page sizes.
- Includes `modern`, `classic`, and `elegant` visual themes.
- Includes `light` and `dark` PDF color schemes.
- Preserves common Latin characters through PDF WinAnsi encoding.

## Install

Requirements:

- Go 1.24 or newer

From the repository root:

```bash
go build -o bin/mdpdf ./cmd/mdpdf
```

Optionally install it into your Go binary path:

```bash
go install ./cmd/mdpdf
```

## Usage

```bash
mdpdf input.md
```

This writes `input.pdf` next to the source file.

Set an explicit output path:

```bash
mdpdf input.md -o output.pdf
```

Use Letter paper and a larger margin:

```bash
mdpdf input.md -page-size letter -margin 64
```

Use the elegant dark style:

```bash
mdpdf input.md -theme elegant -color-scheme dark
```

Set PDF document title metadata:

```bash
mdpdf input.md -title "Project Notes"
```

## Command Reference

```text
Usage: mdpdf [options] input.md

Options:
  -o string
        Output PDF path. Defaults to input path with .pdf extension.
  -title string
        PDF title metadata. Defaults to the input file name.
  -page-size string
        Page size: a4 or letter. Defaults to a4.
  -theme string
        Visual theme: modern, classic, or elegant. Defaults to modern.
  -color-scheme string
        Color scheme: light or dark. Defaults to light.
  -margin float
        Page margin in PDF points. Defaults to 56.
  -quiet
        Do not print the output path.
  -version
        Print version and exit.
```

## Markdown Support

V1 focuses on predictable text documents:

- `#` through `######` headings
- Paragraphs
- `-`, `*`, and `+` unordered lists
- `1.` and `1)` ordered lists
- Fenced code blocks with backticks or tildes
- `>` blockquotes
- `---`, `***`, and `___` horizontal rules
- Simple pipe tables
- Inline links as `label (url)`
- Images as `[image: alt text]`

HTML blocks, embedded remote images, custom fonts, syntax highlighting, task lists, footnotes, and full CommonMark edge cases are not part of V1.

## Themes

The renderer includes three document themes:

- `modern`: clean sans-serif layout, strong hierarchy, blue accent, compact technical style.
- `classic`: serif typography, warm paper tone, restrained rule lines, traditional document feel.
- `elegant`: centered title treatment, serif text, subtle gold accent, polished report style.

Each theme can be rendered in `light` or `dark` mode:

```bash
mdpdf notes.md -theme modern -color-scheme light
mdpdf notes.md -theme classic -color-scheme dark
mdpdf notes.md -theme elegant -color-scheme light
```

## Offline Design

The converter does not call the network and does not use external services. It writes the PDF directly from Go code using standard PDF objects and built-in PDF fonts.

That means the output is deliberately simple and portable. It also means there is no dependency on Chromium, wkhtmltopdf, LaTeX, Typst, Pandoc, or a remote API.

## Development

This project is built and tested with Go 1.24. Use the same major/minor version
or a newer stable Go release.

Architecture notes are available in [docs/architecture.md](docs/architecture.md).

Run tests:

```bash
go test ./...
```

Generate the sample PDF:

```bash
go run ./cmd/mdpdf examples/sample.md -o sample.pdf
```

Generate all style previews:

```bash
go run ./cmd/mdpdf examples/sample.md -o sample-modern-light.pdf -theme modern -color-scheme light
go run ./cmd/mdpdf examples/sample.md -o sample-modern-dark.pdf -theme modern -color-scheme dark
go run ./cmd/mdpdf examples/sample.md -o sample-classic-light.pdf -theme classic -color-scheme light
go run ./cmd/mdpdf examples/sample.md -o sample-elegant-light.pdf -theme elegant -color-scheme light
```

## Roadmap

- Better CommonMark compliance.
- Optional embedded TrueType fonts for broader Unicode support.
- Table layout improvements.
- Optional table of contents generation.
- HTML/CSS rendering backend as an optional V2 mode.
- Web UI as a possible V2 layer on top of the same conversion core.
