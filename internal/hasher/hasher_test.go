package hasher

import (
	"image/color"
	"testing"

	"github.com/afzalbek-dev/image-analysis-service/testutil"
)

func TestHashesIdenticalImages(t *testing.T) {
	img1 := testutil.CreateCheckerboardImage(128, 128, 16)
	img2 := testutil.CreateCheckerboardImage(128, 128, 16)

	ah1 := ComputeAHash(img1)
	ah2 := ComputeAHash(img2)
	if ah1 != ah2 {
		t.Errorf("aHash must be identical for identical images: %x != %x", ah1, ah2)
	}

	dh1 := ComputeDHash(img1)
	dh2 := ComputeDHash(img2)
	if dh1 != dh2 {
		t.Errorf("dHash must be identical for identical images: %x != %x", dh1, dh2)
	}

	ph1 := ComputePHash(img1)
	ph2 := ComputePHash(img2)
	if ph1 != ph2 {
		t.Errorf("pHash must be identical for identical images: %x != %x", ph1, ph2)
	}

	sim := Similarity(ph1, ph2)
	if sim != 1.0 {
		t.Errorf("similarity must be 1.0 for identical images, got %f", sim)
	}
}

func TestHashesDissimilarImages(t *testing.T) {
	solidWhite := testutil.CreateSolidImage(64, 64, color.White)
	gradient := testutil.CreateGradientImage(64, 64)

	ah1 := ComputeAHash(solidWhite)
	ah2 := ComputeAHash(gradient)

	dist := HammingDistance(ah1, ah2)
	if dist == 0 {
		t.Errorf("hamming distance should not be 0 for dissimilar images")
	}

	sim := Similarity(ah1, ah2)
	if sim > 0.95 {
		t.Errorf("similarity should be low for dissimilar images, got %f", sim)
	}
}

func TestHexFormattingAndParsing(t *testing.T) {
	val := uint64(0x123456789abcdef0)
	hexStr := FormatHex(val)
	if hexStr != "123456789abcdef0" {
		t.Errorf("expected 123456789abcdef0, got %s", hexStr)
	}

	parsed, err := ParseHex(hexStr)
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}
	if parsed != val {
		t.Errorf("expected %x, got %x", val, parsed)
	}
}

func TestScaleInvariance(t *testing.T) {
	imgSmall := testutil.CreateCheckerboardImage(64, 64, 8)
	imgLarge := testutil.CreateCheckerboardImage(256, 256, 32)

	phSmall := ComputePHash(imgSmall)
	phLarge := ComputePHash(imgLarge)

	sim := Similarity(phSmall, phLarge)
	if sim < 0.80 {
		t.Errorf("expected scale-invariant similarity >= 0.80, got %f", sim)
	}
}
