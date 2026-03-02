package assets

import (
	"embed"
	"fmt"
	"image"
	_ "image/png"
	"os"
	"path"
	"strings"
	"sync"

	_ "golang.org/x/image/webp"
)

type Request struct {
	Key  string
	Path string
}

type Result struct {
	Key   string
	Image image.Image
	Err   error
}

type Loader struct {
	Req  chan Request
	Res  chan Result
	quit chan struct{}

	closeOnce sync.Once
}

func NewLoader() *Loader {
	l := &Loader{
		Req:  make(chan Request, 16),
		Res:  make(chan Result, 16),
		quit: make(chan struct{}),
	}

	go l.loop()

	return l
}

func (l *Loader) Close() {
	l.closeOnce.Do(func() {
		close(l.quit)
	})
}

func (l *Loader) loop() {
	for {
		select {
		case <-l.quit:
			return
		case req := <-l.Req:
			img, err := loadImage(req.Path)
			res := Result{Key: req.Key, Image: img, Err: err}

			// Never block this goroutine forever if the consumer falls behind.
			select {
			case <-l.quit:
				return
			case l.Res <- res:
			default:
			}
		}
	}
}

func loadImage(path string) (image.Image, error) {
	img, err := loadImageFromOS(path)
	if err == nil {
		return img, nil
	}

	// Fallback to embedded assets so loading works regardless of process cwd.
	embeddedCandidates := embeddedPathCandidates(path)
	for _, candidate := range embeddedCandidates {
		img, embErr := loadImageFromEmbedded(candidate)
		if embErr == nil {
			return img, nil
		}
	}

	return nil, fmt.Errorf("load image %q: %w", path, err)
}

func loadImageFromOS(path string) (image.Image, error) {
	f, err := os.Open(path)

	if err != nil {
		return nil, err
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	return img, err
}

func loadImageFromEmbedded(name string) (image.Image, error) {
	f, err := embeddedAssets.Open(name)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	img, _, err := image.Decode(f)
	return img, err
}

func embeddedPathCandidates(name string) []string {
	clean := path.Clean(strings.ReplaceAll(name, "\\", "/"))
	trimmed := strings.TrimPrefix(clean, "./")
	candidates := []string{trimmed}

	knownPrefixes := []string{
		"internal/assets/",
		"assets/",
	}
	for _, prefix := range knownPrefixes {
		if strings.HasPrefix(trimmed, prefix) {
			candidates = append(candidates, strings.TrimPrefix(trimmed, prefix))
		}
	}

	candidates = append(candidates, path.Base(trimmed))

	seen := make(map[string]struct{}, len(candidates))
	uniq := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		if candidate == "." || candidate == "" {
			continue
		}
		if _, ok := seen[candidate]; ok {
			continue
		}
		seen[candidate] = struct{}{}
		uniq = append(uniq, candidate)
	}

	return uniq
}

//go:embed *.webp characters/*.png characters/top_down/*.png characters/walk_spirte/*.png "ui assets/*.png"
var embeddedAssets embed.FS
