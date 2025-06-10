# pic

A powerful command-line tool to capture full-page screenshots and PDFs of websites using headless Chrome.

## Features

- 📸 Capture full-page screenshots of any website
- 📄 Export web pages as PDF documents
- 🎯 Handle fixed headers/footers with adjustable margins
- 🔗 Automatic HTTPS protocol handling
- 📁 Smart filename generation based on URL
- 🚀 Fast and lightweight
- 🖥️ Cross-platform support
- 🖼️ Automatically opens captured files after saving

## Installation

### Prerequisites

- Go 1.19 or higher
- Chrome or Chromium browser installed

### Build from source

```bash
git clone https://github.com/yourusername/pic.git
cd pic
go build -o pic main.go
```

### Install with Go

```bash
go install github.com/yourusername/pic@latest
```

### Download Pre-built Binary

Visit the [releases page](https://github.com/yourusername/pic/releases) to download pre-compiled binaries for your platform.

## Usage

```bash
pic [options] <URL>
```

### Options

- `--pdf` - Output as PDF instead of PNG
- `--fixed-top <pixels>` - Fixed top margin in pixels (for handling fixed headers)
- `--fixed-bottom <pixels>` - Fixed bottom margin in pixels (for handling fixed footers)

### Examples

```bash
# Basic screenshot
pic https://example.com

# Without protocol (defaults to HTTPS)
pic example.com

# Save as PDF
pic --pdf example.com

# Handle fixed header (100px) and footer (50px)
pic --fixed-top 100 --fixed-bottom 50 example.com

# Combine PDF output with fixed margins
pic --pdf --fixed-top 100 example.com
```

### Output

Screenshots are saved in the current directory with filenames based on the URL:
- `example.com` → `example.com.png`
- `example.com/blog` → `example.com_blog.png`
- `docs.github.com/en` → `docs.github.com_en.png`

## How it works

pic uses [rod](https://github.com/go-rod/rod), a high-level Chrome DevTools Protocol library, to:
1. Launch a headless Chrome instance
2. Navigate to the specified URL
3. Wait for the page to fully load and stabilize
4. Capture a full-page screenshot or generate a PDF
5. Handle fixed headers/footers if specified
6. Save with a sanitized filename
7. Automatically open the saved file

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

- [rod](https://github.com/go-rod/rod) - The Chrome DevTools Protocol library that powers pic
- [gson](https://github.com/ysmood/gson) - Simple JSON library for Go