package duplicates

import (
	"testing"

	"github.com/afzalbek-dev/image-analysis-service/internal/domain"
	"github.com/afzalbek-dev/image-analysis-service/internal/hasher"
	"github.com/afzalbek-dev/image-analysis-service/testutil"
)

func TestDuplicateServiceRegistrationAndLookup(t *testing.T) {
	svc := NewService()

	img := testutil.CreateCheckerboardImage(100, 100, 10)
	fp := hasher.ComputeAll(img)
	meta := domain.ImageMetadata{Format: "png", Width: 100, Height: 100}

	rec := svc.Register("img-101", fp, meta)
	if rec.ID != "img-101" {
		t.Fatalf("expected ID img-101, got %s", rec.ID)
	}

	if svc.Count() != 1 {
		t.Fatalf("expected count 1, got %d", svc.Count())
	}

	// Exact duplicate check
	res := svc.Check(fp, 10, 0.85)
	if !res.IsDuplicate {
		t.Fatalf("expected is_duplicate to be true for exact match")
	}
	if res.BestMatch == nil || res.BestMatch.ID != "img-101" {
		t.Fatalf("expected best match img-101")
	}
	if res.BestMatch.SimilarityScore != 1.0 {
		t.Fatalf("expected similarity score 1.0, got %f", res.BestMatch.SimilarityScore)
	}
}

func TestDuplicateServiceDifferentImage(t *testing.T) {
	svc := NewService()

	img1 := testutil.CreateCheckerboardImage(100, 100, 10)
	fp1 := hasher.ComputeAll(img1)
	meta1 := domain.ImageMetadata{Format: "png", Width: 100, Height: 100}
	svc.Register("img-1", fp1, meta1)

	img2 := testutil.CreateGradientImage(100, 100)
	fp2 := hasher.ComputeAll(img2)

	res := svc.Check(fp2, 5, 0.90)
	if res.IsDuplicate {
		t.Errorf("dissimilar image should not be identified as duplicate")
	}
}

func TestDuplicateServiceDeletion(t *testing.T) {
	svc := NewService()
	meta := domain.ImageMetadata{Format: "png", Width: 50, Height: 50}
	svc.Register("temp-1", domain.HashFingerprint{PHash: "0000000000000000"}, meta)

	if !svc.Delete("temp-1") {
		t.Errorf("delete should return true for existing item")
	}
	if svc.Delete("temp-1") {
		t.Errorf("delete should return false for nonexistent item")
	}
	if svc.Count() != 0 {
		t.Errorf("expected count 0, got %d", svc.Count())
	}
}
