package filepreview

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"log/slog"
	"math"
	"os"
	"strings"
)

// isSixelCapable checks if the terminal supports Sixel graphics
func isSixelCapable() bool {
	termProgram := os.Getenv("TERM_PROGRAM")
	term := os.Getenv("TERM")

	// Check for specific environment variables that indicate Sixel support
	if os.Getenv("WT_SESSION") != "" { // Windows Terminal
		slog.Debug("Sixel protocol supported via WT_SESSION")
		return true
	}

	if os.Getenv("KONSOLE_VERSION") != "" { // KDE Konsole (older versions)
		slog.Debug("Sixel protocol supported via KONSOLE_VERSION")
		return true
	}

	// List of known terminal identifiers that support Sixel graphics
	knownSixelTerminals := []string{
		"foot",         // foot terminal
		"Microsoft",    // Windows Terminal (via TERM_PROGRAM)
		"BlackBox",     // BlackBox terminal
		"rio",          // Rio terminal
		"mlterm",       // mlterm
		"DomTerm",      // DomTerm
		"WezTerm",      // WezTerm supports Sixel
		"iTerm.app",    // iTerm2 supports Sixel
		"Tabby",        // Tabby supports Sixel
		"Hyper",        // Hyper supports Sixel
		"vscode",       // VSCode supports Sixel
		"Bobcat",       // Bobcat supports Sixel
	}

	// Check TERM_PROGRAM environment variable
	for _, knownTerm := range knownSixelTerminals {
		if strings.EqualFold(termProgram, knownTerm) {
			slog.Debug("Sixel protocol supported via TERM_PROGRAM", "terminal", termProgram)
			return true
		}
	}

	// Check TERM environment variable for specific patterns
	termLower := strings.ToLower(term)
	if strings.Contains(termLower, "xterm") ||
		strings.Contains(termLower, "foot") ||
		strings.Contains(termLower, "rio") ||
		strings.Contains(termLower, "mlterm") ||
		strings.Contains(termLower, "sixel") ||
		strings.Contains(termLower, "iterm") ||
		strings.Contains(termLower, "konsole") {
		slog.Debug("Sixel protocol supported via TERM", "terminal", term)
		return true
	}

	slog.Debug("Sixel protocol not supported", "TERM_PROGRAM", termProgram, "TERM", term)
	return false
}

// IsSixelCapable checks if the terminal supports Sixel graphics
func (p *ImagePreviewer) IsSixelCapable() bool {
	return isSixelCapable()
}

// Color represents a quantized RGB color for Sixel
type SixelColor struct {
	R, G, B uint8
	Index   int
}

// ColorQuantizer handles color quantization for Sixel graphics
type ColorQuantizer struct {
	palette []SixelColor
	maxColors int
}

// NewColorQuantizer creates a new color quantizer
func NewColorQuantizer(maxColors int) *ColorQuantizer {
	return &ColorQuantizer{
		palette:   make([]SixelColor, 0, maxColors),
		maxColors: maxColors,
	}
}

// quantizeImage performs simple color quantization using a basic algorithm
func (cq *ColorQuantizer) quantizeImage(img image.Image) ([][]int, error) {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// Extract unique colors
	colorMap := make(map[color.RGBA]int)
	var colors []color.RGBA

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			c := img.At(x, y)
			rgba := color.RGBAModel.Convert(c).(color.RGBA)
			
			// Skip transparent pixels
			if rgba.A == 0 {
				continue
			}
			
			if _, exists := colorMap[rgba]; !exists {
				colorMap[rgba] = len(colors)
				colors = append(colors, rgba)
			}
		}
	}

	// If we have too many colors, perform simple quantization
	if len(colors) > cq.maxColors {
		colors = cq.simplifyPalette(colors, cq.maxColors)
		// Rebuild color map
		colorMap = make(map[color.RGBA]int)
		for i, c := range colors {
			colorMap[c] = i
		}
	}

	// Build palette
	cq.palette = make([]SixelColor, len(colors))
	for i, c := range colors {
		cq.palette[i] = SixelColor{
			R:     c.R,
			G:     c.G,
			B:     c.B,
			Index: i,
		}
	}

	// Create index map
	indices := make([][]int, height)
	for y := 0; y < height; y++ {
		indices[y] = make([]int, width)
		for x := 0; x < width; x++ {
			c := img.At(bounds.Min.X+x, bounds.Min.Y+y)
			rgba := color.RGBAModel.Convert(c).(color.RGBA)
			
			if rgba.A == 0 {
				indices[y][x] = -1 // Transparent
			} else {
				// Find closest color in palette
				if idx, exists := colorMap[rgba]; exists {
					indices[y][x] = idx
				} else {
					indices[y][x] = cq.findClosestColor(rgba)
				}
			}
		}
	}

	return indices, nil
}

