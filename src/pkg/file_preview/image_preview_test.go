package filepreview

import (
	"image"
	"image/color"
	"os"
	"testing"
)

// createTestImage creates a simple test image for testing
func createTestImage(width, height int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	
	// Create a simple gradient pattern
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			r := uint8((x * 255) / width)
			g := uint8((y * 255) / height)
			b := uint8(128)
			img.Set(x, y, color.RGBA{r, g, b, 255})
		}
	}
	
	return img
}

func TestTerminalDetection(t *testing.T) {
	tests := []struct {
		name         string
		termProgram  string
		term         string
		expectKitty  bool
		expectITerm2 bool
		expectSixel  bool
	}{
		{
			name:         "Kitty terminal",
			termProgram:  "",
			term:         "xterm-kitty",
			expectKitty:  true,
			expectITerm2: false,
			expectSixel:  true, // xterm-kitty actually supports Sixel too
		},
		{
			name:         "iTerm2",
			termProgram:  "iTerm.app",
			term:         "",
			expectKitty:  false,
			expectITerm2: true,
			expectSixel:  true,
		},
		{
			name:         "WezTerm",
			termProgram:  "WezTerm",
			term:         "",
			expectKitty:  true,
			expectITerm2: true,
			expectSixel:  true,
		},
		{
			name:         "VSCode",
			termProgram:  "vscode",
			term:         "",
			expectKitty:  false,
			expectITerm2: true,
			expectSixel:  true,
		},
		{
			name:         "foot terminal",
			termProgram:  "foot",
			term:         "foot",
			expectKitty:  false,
			expectITerm2: false,
			expectSixel:  true,
		},
		{
			name:         "xterm",
			termProgram:  "",
			term:         "xterm-256color",
			expectKitty:  false,
			expectITerm2: false,
			expectSixel:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original environment
			origTermProgram := os.Getenv("TERM_PROGRAM")
			origTerm := os.Getenv("TERM")
			
			// Set test environment
			if tt.termProgram != "" {
				os.Setenv("TERM_PROGRAM", tt.termProgram)
			} else {
				os.Unsetenv("TERM_PROGRAM")
			}
			if tt.term != "" {
				os.Setenv("TERM", tt.term)
			} else {
				os.Unsetenv("TERM")
			}
			
			// Test terminal detection
			kittyCapable := isKittyCapable()
			iterm2Capable := isITerm2Capable()
			sixelCapable := isSixelCapable()
			
			if kittyCapable != tt.expectKitty {
				t.Errorf("Kitty detection failed: expected %v, got %v", tt.expectKitty, kittyCapable)
			}
			if iterm2Capable != tt.expectITerm2 {
				t.Errorf("iTerm2 detection failed: expected %v, got %v", tt.expectITerm2, iterm2Capable)
			}
			if sixelCapable != tt.expectSixel {
				t.Errorf("Sixel detection failed: expected %v, got %v", tt.expectSixel, sixelCapable)
			}
			
			// Restore original environment
			if origTermProgram != "" {
				os.Setenv("TERM_PROGRAM", origTermProgram)
			} else {
				os.Unsetenv("TERM_PROGRAM")
			}
			if origTerm != "" {
				os.Setenv("TERM", origTerm)
			} else {
				os.Unsetenv("TERM")
			}
		})
	}
}

func TestImagePreviewerCreation(t *testing.T) {
	previewer := NewImagePreviewer()
	if previewer == nil {
		t.Fatal("Failed to create ImagePreviewer")
	}
	
	// Test that terminal capabilities are initialized
	cellSize := previewer.terminalCap.GetTerminalCellSize()
	if cellSize.PixelsPerColumn <= 0 || cellSize.PixelsPerRow <= 0 {
		t.Errorf("Invalid cell size: %+v", cellSize)
	}
}

func TestImageRendererValues(t *testing.T) {
	// Test that our renderer constants are properly defined
	if RendererANSI != 0 {
		t.Errorf("RendererANSI should be 0, got %d", RendererANSI)
	}
	if RendererKitty != 1 {
		t.Errorf("RendererKitty should be 1, got %d", RendererKitty)
	}
	if RendererITerm2 != 2 {
		t.Errorf("RendererITerm2 should be 2, got %d", RendererITerm2)
	}
	if RendererSixel != 3 {
		t.Errorf("RendererSixel should be 3, got %d", RendererSixel)
	}
}

func TestColorQuantization(t *testing.T) {
	img := createTestImage(10, 10)
	quantizer := NewColorQuantizer(8)
	
	indices, err := quantizer.quantizeImage(img)
	if err != nil {
		t.Fatalf("Color quantization failed: %v", err)
	}
	
	if len(indices) != 10 {
		t.Errorf("Expected 10 rows, got %d", len(indices))
	}
	
	if len(indices[0]) != 10 {
		t.Errorf("Expected 10 columns, got %d", len(indices[0]))
	}
	
	if len(quantizer.palette) == 0 {
		t.Errorf("Palette should not be empty")
	}
	
	if len(quantizer.palette) > 8 {
		t.Errorf("Palette should not exceed max colors: got %d", len(quantizer.palette))
	}
}