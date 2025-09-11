package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
)

func main() {
	// Define flags
	fixedTop := flag.Float64("fixed-top", 0, "Fixed top margin in pixels")
	fixedBottom := flag.Float64("fixed-bottom", 0, "Fixed bottom margin in pixels")
	outputPDF := flag.Bool("pdf", false, "Output as PDF instead of PNG")
	width := flag.Int("width", 1280, "Viewport width in pixels (default: 1280)")
	height := flag.Int("height", 800, "Viewport height in pixels (default: 800)")
	scroll := flag.Bool("scroll", false, "Use scroll fullscreen mode")

	// Custom usage function
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options] <URL or HTML file>\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "\nOptions:\n")
		flag.PrintDefaults()
	}

	flag.Parse()

	// Check for URL/file argument
	if flag.NArg() < 1 {
		flag.Usage()
		os.Exit(1)
	}

	input := flag.Arg(0)
	var url string
	var baseFilename string
	var server *http.Server
	
	// Check if input is an HTML file
	if strings.HasSuffix(strings.ToLower(input), ".html") || strings.HasSuffix(strings.ToLower(input), ".htm") {
		// Check if file exists
		if _, err := os.Stat(input); err != nil {
			fmt.Fprintf(os.Stderr, "HTML file not found: %v\n", err)
			os.Exit(1)
		}
		
		// Start local server for HTML file
		port, serverInstance, err := serveHTMLFile(input)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to serve HTML file: %v\n", err)
			os.Exit(1)
		}
		server = serverInstance
		defer server.Close()
		
		url = fmt.Sprintf("http://localhost:%d/%s", port, filepath.Base(input))
		// Generate filename from HTML file name
		baseFilename = strings.TrimSuffix(filepath.Base(input), filepath.Ext(input))
	} else {
		// Treat as URL
		url = input
		
		// Add protocol if missing
		if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
			url = "https://" + url
		}
		
		// Generate filename from URL
		baseFilename = generateFilename(url)
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Create chrome instance
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true),
	)

	allocCtx, allocCancel := chromedp.NewExecAllocator(ctx, opts...)
	defer allocCancel()

	chromeCtx, chromeCancel := chromedp.NewContext(allocCtx)
	defer chromeCancel()

	// Navigate to the URL
	var initialViewportWidth, initialViewportHeight int64
	if err := chromedp.Run(chromeCtx,
		chromedp.Navigate(url),
		chromedp.Evaluate(`window.innerWidth`, &initialViewportWidth),
		chromedp.Evaluate(`window.innerHeight`, &initialViewportHeight),
	); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to navigate to URL: %v\n", err)
		os.Exit(1)
	}

	// Set viewport size if specified
	if *width > 0 || *height > 0 {
		viewportWidth := *width
		viewportHeight := *height

		// Use default values if not specified
		if viewportWidth <= 0 {
			viewportWidth = int(initialViewportWidth)
		}
		if viewportHeight <= 0 {
			viewportHeight = int(initialViewportHeight)
		}

		// Set the viewport
		if err := chromedp.Run(chromeCtx,
			chromedp.EmulateViewport(int64(viewportWidth), int64(viewportHeight)),
		); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to set viewport: %v\n", err)
			os.Exit(1)
		}
	}

	// Wait for page to stabilize
	if err := chromedp.Run(chromeCtx,
		chromedp.Sleep(1*time.Second),
	); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to wait for page: %v\n", err)
		os.Exit(1)
	}

	var outputFile string

	if *outputPDF {
		// Generate PDF
		outputFile = baseFilename + ".pdf"
		var pdfData []byte
		if err := chromedp.Run(chromeCtx,
			chromedp.ActionFunc(func(ctx context.Context) error {
				var err error
				pdfData, _, err = page.PrintToPDF().
					WithPrintBackground(true).
					WithPreferCSSPageSize(true).
					Do(ctx)
				return err
			}),
		); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to generate PDF: %v\n", err)
			os.Exit(1)
		}

		if err := os.WriteFile(outputFile, pdfData, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to save PDF: %v\n", err)
			os.Exit(1)
		}
	} else {
		// Take screenshot
		outputFile = baseFilename + ".png"

		if *scroll || *fixedTop > 0 || *fixedBottom > 0 {
			// Use custom scroll screenshot implementation
			imgData, err := scrollScreenshot(chromeCtx, *fixedTop, *fixedBottom)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to take scroll screenshot: %v\n", err)
				os.Exit(1)
			}

			// Write image data to file
			if err := os.WriteFile(outputFile, imgData, 0644); err != nil {
				fmt.Fprintf(os.Stderr, "Failed to save screenshot: %v\n", err)
				os.Exit(1)
			}
		} else {
			// Use regular full page screenshot
			var buf []byte
			if err := chromedp.Run(chromeCtx,
				chromedp.FullScreenshot(&buf, 100),
			); err != nil {
				fmt.Fprintf(os.Stderr, "Failed to take screenshot: %v\n", err)
				os.Exit(1)
			}

			if err := os.WriteFile(outputFile, buf, 0644); err != nil {
				fmt.Fprintf(os.Stderr, "Failed to save screenshot: %v\n", err)
				os.Exit(1)
			}
		}
	}

	fmt.Printf("Output saved as: %s\n", outputFile)

	// Create HTML viewer with modal
	htmlFile := baseFilename + "_viewer.html"
	if err := createModalViewer(htmlFile, outputFile, *outputPDF); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create viewer: %v\n", err)
		os.Exit(1)
	}

	// Open the HTML viewer
	cmd := exec.Command("open", htmlFile)
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to open viewer: %v\n", err)
	}
}