// simplifyPalette reduces colors using simple binning
func (cq *ColorQuantizer) simplifyPalette(colors []color.RGBA, maxColors int) []color.RGBA {
	if len(colors) <= maxColors {
		return colors
	}

	// Group colors by similarity (simple binning approach)
	bins := make(map[uint32][]color.RGBA)
	
	for _, c := range colors {
		// Create a simplified color key by reducing precision
		key := uint32(c.R>>4)<<16 | uint32(c.G>>4)<<8 | uint32(c.B>>4)
		bins[key] = append(bins[key], c)
	}

	// If we still have too many bins, further reduce
	for len(bins) > maxColors {
		// Find the smallest bin and merge it with the closest bin
		var smallestKey uint32
		smallestSize := len(colors) + 1
		
		for key, bin := range bins {
			if len(bin) < smallestSize {
				smallestSize = len(bin)
				smallestKey = key
			}
		}
		
		// Find closest bin to merge with
		smallestBin := bins[smallestKey]
		delete(bins, smallestKey)
		
		if len(bins) > 0 {
			// Find the closest remaining bin
			var closestKey uint32
			minDistance := math.MaxFloat64
			
			avgColor := averageColors(smallestBin)
			
			for key, bin := range bins {
				avgOther := averageColors(bin)
				distance := colorDistance(avgColor, avgOther)
				if distance < minDistance {
					minDistance = distance
					closestKey = key
				}
			}
			
			// Merge bins
			bins[closestKey] = append(bins[closestKey], smallestBin...)
		}
	}

	// Create final palette by averaging colors in each bin
	result := make([]color.RGBA, 0, maxColors)
	for _, bin := range bins {
		result = append(result, averageColors(bin))
	}

	return result
}

// averageColors computes the average color of a slice of colors
func averageColors(colors []color.RGBA) color.RGBA {
	if len(colors) == 0 {
		return color.RGBA{}
	}
	
	var totalR, totalG, totalB uint32
	for _, c := range colors {
		totalR += uint32(c.R)
		totalG += uint32(c.G)
		totalB += uint32(c.B)
	}
	
	count := uint32(len(colors))
	return color.RGBA{
		R: uint8(totalR / count),
		G: uint8(totalG / count),
		B: uint8(totalB / count),
		A: 255,
	}
}

// colorDistance computes the Euclidean distance between two colors
func colorDistance(c1, c2 color.RGBA) float64 {
	dr := float64(c1.R) - float64(c2.R)
	dg := float64(c1.G) - float64(c2.G)
	db := float64(c1.B) - float64(c2.B)
	return math.Sqrt(dr*dr + dg*dg + db*db)
}

// findClosestColor finds the index of the closest color in the palette
func (cq *ColorQuantizer) findClosestColor(target color.RGBA) int {
	if len(cq.palette) == 0 {
		return 0
	}
	
	minDistance := math.MaxFloat64
	closestIndex := 0
	
	for i, paletteColor := range cq.palette {
		distance := colorDistance(target, color.RGBA{paletteColor.R, paletteColor.G, paletteColor.B, 255})
		if distance < minDistance {
			minDistance = distance
			closestIndex = i
		}
	}
	
	return closestIndex
}

