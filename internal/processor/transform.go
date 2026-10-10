package processor

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"image/png"
	"math"

	"github.com/afzalbek-dev/image-analysis-service/internal/domain"
)

var (
	ErrInvalidDimensions = errors.New("target dimensions must be greater than zero")
	ErrCropOutOfBounds   = errors.New("crop rectangle extends beyond source image bounds")
	ErrUnsupportedFormat = errors.New("unsupported output format (only jpeg and png are supported)")
)

// Resize resizes an image to target dimensions using bilinear interpolation.
func Resize(src image.Image, targetWidth, targetHeight int) (image.Image, error) {
	if targetWidth <= 0 || targetHeight <= 0 {
		return nil, ErrInvalidDimensions
	}

	srcBounds := src.Bounds()
	srcW := srcBounds.Dx()
	srcH := srcBounds.Dy()

	dst := image.NewRGBA(image.Rect(0, 0, targetWidth, targetHeight))

	for y := 0; y < targetHeight; y++ {
		for x := 0; x < targetWidth; x++ {
			srcX := float64(x) * float64(srcW) / float64(targetWidth)
			srcY := float64(y) * float64(srcH) / float64(targetHeight)

			x0 := int(math.Floor(srcX))
			y0 := int(math.Floor(srcY))
			x1 := int(math.Min(float64(x0+1), float64(srcW-1)))
			y1 := int(math.Min(float64(y0+1), float64(srcH-1)))

			dx := srcX - float64(x0)
			dy := srcY - float64(y0)

			c00 := src.At(srcBounds.Min.X+x0, srcBounds.Min.Y+y0)
			c10 := src.At(srcBounds.Min.X+x1, srcBounds.Min.Y+y0)
			c01 := src.At(srcBounds.Min.X+x0, srcBounds.Min.Y+y1)
			c11 := src.At(srcBounds.Min.X+x1, srcBounds.Min.Y+y1)

			r00, g00, b00, a00 := c00.RGBA()
			r10, g10, b10, a10 := c10.RGBA()
			r01, g01, b01, a01 := c01.RGBA()
			r11, g11, b11, a11 := c11.RGBA()

			interp := func(v00, v10, v01, v11 uint32) uint8 {
				top := float64(v00>>8)*(1.0-dx) + float64(v10>>8)*dx
				bottom := float64(v01>>8)*(1.0-dx) + float64(v11>>8)*dx
				res := top*(1.0-dy) + bottom*dy
				return uint8(math.Min(255, math.Max(0, res)))
			}

			dst.Set(x, y, color.RGBA{
				R: interp(r00, r10, r01, r11),
				G: interp(g00, g10, g01, g11),
				B: interp(b00, b10, b01, b11),
				A: interp(a00, a10, a01, a11),
			})
		}
	}

	return dst, nil
}

// Crop extracts a subregion rectangular window from the image.
func Crop(src image.Image, x, y, width, height int) (image.Image, error) {
	if width <= 0 || height <= 0 {
		return nil, ErrInvalidDimensions
	}
	bounds := src.Bounds()
	cropRect := image.Rect(bounds.Min.X+x, bounds.Min.Y+y, bounds.Min.X+x+width, bounds.Min.Y+y+height)

	if !cropRect.In(bounds) {
		return nil, ErrCropOutOfBounds
	}

	dst := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.Draw(dst, dst.Bounds(), src, cropRect.Min, draw.Src)
	return dst, nil
}

// ToGrayscale converts the image to single-channel luminance grayscale.
func ToGrayscale(src image.Image) image.Image {
	bounds := src.Bounds()
	dst := image.NewGray(bounds)
	draw.Draw(dst, bounds, src, bounds.Min, draw.Src)
	return dst
}

// InvertColors inverts the RGB values of each pixel.
func InvertColors(src image.Image) image.Image {
	bounds := src.Bounds()
	w := bounds.Dx()
	h := bounds.Dy()
	dst := image.NewRGBA(image.Rect(0, 0, w, h))

	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			c := src.At(bounds.Min.X+x, bounds.Min.Y+y)
			r, g, b, a := c.RGBA()
			dst.Set(x, y, color.RGBA{
				R: 255 - uint8(r>>8),
				G: 255 - uint8(g>>8),
				B: 255 - uint8(b>>8),
				A: uint8(a >> 8),
			})
		}
	}
	return dst
}

// ApplyTransform runs the transformation pipeline defined by TransformOptions.
func ApplyTransform(src image.Image, opts domain.TransformOptions) (image.Image, error) {
	current := src

	// 1. Crop if specified
	if opts.CropWidth > 0 && opts.CropHeight > 0 {
		cropped, err := Crop(current, opts.CropX, opts.CropY, opts.CropWidth, opts.CropHeight)
		if err != nil {
			return nil, err
		}
		current = cropped
	}

	// 2. Resize if specified
	if opts.Width > 0 && opts.Height > 0 {
		resized, err := Resize(current, opts.Width, opts.Height)
		if err != nil {
			return nil, err
		}
		current = resized
	}

	// 3. Grayscale
	if opts.Grayscale {
		current = ToGrayscale(current)
	}

	// 4. Invert
	if opts.InvertColors {
		current = InvertColors(current)
	}

	return current, nil
}

// Encode converts an image into byte representation (jpeg or png).
func Encode(img image.Image, format string, quality int) ([]byte, error) {
	var buf bytes.Buffer
	switch format {
	case "jpeg", "jpg":
		if quality <= 0 || quality > 100 {
			quality = 85
		}
		opts := &jpeg.Options{Quality: quality}
		if err := jpeg.Encode(&buf, img, opts); err != nil {
			return nil, err
		}
	case "png", "":
		if err := png.Encode(&buf, img); err != nil {
			return nil, err
		}
	default:
		return nil, ErrUnsupportedFormat
	}
	return buf.Bytes(), nil
}
