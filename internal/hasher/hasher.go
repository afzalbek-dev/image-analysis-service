package hasher

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"math/bits"
	"strconv"

	"github.com/afzalbek-dev/image-analysis-service/internal/domain"
)

// GrayscaleResize scales an image to target width and height using box sampling and converts to grayscale float values [0, 255].
func GrayscaleResize(img image.Image, targetWidth, targetHeight int) [][]float64 {
	bounds := img.Bounds()
	srcWidth := bounds.Dx()
	srcHeight := bounds.Dy()

	result := make([][]float64, targetHeight)
	for y := 0; y < targetHeight; y++ {
		result[y] = make([]float64, targetWidth)
		for x := 0; x < targetWidth; x++ {
			srcX := bounds.Min.X + (x * srcWidth) / targetWidth
			srcY := bounds.Min.Y + (y * srcHeight) / targetHeight
			r, g, b, _ := img.At(srcX, srcY).RGBA()
			// standard luminance calculation: 0.299*R + 0.587*G + 0.114*B
			gray := (0.299*float64(r>>8) + 0.587*float64(g>>8) + 0.114*float64(b>>8))
			result[y][x] = gray
		}
	}
	return result
}

// ComputeAHash calculates Average Hash (64-bit).
func ComputeAHash(img image.Image) uint64 {
	pixels := GrayscaleResize(img, 8, 8)
	var sum float64
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			sum += pixels[y][x]
		}
	}
	avg := sum / 64.0

	var hash uint64
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			if pixels[y][x] >= avg {
				hash |= 1 << (y*8 + x)
			}
		}
	}
	return hash
}

// ComputeDHash calculates Difference Hash (64-bit gradient comparison).
func ComputeDHash(img image.Image) uint64 {
	pixels := GrayscaleResize(img, 9, 8)
	var hash uint64
	bitIndex := 0
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			if pixels[y][x] > pixels[y][x+1] {
				hash |= 1 << bitIndex
			}
			bitIndex++
		}
	}
	return hash
}

// ComputePHash calculates Perceptual Hash using Discrete Cosine Transform (DCT) on 32x32 matrix.
func ComputePHash(img image.Image) uint64 {
	const n = 32
	pixels := GrayscaleResize(img, n, n)

	// 2D DCT-II calculation
	dct := make([][]float64, n)
	for u := 0; u < n; u++ {
		dct[u] = make([]float64, n)
	}

	for u := 0; u < 8; u++ {
		for v := 0; v < 8; v++ {
			var sum float64
			for x := 0; x < n; x++ {
				for y := 0; y < n; y++ {
					cosX := math.Cos(((2*float64(x) + 1) * float64(u) * math.Pi) / (2 * float64(n)))
					cosY := math.Cos(((2*float64(y) + 1) * float64(v) * math.Pi) / (2 * float64(n)))
					sum += pixels[y][x] * cosX * cosY
				}
			}

			cu := 1.0
			if u == 0 {
				cu = 1.0 / math.Sqrt(2)
			}
			cv := 1.0
			if v == 0 {
				cv = 1.0 / math.Sqrt(2)
			}
			dct[u][v] = 0.25 * cu * cv * sum
		}
	}

	// Compute average of lowest 8x8 DCT frequencies, excluding DC coefficient (0,0)
	var sum float64
	count := 0
	for u := 0; u < 8; u++ {
		for v := 0; v < 8; v++ {
			if u == 0 && v == 0 {
				continue
			}
			sum += dct[u][v]
			count++
		}
	}
	avg := sum / float64(count)

	var hash uint64
	bitIndex := 0
	for u := 0; u < 8; u++ {
		for v := 0; v < 8; v++ {
			if u == 0 && v == 0 {
				continue
			}
			if dct[u][v] > avg {
				hash |= 1 << bitIndex
			}
			bitIndex++
			if bitIndex >= 64 {
				break
			}
		}
	}
	return hash
}

// ComputeAll calculates a complete Fingerprint including aHash, dHash, and pHash.
func ComputeAll(img image.Image) domain.HashFingerprint {
	ah := ComputeAHash(img)
	dh := ComputeDHash(img)
	ph := ComputePHash(img)
	return domain.HashFingerprint{
		AHash: FormatHex(ah),
		DHash: FormatHex(dh),
		PHash: FormatHex(ph),
	}
}

// HammingDistance counts differing bits between two 64-bit hashes.
func HammingDistance(h1, h2 uint64) int {
	return bits.OnesCount64(h1 ^ h2)
}

// Similarity calculates ratio between 0.0 (completely dissimilar) and 1.0 (identical).
func Similarity(h1, h2 uint64) float64 {
	dist := HammingDistance(h1, h2)
	return 1.0 - (float64(dist) / 64.0)
}

// FormatHex formats uint64 as a fixed 16-character hexadecimal string.
func FormatHex(h uint64) string {
	return fmt.Sprintf("%016x", h)
}

// ParseHex parses a 16-character hexadecimal string back to uint64.
func ParseHex(s string) (uint64, error) {
	return strconv.ParseUint(s, 16, 64)
}

// Luminance calculates relative perceived brightness of an image.color.
func Luminance(c color.Color) float64 {
	r, g, b, _ := c.RGBA()
	return 0.299*float64(r>>8) + 0.587*float64(g>>8) + 0.114*float64(b>>8)
}
