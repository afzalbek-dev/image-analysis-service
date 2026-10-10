package processor

import (
	"image/color"
	"testing"

	"github.com/afzalbek-dev/image-analysis-service/internal/domain"
	"github.com/afzalbek-dev/image-analysis-service/testutil"
)

func TestResize(t *testing.T) {
	src := testutil.CreateSolidImage(100, 100, color.RGBA{R: 100, G: 150, B: 200, A: 255})
	resized, err := Resize(src, 50, 40)
	if err != nil {
		t.Fatalf("unexpected resize error: %v", err)
	}

	b := resized.Bounds()
	if b.Dx() != 50 || b.Dy() != 40 {
		t.Errorf("expected 50x40, got %dx%d", b.Dx(), b.Dy())
	}

	_, errInvalid := Resize(src, 0, 50)
	if errInvalid != ErrInvalidDimensions {
		t.Errorf("expected ErrInvalidDimensions, got %v", errInvalid)
	}
}

func TestCrop(t *testing.T) {
	src := testutil.CreateCheckerboardImage(100, 100, 10)
	cropped, err := Crop(src, 10, 10, 30, 30)
	if err != nil {
		t.Fatalf("unexpected crop error: %v", err)
	}

	b := cropped.Bounds()
	if b.Dx() != 30 || b.Dy() != 30 {
		t.Errorf("expected 30x30, got %dx%d", b.Dx(), b.Dy())
	}

	_, errOOB := Crop(src, 80, 80, 50, 50)
	if errOOB != ErrCropOutOfBounds {
		t.Errorf("expected ErrCropOutOfBounds, got %v", errOOB)
	}
}

func TestToGrayscaleAndInvert(t *testing.T) {
	src := testutil.CreateSolidImage(10, 10, color.RGBA{R: 200, G: 0, B: 0, A: 255})
	gray := ToGrayscale(src)
	r, g, b, _ := gray.At(0, 0).RGBA()
	if r != g || g != b {
		t.Errorf("expected grayscale pixel where R==G==B, got %d,%d,%d", r, g, b)
	}

	inv := InvertColors(src)
	ir, ig, ib, _ := inv.At(0, 0).RGBA()
	if uint8(ir>>8) != 55 { // 255 - 200 = 55
		t.Errorf("expected inverted red to be 55, got %d", ir>>8)
	}
	if uint8(ig>>8) != 255 { // 255 - 0 = 255
		t.Errorf("expected inverted green to be 255, got %d", ig>>8)
	}
	if uint8(ib>>8) != 255 {
		t.Errorf("expected inverted blue to be 255, got %d", ib>>8)
	}
}

func TestApplyTransformAndEncode(t *testing.T) {
	src := testutil.CreateGradientImage(100, 100)
	opts := domain.TransformOptions{
		CropX:        10,
		CropY:        10,
		CropWidth:    80,
		CropHeight:   80,
		Width:        40,
		Height:       40,
		Grayscale:    true,
		InvertColors: true,
		OutputFormat: "jpeg",
		Quality:      90,
	}

	transformed, err := ApplyTransform(src, opts)
	if err != nil {
		t.Fatalf("unexpected transform error: %v", err)
	}
	if transformed.Bounds().Dx() != 40 || transformed.Bounds().Dy() != 40 {
		t.Errorf("expected 40x40 final bounds")
	}

	jpegBytes, err := Encode(transformed, "jpeg", 90)
	if err != nil {
		t.Fatalf("unexpected encode jpeg error: %v", err)
	}
	if len(jpegBytes) == 0 {
		t.Errorf("expected non-empty jpeg bytes")
	}

	pngBytes, err := Encode(transformed, "png", 0)
	if err != nil {
		t.Fatalf("unexpected encode png error: %v", err)
	}
	if len(pngBytes) == 0 {
		t.Errorf("expected non-empty png bytes")
	}

	_, errBadFormat := Encode(transformed, "bmp", 0)
	if errBadFormat != ErrUnsupportedFormat {
		t.Errorf("expected ErrUnsupportedFormat, got %v", errBadFormat)
	}
}
