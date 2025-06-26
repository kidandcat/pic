package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
	"github.com/ysmood/gson"
)

func main() {
	// Define flags
	fixedTop := flag.Float64("fixed-top", 0, "Fixed top margin in pixels")
	fixedBottom := flag.Float64("fixed-bottom", 0, "Fixed bottom margin in pixels")
	outputPDF := flag.Bool("pdf", false, "Output as PDF instead of PNG")
	width := flag.Int("width", 0, "Viewport width in pixels (default: browser default)")
	height := flag.Int("height", 0, "Viewport height in pixels (default: browser default)")
	scroll := flag.Bool("scroll", false, "Use scroll fullscreen mode")

	// Custom usage function
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options] <URL>\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "\nOptions:\n")
		flag.PrintDefaults()
	}

	flag.Parse()

	// Check for URL argument
	if flag.NArg() < 1 {
		flag.Usage()
		os.Exit(1)
	}

	url := flag.Arg(0)

	// Add protocol if missing
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		url = "https://" + url
	}

	// Generate filename from URL
	baseFilename := generateFilename(url)

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Create browser and page with timeout
	browser := rod.New().Context(ctx).MustConnect()
	defer browser.MustClose()
	
	page := browser.MustPage(url)
	defer page.MustClose()
	
	// Set viewport size if specified
	if *width > 0 || *height > 0 {
		// Get current viewport size as defaults
		metrics, err := proto.PageGetLayoutMetrics{}.Call(page)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to get layout metrics: %v\n", err)
			os.Exit(1)
		}
		
		viewportWidth := int(metrics.LayoutViewport.ClientWidth)
		viewportHeight := int(metrics.LayoutViewport.ClientHeight)
		
		// Override with specified values
		if *width > 0 {
			viewportWidth = *width
		}
		if *height > 0 {
			viewportHeight = *height
		}
		
		// Set the viewport
		err = page.SetViewport(&proto.EmulationSetDeviceMetricsOverride{
			Width:             viewportWidth,
			Height:            viewportHeight,
			DeviceScaleFactor: 1,
			Mobile:            false,
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to set viewport: %v\n", err)
			os.Exit(1)
		}
	}
	
	page.MustWaitStable()

	var outputFile string

	if *outputPDF {
		// Generate PDF
		outputFile = baseFilename + ".pdf"
		page.MustPDF(outputFile)
	} else {
		// Take screenshot
		outputFile = baseFilename + ".png"

		if *scroll || *fixedTop > 0 || *fixedBottom > 0 {
			// Use ScrollScreenshot for scroll mode or when fixed margins are specified
			imgData, err := page.ScrollScreenshot(&rod.ScrollScreenshotOptions{
				Format:        proto.PageCaptureScreenshotFormatPng,
				Quality:       gson.Int(100),
				FixedTop:      *fixedTop,
				FixedBottom:   *fixedBottom,
				WaitPerScroll: 300 * time.Millisecond,
			})
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to take screenshot: %v\n", err)
				os.Exit(1)
			}

			// Write image data to file
			if err := os.WriteFile(outputFile, imgData, 0644); err != nil {
				fmt.Fprintf(os.Stderr, "Failed to save screenshot: %v\n", err)
				os.Exit(1)
			}
		} else {
			// Use regular full page screenshot
			page.MustScreenshotFullPage(outputFile)
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
