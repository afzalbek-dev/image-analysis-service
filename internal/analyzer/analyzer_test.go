package analyzer

import (
	"image/color"
	"testing"

	"github.com/afzalbek-dev/image-analysis-service/testutil"
)

func TestExtractMetadata(t *testing.T) {
	img := testutil.CreateSolidImage(200, 100, color.RGBA{R: 255, G: 0, B: 0, A: 255})
	meta := ExtractMetadata(img, "png", 4096)

	if meta.Width != 200 || meta.Height != 100 {
		t.Errorf("expected 200x100, got %dx%d", meta.Width, meta.Height)
	}
	if meta.AspectRatio != 2.0 {
		t.Errorf("expected aspect ratio 2.0, got %f", meta.AspectRatio)
	}
	if meta.Format != "png" {
		t.Errorf("expected format png, got %s", meta.Format)
	}
	if meta.EstimatedBytes != 4096 {
		t.Errorf("expected 4096 bytes, got %d", meta.EstimatedBytes)
	}
}

func TestAnalyzeColorsSolidAndGrayscale(t *testing.T) {
	white := testutil.CreateSolidImage(50, 50, color.White)
	profWhite := AnalyzeColors(white)

	if !profWhite.IsGrayscale {
		t.Errorf("white image must be identified as grayscale")
	}
	if profWhite.AverageBrightness < 0.95 {
		t.Errorf("white image brightness should be >= 0.95, got %f", profWhite.AverageBrightness)
	}
	if len(profWhite.DominantColors) == 0 {
		t.Fatalf("expected dominant color entries")
	}

	red := testutil.CreateSolidImage(50, 50, color.RGBA{R: 240, G: 20, B: 20, A: 255})
	profRed := AnalyzeColors(red)
	if profRed.IsGrayscale {
		t.Errorf("red image should not be identified as grayscale")
	}
}

func TestAnalyzeQualitySharpness(t *testing.T) {
	solid := testutil.CreateSolidImage(100, 100, color.Black)
	qualSolid := AnalyzeQuality(solid)

	if qualSolid.SharpnessScore != 0 {
		t.Errorf("solid image should have 0 sharpness score, got %f", qualSolid.SharpnessScore)
	}
	if !qualSolid.IsBlurry {
		t.Errorf("solid image should be categorized as blurry/flat")
	}

	checker := testutil.CreateCheckerboardImage(128, 128, 8)
	qualChecker := AnalyzeQuality(checker)

	if qualChecker.SharpnessScore <= 50.0 {
		t.Errorf("high-contrast checkerboard should have high sharpness, got %f", qualChecker.SharpnessScore)
	}
	if qualChecker.IsBlurry {
		t.Errorf("checkerboard should not be marked blurry")
	}
}

func TestAnalyzeComprehensive(t *testing.T) {
	grad := testutil.CreateGradientImage(100, 80)
	res := Analyze(grad, "jpeg", 8192)

	if res.Metadata.Width != 100 || res.Metadata.Height != 80 {
		t.Errorf("expected 100x80 metadata, got %dx%d", res.Metadata.Width, res.Metadata.Height)
	}
	if res.Fingerprint.PHash == "" || res.Fingerprint.AHash == "" {
		t.Errorf("expected non-empty hash fingerprints")
	}
	if res.AnalyzedAt.IsZero() {
		t.Errorf("expected non-zero analyzed timestamp")
	}
}