func scrollScreenshot(ctx context.Context, fixedTop, fixedBottom float64) ([]byte, error) {
	var viewportWidth, viewportHeight int64
	var scrollHeight, scrollY int64

	// Get initial viewport and document dimensions
	if err := chromedp.Run(ctx,
		chromedp.Evaluate(`window.innerWidth`, &viewportWidth),
		chromedp.Evaluate(`window.innerHeight`, &viewportHeight),
		chromedp.Evaluate(`document.documentElement.scrollHeight`, &scrollHeight),
		chromedp.Evaluate(`window.scrollY`, &scrollY),
	); err != nil {
		return nil, fmt.Errorf("failed to get dimensions: %v", err)
	}

	// Reset scroll to top
	if err := chromedp.Run(ctx,
		chromedp.Evaluate(`window.scrollTo(0, 0)`, nil),
		chromedp.Sleep(300*time.Millisecond),
	); err != nil {
		return nil, fmt.Errorf("failed to reset scroll: %v", err)
	}

	// Calculate effective viewport height (excluding fixed areas)
	effectiveHeight := float64(viewportHeight) - fixedTop - fixedBottom
	if effectiveHeight <= 0 {
		return nil, fmt.Errorf("effective height is too small after applying fixed margins")
	}

	// Collect screenshots while scrolling
	var screenshots []image.Image
	var yPositions []int
	currentY := 0.0

	for currentY < float64(scrollHeight) {
		// Scroll to current position
		if err := chromedp.Run(ctx,
			chromedp.Evaluate(fmt.Sprintf(`window.scrollTo(0, %f)`, currentY), nil),
			chromedp.Sleep(300*time.Millisecond),
		); err != nil {
			return nil, fmt.Errorf("failed to scroll: %v", err)
		}

		// Take screenshot
		var buf []byte
		if err := chromedp.Run(ctx,
			chromedp.CaptureScreenshot(&buf),
		); err != nil {
			return nil, fmt.Errorf("failed to capture screenshot: %v", err)
		}

		// Decode image
		img, err := png.Decode(bytes.NewReader(buf))
		if err != nil {
			return nil, fmt.Errorf("failed to decode screenshot: %v", err)
		}

		screenshots = append(screenshots, img)
		yPositions = append(yPositions, int(currentY))

		// Move to next position
		currentY += effectiveHeight

		// For the last screenshot, ensure we capture the bottom
		if currentY >= float64(scrollHeight) && currentY-effectiveHeight < float64(scrollHeight)-effectiveHeight {
			currentY = float64(scrollHeight) - effectiveHeight
			if currentY < 0 {
				currentY = 0
			}
		}
	}

	// Stitch screenshots together
	return stitchScreenshots(screenshots, yPositions, int(viewportWidth), int(scrollHeight), int(fixedTop), int(fixedBottom), int(viewportHeight))
}

func stitchScreenshots(screenshots []image.Image, yPositions []int, width, totalHeight, fixedTop, fixedBottom, viewportHeight int) ([]byte, error) {
	// Create final image
	finalImg := image.NewRGBA(image.Rect(0, 0, width, totalHeight))

	for i, screenshot := range screenshots {
		bounds := screenshot.Bounds()
		scrollY := yPositions[i]

		// Calculate source rectangle (what part of the screenshot to use)
		srcTop := fixedTop
		srcBottom := bounds.Dy() - fixedBottom

		// For the first screenshot, include everything from top
		if i == 0 {
			srcTop = 0
		}

		// For the last screenshot, include everything to bottom
		if i == len(screenshots)-1 {
			srcBottom = bounds.Dy()
		}

		// Calculate destination Y position
		destY := scrollY
		if i > 0 {
			destY += fixedTop
		}

		// Draw the screenshot portion to the final image
		srcRect := image.Rect(0, int(srcTop), bounds.Dx(), int(srcBottom))
		destPoint := image.Point{0, destY}
		draw.Draw(finalImg, image.Rectangle{destPoint, destPoint.Add(srcRect.Size())}, screenshot, srcRect.Min, draw.Src)
	}

	// Encode final image with best compression
	var buf bytes.Buffer
	encoder := png.Encoder{CompressionLevel: png.BestCompression}
	if err := encoder.Encode(&buf, finalImg); err != nil {
		return nil, fmt.Errorf("failed to encode final image: %v", err)
	}

	return buf.Bytes(), nil
}

