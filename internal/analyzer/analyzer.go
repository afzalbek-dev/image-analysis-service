package analyzer

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"sort"
	"time"

	"github.com/afzalbek-dev/image-analysis-service/internal/domain"
	"github.com/afzalbek-dev/image-analysis-service/internal/hasher"
)

// ExtractMetadata extracts structural attributes and format metadata from an image.
func ExtractMetadata(img image.Image, format string, rawBytesLen int64) domain.ImageMetadata {
	bounds := img.Bounds()
	w := bounds.Dx()
	h := bounds.Dy()

	aspectRatio := 0.0
	if h > 0 {
		aspectRatio = math.Round((float64(w)/float64(h))*100) / 100
	}
	megapixels := math.Round((float64(w*h)/1000000.0)*100) / 100

	colorModel := "RGBA"
	switch img.ColorModel() {
	case color.YCbCrModel:
		colorModel = "YCbCr"
	case color.GrayModel:
		colorModel = "Grayscale"
	case color.CMYKModel:
		colorModel = "CMYK"
	}

	return domain.ImageMetadata{
		Format:         format,
		Width:          w,
		Height:         h,
		AspectRatio:    aspectRatio,
		Megapixels:     megapixels,
		ColorModel:     colorModel,
		EstimatedBytes: rawBytesLen,
	}
}

// AnalyzeColors computes color distribution, dominant palette, and lighting dynamics.
func AnalyzeColors(img image.Image) domain.ColorProfile {
	bounds := img.Bounds()
	w := bounds.Dx()
	h := bounds.Dy()

	stepX := int(math.Max(1, float64(w)/64.0))
	stepY := int(math.Max(1, float64(h)/64.0))

	type bucketKey struct {
		r, g, b uint8
	}
	histogram := make(map[bucketKey]int)
	var totalBrightness float64
	var brightnessVals []float64
	totalSamples := 0
	isGrayscale := true

	for y := bounds.Min.Y; y < bounds.Max.Y; y += stepY {
		for x := bounds.Min.X; x < bounds.Max.X; x += stepX {
			c := img.At(x, y)
			r, g, b, _ := c.RGBA()
			r8 := uint8(r >> 8)
			g8 := uint8(g >> 8)
			b8 := uint8(b >> 8)

			if math.Abs(float64(r8)-float64(g8)) > 8 || math.Abs(float64(g8)-float64(b8)) > 8 {
				isGrayscale = false
			}

			// Quantize into 32-step color bins
			qr := (r8 / 32) * 32
			qg := (g8 / 32) * 32
			qb := (b8 / 32) * 32
			histogram[bucketKey{qr, qg, qb}]++

			lum := (0.299*float64(r8) + 0.587*float64(g8) + 0.114*float64(b8)) / 255.0
			totalBrightness += lum
			brightnessVals = append(brightnessVals, lum)
			totalSamples++
		}
	}

	avgBrightness := 0.5
	contrastRatio := 0.0
	if totalSamples > 0 {
		avgBrightness = totalBrightness / float64(totalSamples)

		var varianceSum float64
		for _, b := range brightnessVals {
			varianceSum += (b - avgBrightness) * (b - avgBrightness)
		}
		contrastRatio = math.Sqrt(varianceSum / float64(totalSamples))
	}

	// Sort dominant color buckets
	type countItem struct {
		key   bucketKey
		count int
	}
	var sortedBuckets []countItem
	for k, cnt := range histogram {
		sortedBuckets = append(sortedBuckets, countItem{k, cnt})
	}
	sort.Slice(sortedBuckets, func(i, j int) bool {
		return sortedBuckets[i].count > sortedBuckets[j].count
	})

	var dominant []domain.DominantColor
	topLimit := int(math.Min(5, float64(len(sortedBuckets))))
	for i := 0; i < topLimit; i++ {
		b := sortedBuckets[i]
		pct := 0.0
		if totalSamples > 0 {
			pct = math.Round((float64(b.count)/float64(totalSamples))*1000) / 10
		}
		hexStr := fmt.Sprintf("#%02X%02X%02X", b.key.r, b.key.g, b.key.b)
		dominant = append(dominant, domain.DominantColor{
			Hex:        hexStr,
			RGB:        [3]uint8{b.key.r, b.key.g, b.key.b},
			Percentage: pct,
		})
	}

	return domain.ColorProfile{
		DominantColors:    dominant,
		AverageBrightness: math.Round(avgBrightness*100) / 100,
		ContrastRatio:     math.Round(contrastRatio*100) / 100,
		IsGrayscale:       isGrayscale,
	}
}

// AnalyzeQuality computes sharpness (Laplacian variance proxy) and exposure health.
func AnalyzeQuality(img image.Image) domain.QualityMetrics {
	// Scale down for rapid Laplacian calculation (64x64)
	const size = 64
	pixels := hasher.GrayscaleResize(img, size, size)

	var laplacianVals []float64
	var sumLaplacian float64

	for y := 1; y < size-1; y++ {
		for x := 1; x < size-1; x++ {
			// Laplacian kernel:
			//   0,  1,  0
			//   1, -4,  1
			//   0,  1,  0
			val := pixels[y-1][x] + pixels[y+1][x] + pixels[y][x-1] + pixels[y][x+1] - (4.0 * pixels[y][x])
			laplacianVals = append(laplacianVals, val)
			sumLaplacian += val
		}
	}

	n := float64(len(laplacianVals))
	sharpness := 0.0
	if n > 0 {
		mean := sumLaplacian / n
		var varianceSum float64
		for _, v := range laplacianVals {
			diff := v - mean
			varianceSum += diff * diff
		}
		sharpness = varianceSum / n
	}

	isBlurry := sharpness < 40.0

	// Exposure evaluation
	colorProf := AnalyzeColors(img)
	exposure := "well_balanced"
	if colorProf.AverageBrightness < 0.20 {
		exposure = "under_exposed"
	} else if colorProf.AverageBrightness > 0.85 {
		exposure = "over_exposed"
	}

	return domain.QualityMetrics{
		SharpnessScore:   math.Round(sharpness*100) / 100,
		IsBlurry:         isBlurry,
		ExposureStatus:   exposure,
		DynamicRangeBits: 8,
	}
}

// Analyze runs comprehensive analysis on an image.
func Analyze(img image.Image, format string, rawBytesLen int64) domain.AnalysisResult {
	start := time.Now()

	meta := ExtractMetadata(img, format, rawBytesLen)
	colorProf := AnalyzeColors(img)
	quality := AnalyzeQuality(img)
	fp := hasher.ComputeAll(img)

	duration := time.Since(start).Milliseconds()

	return domain.AnalysisResult{
		Metadata:         meta,
		Color:            colorProf,
		Quality:          quality,
		Fingerprint:      fp,
		AnalyzedAt:       time.Now().UTC(),
		ProcessingTimeMs: duration,
	}
}