// renderWithSixel renders an image using Sixel graphics protocol
func (p *ImagePreviewer) renderWithSixel(img image.Image, _ string,
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

	// Resize image to fit display area
	resizedImg := resizeImage(img, displayWidth, displayHeight)
	
	slog.Debug("Sixel image dimensions",
		"original", fmt.Sprintf("%dx%d", originalWidth, originalHeight),
		"display", fmt.Sprintf("%dx%d", displayWidth, displayHeight),
		"resized", fmt.Sprintf("%dx%d", resizedImg.Bounds().Dx(), resizedImg.Bounds().Dy()),
		"cell_size", fmt.Sprintf("%dx%d", pixelsPerColumn, pixelsPerRow))

	// Quantize colors for Sixel (limit to 256 colors for compatibility)
	quantizer := NewColorQuantizer(256)
	indices, err := quantizer.quantizeImage(resizedImg)
	if err != nil {
		return "", fmt.Errorf("failed to quantize image colors: %w", err)
	}

	// Generate Sixel data
	sixelData, err := p.generateSixelData(resizedImg, indices, quantizer.palette)
	if err != nil {
		return "", fmt.Errorf("failed to generate Sixel data: %w", err)
	}

	// Build final result with cursor positioning
	var result strings.Builder
	
	// Position cursor to account for side area width
	if sideAreaWidth > 0 {
		result.WriteString(fmt.Sprintf("\x1b[1;%dH", sideAreaWidth))
	}
	
	result.WriteString(sixelData)

	return result.String(), nil
}

// generateSixelData creates the actual Sixel escape sequence
func (p *ImagePreviewer) generateSixelData(img image.Image, indices [][]int, palette []SixelColor) (string, error) {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	var buf bytes.Buffer

	// Start Sixel sequence
	buf.WriteString("\x1bP0;1;8q")

	// Write palette
	for i, color := range palette {
		// Convert RGB (0-255) to percentage (0-100)
		r := int(color.R) * 100 / 255
		g := int(color.G) * 100 / 255
		b := int(color.B) * 100 / 255
		buf.WriteString(fmt.Sprintf("#%d;2;%d;%d;%d", i, r, g, b))
	}

	// Process image in strips of 6 pixels high (Sixel limitation)
	for y := 0; y < height; y += 6 {
		// Process each color in the palette
		for colorIndex, _ := range palette {
			buf.WriteString(fmt.Sprintf("#%d", colorIndex))
			
			// Process this color strip
			x := 0
			for x < width {
				// Build sixel character for this position
				sixelChar := '?'  // Base character (63 in ASCII)
				for bit := 0; bit < 6; bit++ {
					pixelY := y + bit
					if pixelY < height && indices[pixelY][x] == colorIndex {
						sixelChar += (1 << bit)
					}
				}
				
				// Count consecutive identical characters for run-length encoding
				runLength := 1
				for x+runLength < width {
					nextChar := '?'
					for bit := 0; bit < 6; bit++ {
						pixelY := y + bit
						if pixelY < height && indices[pixelY][x+runLength] == colorIndex {
							nextChar += (1 << bit)
						}
					}
					if nextChar != sixelChar {
						break
					}
					runLength++
				}
				
				// Write run-length encoded data
				if runLength > 1 {
					buf.WriteString(fmt.Sprintf("!%d%c", runLength, sixelChar))
				} else {
					buf.WriteByte(byte(sixelChar))
				}
				
				x += runLength
			}
			
			// End this color's data for this strip
			buf.WriteString("$") // Carriage return
		}
		
		// Move to next strip
		if y+6 < height {
			buf.WriteString("-") // Line feed
		}
	}

	// End Sixel sequence
	buf.WriteString("\x1b\\")

	return buf.String(), nil
}

// ClearSixelImages clears Sixel graphics from the terminal
func (p *ImagePreviewer) ClearSixelImages() string {
	// Sixel images are part of the terminal scrollback buffer
	// They get cleared when content scrolls away or terminal is cleared
	// We can potentially overwrite the area with spaces
	return ""
}

// ClearSixelImages clears Sixel graphics from the terminal
func ClearSixelImages() string {
	// Sixel images are cleared when terminal content scrolls away
	return ""
}