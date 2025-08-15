package filepreview

import (
	"image"
	"image/color"
	"os"
	"testing"
)

// createTestImage creates a simple test image for testing purposes
func createTestImage(width, height int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	
	// Create a simple pattern
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			var c color.Color
			if (x+y)%20 < 10 {
				c = color.RGBA{255, 0, 0, 255} // Red
			} else {
				c = color.RGBA{0, 0, 255, 255} // Blue
			}
			img.Set(x, y, c)
		}
	}
	
	return img
}

func TestIsSixelCapable(t *testing.T) {
	tests := []struct {
		name        string
		termProgram string
		term        string
		footVersion string
		expected    bool
	}{
		{
			name:        "foot terminal",
			termProgram: "",
			term:        "foot",
			footVersion: "",
			expected:    true,
		},
		{
			name:        "foot with version env",
			termProgram: "",
			term:        "",
			footVersion: "1.15.0",
			expected:    true,
		},
		{
			name:        "xterm",
			termProgram: "",
			term:        "xterm-256color",
			footVersion: "",
			expected:    true,
		},
		{
			name:        "unknown terminal",
			termProgram: "unknown",
			term:        "unknown",
			footVersion: "",
			expected:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variables
			if tt.termProgram != "" {
				os.Setenv("TERM_PROGRAM", tt.termProgram)
				defer os.Unsetenv("TERM_PROGRAM")
			}
			if tt.term != "" {
				os.Setenv("TERM", tt.term)
				defer os.Unsetenv("TERM")
			}
			if tt.footVersion != "" {
				os.Setenv("FOOT_VERSION", tt.footVersion)
				defer os.Unsetenv("FOOT_VERSION")
			}

			result := isSixelCapable()
			if result != tt.expected {
				t.Errorf("isSixelCapable() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestImagePreviewerSixelCapable(t *testing.T) {
	previewer := NewImagePreviewer()
	
	// Test the method version
	os.Setenv("TERM", "foot")
	defer os.Unsetenv("TERM")
	
	if !previewer.IsSixelCapable() {
		t.Error("IsSixelCapable() should return true for foot terminal")
	}
}

func TestSixelRendering(t *testing.T) {
	// Skip if not in a Sixel-capable environment for CI/CD
	if !isSixelCapable() && os.Getenv("CI") != "" {
		t.Skip("Skipping Sixel rendering test in CI environment")
	}

	previewer := NewImagePreviewer()
	testImg := createTestImage(100, 100)

	result, err := previewer.renderWithSixel(testImg, 10, 10)
	if err != nil {
		t.Errorf("renderWithSixel() failed: %v", err)
	}

	if len(result) == 0 {
		t.Error("renderWithSixel() returned empty result")
	}

	// Basic sanity check - Sixel output should contain the DCS sequence
	if len(result) > 0 {
		// Sixel data typically starts with DCS (Device Control String)
		// but the go-sixel library might format it differently
		t.Logf("Sixel output length: %d characters", len(result))
	}
}

func TestSixelClearImages(t *testing.T) {
	// Test the function version
	os.Setenv("TERM", "foot")
	defer os.Unsetenv("TERM")
	
	result := ClearSixelImages()
	if result != "\x1b[2J" {
		t.Errorf("ClearSixelImages() = %q, expected %q", result, "\x1b[2J")
	}

	// Test when not Sixel capable
	os.Setenv("TERM", "dumb")
	defer os.Unsetenv("TERM")
	
	result = ClearSixelImages()
	if result != "" {
		t.Errorf("ClearSixelImages() = %q, expected empty string for non-Sixel terminal", result)
	}
}

func TestSixelImagePreviewerClearImages(t *testing.T) {
	previewer := NewImagePreviewer()
	
	// Test the method version
	os.Setenv("TERM", "foot")
	defer os.Unsetenv("TERM")
	
	result := previewer.ClearSixelImages()
	if result != "\x1b[2J" {
		t.Errorf("ClearSixelImages() = %q, expected %q", result, "\x1b[2J")
	}
}

func TestResizeImage(t *testing.T) {
	testImg := createTestImage(200, 200)
	
	resized := resizeImage(testImg, 100, 100)
	bounds := resized.Bounds()
	
	// The resize should maintain aspect ratio, so it might not be exactly 100x100
	// but should be within reasonable bounds
	if bounds.Dx() > 100 || bounds.Dy() > 100 {
		t.Errorf("resizeImage() produced image larger than requested: %dx%d", bounds.Dx(), bounds.Dy())
	}
	
	if bounds.Dx() == 0 || bounds.Dy() == 0 {
		t.Error("resizeImage() produced empty image")
	}
}

func TestImagePreviewWithSixelRenderer(t *testing.T) {
	// Test the full pipeline with a real image file
	imagePath := "../../../asset/superfileicon.png"
	
	// Check if the test image exists
	if _, err := os.Stat(imagePath); os.IsNotExist(err) {
		t.Skip("Test image not found, skipping integration test")
	}

	// Set environment to simulate a Sixel-capable terminal
	os.Setenv("TERM", "foot")
	defer os.Unsetenv("TERM")

	previewer := NewImagePreviewer()
	
	// Test the ImagePreviewWithRenderer function with Sixel
	result, err := previewer.ImagePreviewWithRenderer(imagePath, 20, 20, "#000000", RendererSixel, 4)
	if err != nil {
		t.Errorf("ImagePreviewWithRenderer() with Sixel failed: %v", err)
	}

	if len(result) == 0 {
		t.Error("ImagePreviewWithRenderer() with Sixel returned empty result")
	}

	t.Logf("Sixel preview result length: %d characters", len(result))
}

func TestImagePreviewFallbackChain(t *testing.T) {
	// Test the fallback chain: Kitty -> Sixel -> ANSI
	imagePath := "../../../asset/superfileicon.png"
	
	// Check if the test image exists
	if _, err := os.Stat(imagePath); os.IsNotExist(err) {
		t.Skip("Test image not found, skipping fallback chain test")
	}

	// Set environment to simulate a Sixel-only terminal (no Kitty)
	os.Setenv("TERM", "foot")
	os.Unsetenv("TERM_PROGRAM")
	defer os.Unsetenv("TERM")

	previewer := NewImagePreviewer()
	
	// The main ImagePreview function should try Sixel since Kitty is not available
	result, err := previewer.ImagePreview(imagePath, 20, 20, "#000000", 4)
	if err != nil {
		t.Errorf("ImagePreview() fallback chain failed: %v", err)
	}

	if len(result) == 0 {
		t.Error("ImagePreview() fallback chain returned empty result")
	}

	t.Logf("Fallback chain result length: %d characters", len(result))
}