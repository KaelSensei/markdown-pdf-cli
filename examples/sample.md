# Markdown PDF CLI Sample

This document demonstrates the default rendering features of `mdpdf`.

## Lists

- Offline conversion
- No browser dependency
- No LLM calls
- Standard-library Go implementation

1. Build the binary.
2. Convert a Markdown file.
3. Share the generated PDF.

## Blockquote

> Simple text documents should be easy to convert locally, without sending content to a remote service.

## Code

```go
package main

import "fmt"

func main() {
    fmt.Println("Markdown to PDF")
}
```

## Table

| Area | V1 Support |
| --- | --- |
| Headings | Yes |
| Lists | Yes |
| Code blocks | Yes |
| Remote images | No |

## Links And Images

A link is rendered as [project repository](https://github.com/KaelSensei/markdown-pdf-cli).

An image is rendered as ![architecture diagram](diagram.png) so the PDF remains self-contained.
