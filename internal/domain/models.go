package domain

import (
	"time"
)

// ImageMetadata represents structural and format characteristics of an image.
type ImageMetadata struct {
	Format         string  `json:"format"`
	Width          int     `json:"width"`
	Height         int     `json:"height"`
	AspectRatio    float64 `json:"aspect_ratio"`
	Megapixels     float64 `json:"megapixels"`
	ColorModel     string  `json:"color_model"`
	EstimatedBytes int64   `json:"estimated_bytes"`
}

// DominantColor represents a primary color cluster with percentage coverage.
type DominantColor struct {
	Hex        string  `json:"hex"`
	RGB        [3]uint8 `json:"rgb"`
	Percentage float64 `json:"percentage"`
}

// ColorProfile represents color metrics including distribution and brightness.
type ColorProfile struct {
	DominantColors    []DominantColor `json:"dominant_colors"`
	AverageBrightness float64         `json:"average_brightness"`
	ContrastRatio     float64         `json:"contrast_ratio"`
	IsGrayscale       bool            `json:"is_grayscale"`
}

// QualityMetrics represents sharpness, blur, and lighting evaluations.
type QualityMetrics struct {
	SharpnessScore   float64 `json:"sharpness_score"`
	IsBlurry         bool    `json:"is_blurry"`
	ExposureStatus   string  `json:"exposure_status"` // "under_exposed", "well_balanced", "over_exposed"
	DynamicRangeBits int     `json:"dynamic_range_bits"`
}

// HashFingerprint holds perceptual hashes for similarity and duplicate matching.
type HashFingerprint struct {
	AHash string `json:"ahash"` // Average Hash (hex string)
	DHash string `json:"dhash"` // Difference Hash (hex string)
	PHash string `json:"phash"` // Perceptual DCT Hash (hex string)
}

// AnalysisResult compiles comprehensive analysis output.
type AnalysisResult struct {
	Metadata         ImageMetadata   `json:"metadata"`
	Color            ColorProfile    `json:"color_profile"`
	Quality          QualityMetrics  `json:"quality"`
	Fingerprint      HashFingerprint `json:"fingerprint"`
	AnalyzedAt       time.Time       `json:"analyzed_at"`
	ProcessingTimeMs int64           `json:"processing_time_ms"`
}

// DuplicateRecord represents an indexed image in the deduplication registry.
type DuplicateRecord struct {
	ID          string          `json:"id"`
	Fingerprint HashFingerprint `json:"fingerprint"`
	Metadata    ImageMetadata   `json:"metadata"`
	IndexedAt   time.Time       `json:"indexed_at"`
}

// DuplicateMatch represents a matching candidate in comparison.
type DuplicateMatch struct {
	ID              string  `json:"id"`
	SimilarityScore float64 `json:"similarity_score"` // 0.0 to 1.0 (1.0 = exact match)
	HammingDistance int     `json:"hamming_distance"`
	HashType        string  `json:"hash_type"`
}

// DuplicateCheckResult details results of searching for identical or near-identical images.
type DuplicateCheckResult struct {
	IsDuplicate    bool             `json:"is_duplicate"`
	BestMatch      *DuplicateMatch  `json:"best_match,omitempty"`
	Matches        []DuplicateMatch `json:"matches"`
	CandidateCount int              `json:"total_candidates"`
}

// TransformOptions defines parameters for resizing, cropping, and color filtering.
type TransformOptions struct {
	Width        int    `json:"width"`
	Height       int    `json:"height"`
	CropX        int    `json:"crop_x"`
	CropY        int    `json:"crop_y"`
	CropWidth    int    `json:"crop_width"`
	CropHeight   int    `json:"crop_height"`
	Grayscale    bool   `json:"grayscale"`
	InvertColors bool   `json:"invert_colors"`
	OutputFormat string `json:"output_format"` // "jpeg" or "png"
	Quality      int    `json:"quality"`       // 1-100 for JPEG (default 85)
}
