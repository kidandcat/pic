package main

import (
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

	// Create browser and page
	page := rod.New().MustConnect().MustPage(url)
	
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

	// Open the output file
	cmd := exec.Command("open", outputFile)
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to open output file: %v\n", err)
	}
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
