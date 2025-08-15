package filepreview

import (
	"bytes"
	"fmt"
	"image"
	"log/slog"
	"os"
	"strings"

	"github.com/disintegration/imaging"
	"github.com/mattn/go-sixel"
)

// IsSixelCapable checks if the terminal supports Sixel graphics protocol
func (p *ImagePreviewer) IsSixelCapable() bool {
	return isSixelCapable()
}

// isSixelCapable checks if the terminal supports Sixel graphics protocol
func isSixelCapable() bool {
	// Check environment variables for known Sixel-capable terminals
	termProgram := os.Getenv("TERM_PROGRAM")
	term := os.Getenv("TERM")

	// List of known terminal identifiers that support Sixel protocol
	knownSixelTerminals := []string{
		"foot",
		"mlterm",
		"xterm",
		"mintty",
	}

	// Check TERM_PROGRAM first
	for _, knownTerm := range knownSixelTerminals {
		if strings.EqualFold(termProgram, knownTerm) {
			return true
		}
	}

	// Check TERM variable
	for _, knownTerm := range knownSixelTerminals {
		if strings.EqualFold(term, knownTerm) || strings.Contains(strings.ToLower(term), knownTerm) {
			return true
		}
	}

	// Check for xterm variants that typically support Sixel
	if strings.Contains(strings.ToLower(term), "xterm") {
		return true
	}

	// Additional check for specific terminal capabilities
	// Some terminals set specific environment variables
	if os.Getenv("FOOT_VERSION") != "" {
		return true
	}

	return false
}

// renderWithSixel renders an image using Sixel graphics protocol
func (p *ImagePreviewer) renderWithSixel(img image.Image, maxWidth, maxHeight int) (string, error) {
	// Validate dimensions
	if maxWidth <= 0 || maxHeight <= 0 {
		return "", fmt.Errorf("dimensions must be positive (maxWidth=%d, maxHeight=%d)", maxWidth, maxHeight)
	}

	// Get terminal cell size for proper scaling
	cellSize := p.terminalCap.GetTerminalCellSize()
	pixelsPerColumn := cellSize.PixelsPerColumn
	pixelsPerRow := cellSize.PixelsPerRow

	// Calculate target dimensions in pixels
	targetPixelWidth := maxWidth * pixelsPerColumn
	targetPixelHeight := maxHeight * pixelsPerRow

	// Get original image dimensions
	originalWidth := img.Bounds().Dx()
	originalHeight := img.Bounds().Dy()

	// Calculate aspect ratios
	imgRatio := float64(originalWidth) / float64(originalHeight)
	termRatio := float64(targetPixelWidth) / float64(targetPixelHeight)

	// Calculate final dimensions maintaining aspect ratio
	var finalWidth, finalHeight int
	if imgRatio > termRatio {
		// Image is wider than terminal ratio, fit to width
		finalWidth = targetPixelWidth
		finalHeight = int(float64(targetPixelWidth) / imgRatio)
	} else {
		// Image is taller than terminal ratio, fit to height
		finalHeight = targetPixelHeight
		finalWidth = int(float64(targetPixelHeight) * imgRatio)
	}

	// Resize image to final dimensions
	resizedImg := resizeImageForSixel(img, finalWidth, finalHeight)

	// Convert to Sixel format
	var buf bytes.Buffer
	enc := sixel.NewEncoder(&buf)

	// Configure encoder for better quality
	enc.Colors = 256 // Use 256 colors for better quality
	enc.Dither = true

	err := enc.Encode(resizedImg)
	if err != nil {
		return "", fmt.Errorf("failed to encode image as Sixel: %w", err)
	}

	result := buf.String()
	slog.Debug("Sixel rendering completed",
		"original_size", fmt.Sprintf("%dx%d", originalWidth, originalHeight),
		"final_size", fmt.Sprintf("%dx%d", finalWidth, finalHeight),
		"output_size", len(result))

	return result, nil
}

// resizeImageForSixel resizes image for Sixel rendering while maintaining quality
func resizeImageForSixel(img image.Image, width, height int) image.Image {
	// Use the existing imaging library for consistent image processing
	return resizeImage(img, width, height)
}

// resizeImage is a helper function that uses the imaging library
// This maintains consistency with other renderers
func resizeImage(img image.Image, width, height int) image.Image {
	// Use the same imaging library that's already used in the project
	// This ensures consistent image processing across all renderers
	return imaging.Fit(img, width, height, imaging.Lanczos)
}

// ClearSixelImages clears Sixel images from the terminal
func ClearSixelImages() string {
	if !isSixelCapable() {
		return "" // No need to clear if terminal doesn't support Sixel
	}

	// Sixel clear sequence
	return "\x1b[2J" // Clear screen
}

// ClearSixelImages clears Sixel images from the terminal (method version)
func (p *ImagePreviewer) ClearSixelImages() string {
	if !p.IsSixelCapable() {
		return ""
	}

	return ClearSixelImages()
}