# Security Policy

## Supported Versions

Security fixes are applied to the latest released version.

## Reporting A Vulnerability

Please report security issues privately through GitHub Security Advisories when
available. If advisories are not available, open a minimal issue that does not
include exploit details and ask for a private contact path.

Do not publish proof-of-concept payloads publicly until a fix is available.

## Security Model

`mdpdf` is a local CLI tool. It reads local files and writes local files. It does
not call the network at conversion time and does not evaluate JavaScript, HTML,
or Markdown as executable content.

Markdown-to-PDF rendering writes PDF text through escaped PDF string literals.
This is intended to prevent Markdown content from breaking out of text objects
and injecting PDF operators.

PDF-to-Markdown reverse conversion treats input PDFs as untrusted files. It uses
a Go PDF extraction library and converts extracted text into Markdown with
best-effort heuristics.

## Untrusted Inputs

Use normal caution with untrusted files:

- Run conversion in a low-privilege directory when processing unknown PDFs.
- Review Markdown extracted from untrusted PDFs before publishing it to a web
  renderer.
- Do not assume reverse conversion sanitizes HTML-like text for every downstream
  Markdown renderer.
- Scanned PDFs or malformed PDFs may fail extraction or produce incomplete
  Markdown.

## Dependency Monitoring

Dependabot monitors:

- Go modules
- GitHub Actions

GitHub Dependabot alerts and automated security fixes are enabled for this
repository.

## Security Automation

The repository runs:

- Go tests, `go vet`, and a CLI build in CI.
- CodeQL analysis for Go.
- `govulncheck` against all Go packages.

Private vulnerability reporting is enabled on GitHub.