func createModalViewer(htmlFile, contentFile string, isPDF bool) error {
	htmlTemplate := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Screenshot Viewer</title>
    <style>
        body {
            margin: 0;
            padding: 0;
            font-family: Arial, sans-serif;
            background-color: #f0f0f0;
            height: 100vh;
            display: flex;
            justify-content: center;
            align-items: center;
        }
        
        .modal-backdrop {
            position: fixed;
            top: 0;
            left: 0;
            width: 100%;
            height: 100%;
            background-color: rgba(0, 0, 0, 0.8);
            display: flex;
            justify-content: center;
            align-items: center;
            cursor: pointer;
            z-index: 1000;
        }
        
        .modal-content {
            position: relative;
            max-width: 90%;
            max-height: 90%;
            background-color: white;
            border-radius: 8px;
            box-shadow: 0 4px 6px rgba(0, 0, 0, 0.1);
            cursor: default;
        }
        
        .modal-content img {
            max-width: 100%;
            max-height: 80vh;
            display: block;
            border-radius: 8px;
        }
        
        .modal-content iframe {
            width: 80vw;
            height: 80vh;
            border: none;
            border-radius: 8px;
        }
        
        .close-button {
            position: absolute;
            top: -40px;
            right: 0;
            background-color: white;
            color: #333;
            border: none;
            border-radius: 50%;
            width: 36px;
            height: 36px;
            font-size: 20px;
            cursor: pointer;
            display: flex;
            justify-content: center;
            align-items: center;
            transition: background-color 0.3s;
        }
        
        .close-button:hover {
            background-color: #f0f0f0;
        }
        
        .info {
            position: absolute;
            bottom: -40px;
            left: 50%;
            transform: translateX(-50%);
            color: white;
            font-size: 14px;
            white-space: nowrap;
        }
    </style>
</head>
<body>
    <div class="modal-backdrop" onclick="closeModal(event)">
        <div class="modal-content" onclick="event.stopPropagation()">
            <button class="close-button" onclick="closeModal(event)">×</button>
            {{CONTENT}}
            <div class="info">Click outside or press ESC to close</div>
        </div>
    </div>
    
    <script>
        function closeModal(event) {
            if (event) {
                event.preventDefault();
            }
            window.close();
        }
        
        // Close on ESC key
        document.addEventListener('keydown', function(event) {
            if (event.key === 'Escape') {
                closeModal();
            }
        });
        
        // Auto-open modal on page load
        window.onload = function() {
            document.body.style.display = 'flex';
        };
    </script>
</body>
</html>`

	// Replace content based on type
	var content string
	if isPDF {
		content = fmt.Sprintf(`<iframe src="%s"></iframe>`, contentFile)
	} else {
		content = fmt.Sprintf(`<img src="%s" alt="Screenshot">`, contentFile)
	}

	html := strings.Replace(htmlTemplate, "{{CONTENT}}", content, 1)

	return os.WriteFile(htmlFile, []byte(html), 0644)
}

func generateFilename(url string) string {
	// Remove protocol
	cleanURL := strings.TrimPrefix(url, "https://")
	cleanURL = strings.TrimPrefix(cleanURL, "http://")

	// Replace special characters with underscores
	reg := regexp.MustCompile(`[^a-zA-Z0-9.-]+`)
	cleanURL = reg.ReplaceAllString(cleanURL, "_")

	// Remove trailing underscores
	cleanURL = strings.Trim(cleanURL, "_")

	// Return without extension (will be added based on output type)
	return cleanURL
}

func serveHTMLFile(htmlFile string) (int, *http.Server, error) {
	// Get absolute path
	absPath, err := filepath.Abs(htmlFile)
	if err != nil {
		return 0, nil, fmt.Errorf("failed to get absolute path: %v", err)
	}
	
	// Get the directory containing the HTML file
	dir := filepath.Dir(absPath)
	
	// Find an available port
	listener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		return 0, nil, fmt.Errorf("failed to find available port: %v", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()
	
	// Create file server
	mux := http.NewServeMux()
	
	// Serve the directory containing the HTML file
	fileServer := http.FileServer(http.Dir(dir))
	mux.Handle("/", fileServer)
	
	// Create server
	server := &http.Server{
		Addr:    fmt.Sprintf("localhost:%d", port),
		Handler: mux,
	}
	
	// Start server in background
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
		}
	}()
	
	// Wait a bit for server to start
	time.Sleep(100 * time.Millisecond)
	
	return port, server, nil
}