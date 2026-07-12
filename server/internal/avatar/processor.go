package avatar

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	_ "image/gif"
	"image/jpeg"
	_ "image/png"
	"io"

	"github.com/disintegration/imaging"
	xdraw "golang.org/x/image/draw"
	_ "golang.org/x/image/webp"
)

const (
	MaxInputBytes          = 5 << 20
	MaxPixels              = 25_000_000
	OutputSize             = 256
	JPEGQuality            = 85
	MaxConcurrentProcesses = 1
)

var processingSlots = make(chan struct{}, MaxConcurrentProcesses)

var (
	ErrInputTooLarge      = errors.New("avatar exceeds maximum file size")
	ErrUnsupportedFormat  = errors.New("avatar must be JPEG or PNG")
	ErrDimensionsTooLarge = errors.New("avatar dimensions are too large")
	ErrInvalidImage       = errors.New("invalid avatar image")
)

type ProcessedImage struct {
	Data        []byte
	ContentType string
	Extension   string
}

type Processor struct{}

func (Processor) Process(input io.Reader) (ProcessedImage, error) {
	if input == nil {
		return ProcessedImage{}, ErrInvalidImage
	}
	processingSlots <- struct{}{}
	defer func() { <-processingSlots }()
	raw, err := io.ReadAll(io.LimitReader(input, MaxInputBytes+1))
	if err != nil {
		return ProcessedImage{}, fmt.Errorf("read avatar: %w", err)
	}
	if len(raw) > MaxInputBytes {
		return ProcessedImage{}, ErrInputTooLarge
	}

	config, format, err := image.DecodeConfig(bytes.NewReader(raw))
	if err != nil {
		return ProcessedImage{}, ErrInvalidImage
	}
	if format != "jpeg" && format != "png" {
		return ProcessedImage{}, ErrUnsupportedFormat
	}
	if config.Width <= 0 || config.Height <= 0 ||
		int64(config.Width) > int64(MaxPixels)/int64(config.Height) {
		return ProcessedImage{}, ErrDimensionsTooLarge
	}

	source, err := imaging.Decode(bytes.NewReader(raw), imaging.AutoOrientation(true))
	if err != nil {
		return ProcessedImage{}, ErrInvalidImage
	}
	normalized := normalize(source)

	var output bytes.Buffer
	if err := jpeg.Encode(&output, normalized, &jpeg.Options{Quality: JPEGQuality}); err != nil {
		return ProcessedImage{}, fmt.Errorf("encode avatar: %w", err)
	}
	return ProcessedImage{Data: output.Bytes(), ContentType: "image/jpeg", Extension: ".jpg"}, nil
}

func normalize(source image.Image) image.Image {
	bounds := source.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	squareSize := min(width, height)
	left := bounds.Min.X + (width-squareSize)/2
	top := bounds.Min.Y + (height-squareSize)/2
	cropBounds := image.Rect(left, top, left+squareSize, top+squareSize)

	resized := image.NewNRGBA(image.Rect(0, 0, OutputSize, OutputSize))
	xdraw.CatmullRom.Scale(resized, resized.Bounds(), source, cropBounds, draw.Over, nil)

	background := image.NewNRGBA(resized.Bounds())
	draw.Draw(background, background.Bounds(), &image.Uniform{C: color.White}, image.Point{}, draw.Src)
	draw.Draw(background, background.Bounds(), resized, image.Point{}, draw.Over)
	return background
}
