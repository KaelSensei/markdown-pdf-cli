package main

import (
	"flag"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"path/filepath"
	"strings"

	"github.com/KaelSensei/markdown-pdf-cli/internal/docx"
	"github.com/KaelSensei/markdown-pdf-cli/internal/render"
	"github.com/KaelSensei/markdown-pdf-cli/internal/reverse"
)

const version = "0.2.0"

func main() {
	if len(os.Args) > 1 && os.Args[1] == "reverse" {
		runReverse(os.Args[2:])
		return
	}
	runInputToPDF(os.Args[1:])
}

func runInputToPDF(args []string) {
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
		fmt.Fprintf(flags.Output(), "Usage: %s [options] input.md|input.txt|input.docx|input.png|input.jpg|input.jpeg\n", filepath.Base(os.Args[0]))
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

	pdf, err := inputToPDF(inputPath, render.Options{
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

func inputToPDF(inputPath string, opts render.Options) ([]byte, error) {
	switch strings.ToLower(filepath.Ext(inputPath)) {
	case ".md", ".markdown":
		source, err := os.ReadFile(inputPath)
		if err != nil {
			return nil, fmt.Errorf("read input: %w", err)
		}
		return render.MarkdownToPDF(string(source), opts)
	case ".txt", ".text":
		source, err := os.ReadFile(inputPath)
		if err != nil {
			return nil, fmt.Errorf("read input: %w", err)
		}
		return render.PlainTextToPDF(string(source), opts)
	case ".docx":
		// DOCX files are ZIP packages. The docx package extracts document.xml
		// into simple Markdown so the existing renderer stays the single text
		// layout path.
		markdown, err := docx.FileToMarkdown(inputPath)
		if err != nil {
			return nil, err
		}
		return render.MarkdownToPDF(markdown, opts)
	case ".png", ".jpg", ".jpeg":
		img, err := readImage(inputPath)
		if err != nil {
			return nil, err
		}
		return render.ImageToPDF(img, opts)
	default:
		return nil, fmt.Errorf("unsupported input format %q: use .md, .txt, .docx, .png, .jpg, or .jpeg", filepath.Ext(inputPath))
	}
}

func readImage(inputPath string) (image.Image, error) {
	file, err := os.Open(inputPath)
	if err != nil {
		return nil, fmt.Errorf("read input: %w", err)
	}
	defer file.Close()

	// Blank image imports above register PNG and JPEG decoders with image.Decode
	// while keeping the CLI independent of external image tools.
	img, format, err := image.Decode(file)
	if err != nil {
		return nil, fmt.Errorf("decode image: %w", err)
	}
	if format != "png" && format != "jpeg" {
		return nil, fmt.Errorf("unsupported image format %q", format)
	}
	return img, nil
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
