package assets

import (
	_ "golang.org/x/image/webp"
	"image"
	_ "image/png"
	"os"
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

func (l *Loader) Close() { close(l.quit) }

func (l *Loader) loop() {
	for {
		select {
		case <-l.quit:
			return
		case req := <-l.Req:
			img, err := loadImage(req.Path)
			l.Res <- Result{Key: req.Key, Image: img, Err: err}
		}
	}
}

func loadImage(path string) (image.Image, error) {
	f, err := os.Open(path)

	if err != nil {
		return nil, err
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	return img, err
}
