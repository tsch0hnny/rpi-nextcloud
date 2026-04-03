package ui

import (
	"bytes"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/tsch0hnny/rpi-nextcloud/internal/style"
)

// ImageMode determines how images are rendered.
type ImageMode int

const (
	ImageModeAuto    ImageMode = iota
	ImageModeSixel
	ImageModeUnicode
	ImageModeNone
)

var currentImageMode = ImageModeAuto
var cacheDir = filepath.Join(os.TempDir(), "nextcloud-installer-images")

// SetImageMode overrides the auto-detected mode.
func SetImageMode(mode ImageMode) {
	currentImageMode = mode
}

// ParseImageMode parses a CLI flag value.
func ParseImageMode(s string) ImageMode {
	switch strings.ToLower(s) {
	case "sixel":
		return ImageModeSixel
	case "unicode":
		return ImageModeUnicode
	case "none":
		return ImageModeNone
	default:
		return ImageModeAuto
	}
}

// DetectSixelSupport checks if the terminal supports sixel graphics.
func DetectSixelSupport() bool {
	// Check known sixel-capable terminals via environment
	term := os.Getenv("TERM")
	termProgram := os.Getenv("TERM_PROGRAM")

	sixelTerms := []string{"foot", "mlterm", "xterm", "contour", "wezterm", "mintty"}
	for _, t := range sixelTerms {
		if strings.Contains(strings.ToLower(term), t) || strings.Contains(strings.ToLower(termProgram), t) {
			return true
		}
	}

	// Check SIXEL env var (some terminals set this)
	if os.Getenv("SIXEL") == "1" {
		return true
	}

	return false
}

// HasChafa checks if chafa is available on the system.
func HasChafa() bool {
	_, err := exec.LookPath("chafa")
	return err == nil
}

// ResolveImageMode determines the effective rendering mode.
func ResolveImageMode() ImageMode {
	if currentImageMode != ImageModeAuto {
		return currentImageMode
	}
	if DetectSixelSupport() {
		return ImageModeSixel
	}
	if HasChafa() {
		return ImageModeUnicode
	}
	return ImageModeNone
}

// ensureCacheDir creates the image cache directory.
func ensureCacheDir() error {
	return os.MkdirAll(cacheDir, 0o755)
}

// cachePathForURL returns a local file path for a cached image.
func cachePathForURL(url string) string {
	// Simple hash: use the filename from URL
	name := filepath.Base(url)
	if name == "" || name == "." || name == "/" {
		name = "image"
	}
	return filepath.Join(cacheDir, name)
}

// DownloadImage downloads an image and caches it locally.
func DownloadImage(url string) (string, error) {
	if err := ensureCacheDir(); err != nil {
		return "", err
	}

	path := cachePathForURL(url)

	// Return cached version if exists
	if _, err := os.Stat(path); err == nil {
		return path, nil
	}

	resp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to download image: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("image download returned status %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read image data: %w", err)
	}

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return "", fmt.Errorf("failed to cache image: %w", err)
	}

	return path, nil
}

// RenderImage renders an image for the terminal at the given width.
func RenderImage(path string, maxWidth int) string {
	mode := ResolveImageMode()
	switch mode {
	case ImageModeSixel:
		return renderSixel(path, maxWidth)
	case ImageModeUnicode:
		return renderChafa(path, maxWidth)
	default:
		return renderPlaceholder(path)
	}
}

// renderSixel uses img2sixel or chafa in sixel mode.
func renderSixel(path string, maxWidth int) string {
	// Try img2sixel first
	if p, err := exec.LookPath("img2sixel"); err == nil {
		cmd := exec.Command(p, "-w", fmt.Sprintf("%d", maxWidth*8), path)
		out, err := cmd.Output()
		if err == nil {
			return string(out)
		}
	}

	// Fall back to chafa in sixel mode
	if HasChafa() {
		w := maxWidth
		if w > 80 {
			w = 80
		}
		cmd := exec.Command("chafa", "-f", "sixels", "--size", fmt.Sprintf("%dx", w), path)
		out, err := cmd.Output()
		if err == nil {
			return string(out)
		}
	}

	return renderChafa(path, maxWidth)
}

// renderChafa renders an image using chafa (unicode/braille).
func renderChafa(path string, maxWidth int) string {
	if !HasChafa() {
		return renderPlaceholder(path)
	}

	w := maxWidth
	if w > 80 {
		w = 80
	}
	h := w / 2 // Approximate aspect ratio for terminal chars
	if h < 10 {
		h = 10
	}

	cmd := exec.Command("chafa",
		"--size", fmt.Sprintf("%dx%d", w, h),
		"--animate", "off",
		path,
	)
	out, err := cmd.Output()
	if err != nil {
		return renderPlaceholder(path)
	}
	return string(out)
}

// renderPlaceholder shows a text-based placeholder for the image.
func renderPlaceholder(path string) string {
	// Try to get image dimensions
	f, err := os.Open(path)
	if err != nil {
		return style.DescriptionStyle.Render("[Image: " + filepath.Base(path) + "]")
	}
	defer f.Close()

	config, _, err := image.DecodeConfig(f)
	if err != nil {
		return style.DescriptionStyle.Render("[Image: " + filepath.Base(path) + "]")
	}

	box := style.BoxStyle.
		Width(40).
		Foreground(style.ColorSubtle).
		Render(fmt.Sprintf(
			"📷 Image: %s\n   Size: %dx%d\n   (install chafa for preview)",
			filepath.Base(path), config.Width, config.Height,
		))
	return box
}

// RenderImageFromURL downloads and renders an image.
func RenderImageFromURL(url string, maxWidth int) string {
	path, err := DownloadImage(url)
	if err != nil {
		return style.DescriptionStyle.Render("[Could not load image]")
	}
	return RenderImage(path, maxWidth)
}

// ImageCaption renders a caption below an image.
func ImageCaption(caption string) string {
	return style.DescriptionStyle.Italic(true).Render("  " + caption)
}

// ImageWithCaption renders an image with a caption below it.
func ImageWithCaption(url string, caption string, maxWidth int) string {
	img := RenderImageFromURL(url, maxWidth)

	var buf bytes.Buffer
	buf.WriteString(img)
	buf.WriteString("\n")
	buf.WriteString(ImageCaption(caption))
	return buf.String()
}
