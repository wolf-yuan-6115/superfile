package filepreview

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/png"
	"log/slog"
	"os"
	"strings"
)

// isITerm2Capable checks if the terminal supports iTerm2 inline image protocol
func isITerm2Capable() bool {
	termProgram := os.Getenv("TERM_PROGRAM")
	term := os.Getenv("TERM")

	// Check for specific environment variables that indicate iTerm2 support
	if os.Getenv("ITERM_SESSION_ID") != "" {
		slog.Debug("iTerm2 protocol supported via ITERM_SESSION_ID")
		return true
	}

	if os.Getenv("VSCODE_INJECTION") != "" {
		slog.Debug("iTerm2 protocol supported via VSCODE_INJECTION")
		return true
	}

	if os.Getenv("TABBY_CONFIG_DIRECTORY") != "" {
		slog.Debug("iTerm2 protocol supported via TABBY_CONFIG_DIRECTORY")
		return true
	}

	if os.Getenv("WARP_HONOR_PS1") != "" {
		slog.Debug("iTerm2 protocol supported via WARP_HONOR_PS1")
		return true
	}

	// List of known terminal identifiers that support iTerm2 inline image protocol
	knownTerminals := []string{
		"iTerm.app",      // iTerm2
		"vscode",         // VSCode integrated terminal
		"Tabby",          // Tabby terminal
		"Hyper",          // Hyper terminal
		"konsole",        // KDE Konsole
		"Mintty",         // Windows Mintty (Git Bash, etc.)
		"WarpTerminal",   // Warp terminal
		"WezTerm",        // WezTerm also supports iTerm2 protocol
		"rio",            // Rio terminal
		"Bobcat",         // Bobcat terminal
	}

	// Check TERM_PROGRAM environment variable
	for _, knownTerm := range knownTerminals {
		if strings.EqualFold(termProgram, knownTerm) {
			slog.Debug("iTerm2 protocol supported via TERM_PROGRAM", "terminal", termProgram)
			return true
		}
	}

	// Check TERM environment variable for some cases
	if strings.Contains(strings.ToLower(term), "iterm") ||
		strings.Contains(strings.ToLower(term), "konsole") {
		slog.Debug("iTerm2 protocol supported via TERM", "terminal", term)
		return true
	}

	slog.Debug("iTerm2 protocol not supported", "TERM_PROGRAM", termProgram, "TERM", term)
	return false
}

// IsITerm2Capable checks if the terminal supports iTerm2 inline image protocol
func (p *ImagePreviewer) IsITerm2Capable() bool {
	return isITerm2Capable()
}

// renderWithITerm2 renders an image using iTerm2 inline image protocol
func (p *ImagePreviewer) renderWithITerm2(img image.Image, _ string,
	originalWidth, originalHeight, maxWidth, maxHeight int, sideAreaWidth int) (string, error) {
	// Validate dimensions
	if maxWidth <= 0 || maxHeight <= 0 {
		return "", fmt.Errorf("dimensions must be positive (maxWidth=%d, maxHeight=%d)", maxWidth, maxHeight)
	}

	// Get terminal cell size for proper scaling
	cellSize := p.terminalCap.GetTerminalCellSize()
	pixelsPerColumn := cellSize.PixelsPerColumn
	pixelsPerRow := cellSize.PixelsPerRow

	// Calculate display dimensions while preserving aspect ratio
	imgRatio := float64(originalWidth) / float64(originalHeight)
	maxPixelsWidth := maxWidth * pixelsPerColumn
	maxPixelsHeight := maxHeight * pixelsPerRow
	termRatio := float64(maxPixelsWidth) / float64(maxPixelsHeight)

	var displayWidth, displayHeight int

	if imgRatio > termRatio {
		// Image is wider than terminal area - constrain by width
		displayWidth = maxPixelsWidth
		displayHeight = int(float64(maxPixelsWidth) / imgRatio)
	} else {
		// Image is taller than terminal area - constrain by height
		displayHeight = maxPixelsHeight
		displayWidth = int(float64(maxPixelsHeight) * imgRatio)
	}

	slog.Debug("iTerm2 image dimensions",
		"original", fmt.Sprintf("%dx%d", originalWidth, originalHeight),
		"display", fmt.Sprintf("%dx%d", displayWidth, displayHeight),
		"max_pixels", fmt.Sprintf("%dx%d", maxPixelsWidth, maxPixelsHeight),
		"cell_size", fmt.Sprintf("%dx%d", pixelsPerColumn, pixelsPerRow))

	// Convert image to PNG format for iTerm2
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return "", fmt.Errorf("failed to encode image as PNG: %w", err)
	}

	// Encode image data as base64
	encodedData := base64.StdEncoding.EncodeToString(buf.Bytes())

	// Build iTerm2 escape sequence
	// Format: \x1b]1337;File=inline=1;width=<width>px;height=<height>px:<base64_data>\x07
	var result strings.Builder

	// Start escape sequence
	result.WriteString("\x1b]1337;File=inline=1")

	// Add dimensions if we want to specify them
	result.WriteString(fmt.Sprintf(";width=%dpx;height=%dpx", displayWidth, displayHeight))

	// Add the base64 encoded image data
	result.WriteString(":")
	result.WriteString(encodedData)

	// End escape sequence
	result.WriteString("\x07")

	// Position cursor to account for side area width
	if sideAreaWidth > 0 {
		result.WriteString(fmt.Sprintf("\x1b[1;%dH", sideAreaWidth))
	}

	return result.String(), nil
}

// ClearITerm2Images clears iTerm2 inline images (currently no standard way to clear specific images)
func (p *ImagePreviewer) ClearITerm2Images() string {
	// iTerm2 doesn't have a standard way to clear specific inline images
	// The images are part of the terminal scrollback and get cleared when the terminal is cleared
	// We could potentially use cursor positioning to overwrite the area, but that's complex
	// For now, return empty string as clearing is handled by terminal scrollback management
	return ""
}

// ClearITerm2Images clears iTerm2 inline images from the terminal
func ClearITerm2Images() string {
	// iTerm2 doesn't have a direct clear command for inline images like Kitty
	// Images are cleared when terminal content scrolls away or terminal is cleared
	return ""
}
