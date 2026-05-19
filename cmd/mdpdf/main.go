package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/KaelSensei/markdown-pdf-cli/internal/render"
	"github.com/KaelSensei/markdown-pdf-cli/internal/reverse"
)

const version = "0.2.0"

func main() {
	if len(os.Args) > 1 && os.Args[1] == "reverse" {
		runReverse(os.Args[2:])
		return
	}
	runMarkdownToPDF(os.Args[1:])
}

func runMarkdownToPDF(args []string) {
	var (
		outputPath string
		title      string
		pageSize   string
		theme      string
		scheme     string
		margin     float64
		quiet      bool
		showVer    bool
	)

	flags := flag.NewFlagSet("mdpdf", flag.ExitOnError)
	flags.StringVar(&outputPath, "o", "", "Output PDF path. Defaults to input path with .pdf extension.")
	flags.StringVar(&title, "title", "", "PDF title metadata. Defaults to the input file name.")
	flags.StringVar(&pageSize, "page-size", "a4", "Page size: a4 or letter.")
	flags.StringVar(&theme, "theme", "modern", "Visual theme: modern, classic, or elegant.")
	flags.StringVar(&scheme, "color-scheme", "light", "Color scheme: light or dark.")
	flags.Float64Var(&margin, "margin", 56, "Page margin in PDF points.")
	flags.BoolVar(&quiet, "quiet", false, "Do not print the output path.")
	flags.BoolVar(&showVer, "version", false, "Print version and exit.")
	flags.Usage = func() {
		fmt.Fprintf(flags.Output(), "Usage: %s [options] input.md\n", filepath.Base(os.Args[0]))
		fmt.Fprintf(flags.Output(), "       %s reverse [options] input.pdf\n\nOptions:\n", filepath.Base(os.Args[0]))
		flags.PrintDefaults()
	}

	if err := flags.Parse(normalizeArgs(args)); err != nil {
		exitError(err)
	}
	if showVer {
		fmt.Println(version)
		return
	}
	if flags.NArg() != 1 {
		flags.Usage()
		os.Exit(2)
	}

	inputPath := flags.Arg(0)
	if outputPath == "" {
		outputPath = defaultOutputPath(inputPath, ".pdf")
	}
	if title == "" {
		title = defaultTitle(inputPath)
	}

	source, err := os.ReadFile(inputPath)
	if err != nil {
		exitError(fmt.Errorf("read input: %w", err))
	}

	pdf, err := render.MarkdownToPDF(string(source), render.Options{
		Title:       title,
		PageSize:    pageSize,
		Margin:      margin,
		Theme:       theme,
		ColorScheme: scheme,
	})
	if err != nil {
		exitError(err)
	}

	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil && filepath.Dir(outputPath) != "." {
		exitError(fmt.Errorf("create output directory: %w", err))
	}
	if err := os.WriteFile(outputPath, pdf, 0o644); err != nil {
		exitError(fmt.Errorf("write output: %w", err))
	}
	if !quiet {
		fmt.Printf("Wrote %s\n", outputPath)
	}
}

func runReverse(args []string) {
	var (
		outputPath    string
		quiet         bool
		preservePages bool
	)

	flags := flag.NewFlagSet("mdpdf reverse", flag.ExitOnError)
	flags.StringVar(&outputPath, "o", "", "Output Markdown path. Defaults to input path with .md extension.")
	flags.BoolVar(&preservePages, "preserve-pages", false, "Insert page boundary comments into the Markdown output.")
	flags.BoolVar(&quiet, "quiet", false, "Do not print the output path.")
	flags.Usage = func() {
		fmt.Fprintf(flags.Output(), "Usage: %s reverse [options] input.pdf\n\nOptions:\n", filepath.Base(os.Args[0]))
		flags.PrintDefaults()
	}

	if err := flags.Parse(normalizeArgs(args)); err != nil {
		exitError(err)
	}
	if flags.NArg() != 1 {
		flags.Usage()
		os.Exit(2)
	}

	inputPath := flags.Arg(0)
	if outputPath == "" {
		outputPath = defaultOutputPath(inputPath, ".md")
	}

	markdown, err := reverse.PDFFileToMarkdown(inputPath, reverse.Options{
		PreservePages: preservePages,
	})
	if err != nil {
		exitError(err)
	}

	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil && filepath.Dir(outputPath) != "." {
		exitError(fmt.Errorf("create output directory: %w", err))
	}
	if err := os.WriteFile(outputPath, []byte(markdown), 0o644); err != nil {
		exitError(fmt.Errorf("write output: %w", err))
	}
	if !quiet {
		fmt.Printf("Wrote %s\n", outputPath)
	}
}

func defaultOutputPath(inputPath, targetExt string) string {
	if targetExt == "" || !strings.HasPrefix(targetExt, ".") {
		targetExt = "." + targetExt
	}
	sourceExt := filepath.Ext(inputPath)
	if sourceExt == "" {
		return inputPath + targetExt
	}
	return strings.TrimSuffix(inputPath, sourceExt) + targetExt
}

// normalizeArgs lets users place flags before or after the input path.
//
// Go's standard flag package stops parsing at the first positional argument,
// while many CLI users expect "mdpdf input.md -o out.pdf" to work.
func normalizeArgs(args []string) []string {
	var flagArgs []string
	var positional []string

	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			positional = append(positional, args[i+1:]...)
			break
		}
		if strings.HasPrefix(arg, "-") && arg != "-" {
			flagArgs = append(flagArgs, arg)
			name := strings.TrimLeft(arg, "-")
			if before, _, ok := strings.Cut(name, "="); ok {
				name = before
			} else if flagNeedsValue(name) && i+1 < len(args) {
				i++
				flagArgs = append(flagArgs, args[i])
			}
			continue
		}
		positional = append(positional, arg)
	}

	return append(flagArgs, positional...)
}

func flagNeedsValue(name string) bool {
	switch name {
	case "o", "title", "page-size", "theme", "color-scheme", "margin":
		return true
	default:
		return false
	}
}

func defaultTitle(inputPath string) string {
	name := filepath.Base(inputPath)
	ext := filepath.Ext(name)
	return strings.TrimSuffix(name, ext)
}

func exitError(err error) {
	fmt.Fprintf(os.Stderr, "mdpdf: %v\n", err)
	os.Exit(1)
}
