package ui

import (
	"bytes"
	"context"
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
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
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

const imageDownloadTimeout = 15 * time.Second

var currentImageMode = ImageModeAuto
var cacheDir = filepath.Join(os.TempDir(), "nextcloud-installer-images")

// image cache for async loading
var (
	imageCache   = make(map[string]string) // url -> rendered string
	imageCacheMu sync.RWMutex
)

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
	term := os.Getenv("TERM")
	termProgram := os.Getenv("TERM_PROGRAM")

	sixelTerms := []string{"foot", "mlterm", "xterm", "contour", "wezterm", "mintty"}
	for _, t := range sixelTerms {
		if strings.Contains(strings.ToLower(term), t) || strings.Contains(strings.ToLower(termProgram), t) {
			return true
		}
	}

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

func ensureCacheDir() error {
	return os.MkdirAll(cacheDir, 0o755)
}

func cachePathForURL(url string) string {
	name := filepath.Base(url)
	if name == "" || name == "." || name == "/" {
		name = "image"
	}
	return filepath.Join(cacheDir, name)
}

// ImageLoadedMsg is sent when an async image download+render completes.
type ImageLoadedMsg struct {
	URL      string
	Rendered string
	Err      error
}

// LoadImageAsync starts downloading and rendering an image in the background.
func LoadImageAsync(url string, maxWidth int) tea.Cmd {
	return func() tea.Msg {
		// Check if already cached in render cache
		imageCacheMu.RLock()
		if rendered, ok := imageCache[url]; ok {
			imageCacheMu.RUnlock()
			return ImageLoadedMsg{URL: url, Rendered: rendered}
		}
		imageCacheMu.RUnlock()

		// Download with timeout
		path, err := downloadImageWithTimeout(url)
		if err != nil {
			return ImageLoadedMsg{URL: url, Err: err}
		}

		// Render
		rendered := renderImage(path, maxWidth)

		// Cache the render
		imageCacheMu.Lock()
		imageCache[url] = rendered
		imageCacheMu.Unlock()

		return ImageLoadedMsg{URL: url, Rendered: rendered}
	}
}

// GetCachedImage returns the cached rendered image or a placeholder.
func GetCachedImage(url string) string {
	imageCacheMu.RLock()
	defer imageCacheMu.RUnlock()
	if rendered, ok := imageCache[url]; ok {
		return rendered
	}
	return style.DescriptionStyle.Render("  Loading image...")
}

func downloadImageWithTimeout(url string) (string, error) {
	if err := ensureCacheDir(); err != nil {
		return "", err
	}

	path := cachePathForURL(url)

	// Return cached file if exists
	if _, err := os.Stat(path); err == nil {
		return path, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), imageDownloadTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("download failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("status %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read failed: %w", err)
	}

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return "", fmt.Errorf("cache write failed: %w", err)
	}

	return path, nil
}

func renderImage(path string, maxWidth int) string {
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

func renderSixel(path string, maxWidth int) string {
	if p, err := exec.LookPath("img2sixel"); err == nil {
		cmd := exec.Command(p, "-w", fmt.Sprintf("%d", maxWidth*8), path)
		out, err := cmd.Output()
		if err == nil {
			return string(out)
		}
	}

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

func renderChafa(path string, maxWidth int) string {
	if !HasChafa() {
		return renderPlaceholder(path)
	}

	w := maxWidth
	if w > 80 {
		w = 80
	}
	h := w / 2
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

func renderPlaceholder(path string) string {
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
			"  Image: %s\n   Size: %dx%d\n   (install chafa for preview)",
			filepath.Base(path), config.Width, config.Height,
		))
	return box
}

// ImageCaption renders a caption below an image.
func ImageCaption(caption string) string {
	return style.DescriptionStyle.Italic(true).Render("  " + caption)
}

// ImageWithCaption renders a cached image with a caption, or a loading placeholder.
func ImageWithCaption(url string, caption string, _ int) string {
	img := GetCachedImage(url)

	var buf bytes.Buffer
	buf.WriteString(img)
	buf.WriteString("\n")
	buf.WriteString(ImageCaption(caption))
	return buf.String()
}
