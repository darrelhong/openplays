package avatar

import (
	"bytes"
	"encoding/binary"
	"errors"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"image/png"
	"testing"
)

func TestProcessorNormalizesJPEGAndPNG(t *testing.T) {
	for _, format := range []string{"jpeg", "png"} {
		t.Run(format, func(t *testing.T) {
			source := image.NewNRGBA(image.Rect(0, 0, 400, 200))
			for y := 0; y < 200; y++ {
				for x := 0; x < 400; x++ {
					source.Set(x, y, color.RGBA{R: uint8(x / 2), G: uint8(y), B: 80, A: 255})
				}
			}
			var input bytes.Buffer
			if format == "jpeg" {
				if err := jpeg.Encode(&input, source, nil); err != nil {
					t.Fatal(err)
				}
			} else if err := png.Encode(&input, source); err != nil {
				t.Fatal(err)
			}

			got, err := (Processor{}).Process(&input)
			if err != nil {
				t.Fatalf("Process: %v", err)
			}
			if got.ContentType != "image/jpeg" || got.Extension != ".jpg" {
				t.Fatalf("metadata = %q, %q", got.ContentType, got.Extension)
			}
			decoded, decodedFormat, err := image.Decode(bytes.NewReader(got.Data))
			if err != nil {
				t.Fatalf("decode result: %v", err)
			}
			if decodedFormat != "jpeg" || decoded.Bounds().Dx() != OutputSize || decoded.Bounds().Dy() != OutputSize {
				t.Fatalf("result = %s %v", decodedFormat, decoded.Bounds())
			}
		})
	}
}

func TestProcessorFlattensTransparencyOntoWhite(t *testing.T) {
	source := image.NewNRGBA(image.Rect(0, 0, 2, 2))
	var input bytes.Buffer
	if err := png.Encode(&input, source); err != nil {
		t.Fatal(err)
	}
	got, err := (Processor{}).Process(&input)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := jpeg.Decode(bytes.NewReader(got.Data))
	if err != nil {
		t.Fatal(err)
	}
	r, g, b, _ := decoded.At(100, 100).RGBA()
	if r < 0xf000 || g < 0xf000 || b < 0xf000 {
		t.Fatalf("transparent pixel became %#x %#x %#x, want white", r, g, b)
	}
}

func TestProcessorAppliesEXIFOrientation(t *testing.T) {
	source := image.NewRGBA(image.Rect(0, 0, 40, 20))
	for y := 0; y < 20; y++ {
		for x := 0; x < 40; x++ {
			if x < 20 {
				source.Set(x, y, color.RGBA{R: 255, A: 255})
			} else {
				source.Set(x, y, color.RGBA{B: 255, A: 255})
			}
		}
	}
	var encoded bytes.Buffer
	if err := jpeg.Encode(&encoded, source, &jpeg.Options{Quality: 100}); err != nil {
		t.Fatal(err)
	}
	raw := withEXIFOrientation(encoded.Bytes(), 6) // rotate 90 degrees clockwise
	got, err := (Processor{}).Process(bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := jpeg.Decode(bytes.NewReader(got.Data))
	if err != nil {
		t.Fatal(err)
	}
	topR, _, topB, _ := decoded.At(128, 32).RGBA()
	bottomR, _, bottomB, _ := decoded.At(128, 224).RGBA()
	if topR <= topB || bottomB <= bottomR {
		t.Fatalf("orientation not applied: top r/b=%d/%d, bottom r/b=%d/%d", topR, topB, bottomR, bottomB)
	}
}

func withEXIFOrientation(jpegData []byte, orientation uint16) []byte {
	payload := []byte{
		'E', 'x', 'i', 'f', 0, 0,
		'I', 'I', 0x2a, 0,
		8, 0, 0, 0,
		1, 0,
		0x12, 0x01, 3, 0, 1, 0, 0, 0,
		0, 0, 0, 0,
		0, 0, 0, 0,
	}
	binary.LittleEndian.PutUint16(payload[24:26], orientation)
	segment := []byte{0xff, 0xe1, 0, 0}
	binary.BigEndian.PutUint16(segment[2:4], uint16(len(payload)+2))
	result := make([]byte, 0, len(jpegData)+len(segment)+len(payload))
	result = append(result, jpegData[:2]...)
	result = append(result, segment...)
	result = append(result, payload...)
	return append(result, jpegData[2:]...)
}

func TestProcessorRejectsInvalidInputs(t *testing.T) {
	var gifInput bytes.Buffer
	if err := gif.Encode(&gifInput, image.NewPaletted(image.Rect(0, 0, 1, 1), color.Palette{color.Black}), nil); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name  string
		input []byte
		want  error
	}{
		{"nil data", nil, ErrInvalidImage},
		{"garbage", []byte("not an image"), ErrInvalidImage},
		{"gif", gifInput.Bytes(), ErrUnsupportedFormat},
		{"too large", make([]byte, MaxInputBytes+1), ErrInputTooLarge},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := (Processor{}).Process(bytes.NewReader(tt.input))
			if !errors.Is(err, tt.want) {
				t.Fatalf("error = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestProcessorRejectsExcessiveDimensions(t *testing.T) {
	source := image.NewGray(image.Rect(0, 0, 5001, 5000))
	var input bytes.Buffer
	if err := png.Encode(&input, source); err != nil {
		t.Fatal(err)
	}
	_, err := (Processor{}).Process(&input)
	if !errors.Is(err, ErrDimensionsTooLarge) {
		t.Fatalf("error = %v, want dimensions too large", err)
	}
}
