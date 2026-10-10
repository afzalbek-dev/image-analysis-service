package api

import (
	"encoding/json"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/afzalbek-dev/image-analysis-service/internal/analyzer"
	"github.com/afzalbek-dev/image-analysis-service/internal/config"
	"github.com/afzalbek-dev/image-analysis-service/internal/domain"
	"github.com/afzalbek-dev/image-analysis-service/internal/duplicates"
	"github.com/afzalbek-dev/image-analysis-service/internal/processor"
)

// Server encapsulates HTTP handler state and dependencies.
type Server struct {
	cfg        *config.Config
	dupService *duplicates.Service
	startTime  time.Time
}

// NewServer initializes a new Server.
func NewServer(cfg *config.Config, dupService *duplicates.Service) *Server {
	return &Server{
		cfg:        cfg,
		dupService: dupService,
		startTime:  time.Now(),
	}
}

func (s *Server) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func (s *Server) writeError(w http.ResponseWriter, status int, message string) {
	s.writeJSON(w, status, map[string]string{
		"error": message,
	})
}

// HandleHealth returns service health and uptime status.
func (s *Server) HandleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	uptime := int64(time.Since(s.startTime).Seconds())
	s.writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":         "ok",
		"service":        "image-analysis-service",
		"version":        "1.0.0",
		"environment":    s.cfg.Environment,
		"uptime_seconds": uptime,
		"indexed_images": s.dupService.Count(),
	})
}

// readUploadedImage extracts image from multipart form or request body.
func (s *Server) readUploadedImage(r *http.Request) (image.Image, string, int64, error) {
	maxBytes := s.cfg.MaxUploadSizeMB * 1024 * 1024
	r.Body = http.MaxBytesReader(nil, r.Body, maxBytes)

	var reader io.Reader
	var formatHint string

	if strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data") {
		if err := r.ParseMultipartForm(maxBytes); err != nil {
			return nil, "", 0, fmt.Errorf("failed to parse multipart form: %w", err)
		}
		file, header, err := r.FormFile("file")
		if err != nil {
			return nil, "", 0, fmt.Errorf("form field 'file' is required")
		}
		defer file.Close()
		reader = file
		formatHint = header.Filename
	} else {
		reader = r.Body
	}

	// Buffer bytes to allow format identification and size calculation
	bytesData, err := io.ReadAll(reader)
	if err != nil {
		return nil, "", 0, fmt.Errorf("failed to read image data: %w", err)
	}
	if len(bytesData) == 0 {
		return nil, "", 0, fmt.Errorf("empty image payload received")
	}

	img, format, err := image.Decode(strings.NewReader(string(bytesData)))
	if err != nil {
		// Fallback check extension
		if strings.HasSuffix(strings.ToLower(formatHint), ".png") {
			format = "png"
		} else if strings.HasSuffix(strings.ToLower(formatHint), ".jpg") || strings.HasSuffix(strings.ToLower(formatHint), ".jpeg") {
			format = "jpeg"
		} else {
			return nil, "", 0, fmt.Errorf("unsupported or corrupt image format: %w", err)
		}
	}

	return img, format, int64(len(bytesData)), nil
}

// HandleAnalyze inspects an uploaded image and computes metadata, color palette, quality, and hashes.
func (s *Server) HandleAnalyze(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	img, format, bytesLen, err := s.readUploadedImage(r)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	result := analyzer.Analyze(img, format, bytesLen)
	s.writeJSON(w, http.StatusOK, result)
}

// HandleDuplicateRegister registers an image fingerprint into the deduplication catalog.
func (s *Server) HandleDuplicateRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	id := r.URL.Query().Get("id")
	if id == "" {
		id = fmt.Sprintf("img-%d", time.Now().UnixNano())
	}

	img, format, bytesLen, err := s.readUploadedImage(r)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	meta := analyzer.ExtractMetadata(img, format, bytesLen)
	analysis := analyzer.Analyze(img, format, bytesLen)

	record := s.dupService.Register(id, analysis.Fingerprint, meta)
	s.writeJSON(w, http.StatusCreated, record)
}

// HandleDuplicateCheck compares an uploaded image against registered images.
func (s *Server) HandleDuplicateCheck(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	maxDist := s.cfg.HammingDistanceThreshold
	if distStr := r.URL.Query().Get("max_distance"); distStr != "" {
		if parsed, err := strconv.Atoi(distStr); err == nil {
			maxDist = parsed
		}
	}

	minSim := s.cfg.SimilarityThreshold
	if simStr := r.URL.Query().Get("min_similarity"); simStr != "" {
		if parsed, err := strconv.ParseFloat(simStr, 64); err == nil {
			minSim = parsed
		}
	}

	img, format, bytesLen, err := s.readUploadedImage(r)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	analysis := analyzer.Analyze(img, format, bytesLen)
	checkResult := s.dupService.Check(analysis.Fingerprint, maxDist, minSim)

	s.writeJSON(w, http.StatusOK, checkResult)
}

// HandleTransform applies crop, resize, grayscale, or invert transformations and encodes output.
func (s *Server) HandleTransform(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	img, _, _, err := s.readUploadedImage(r)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	q := r.URL.Query()
	opts := domain.TransformOptions{
		Width:        getIntParam(q.Get("width"), 0),
		Height:       getIntParam(q.Get("height"), 0),
		CropX:        getIntParam(q.Get("crop_x"), 0),
		CropY:        getIntParam(q.Get("crop_y"), 0),
		CropWidth:    getIntParam(q.Get("crop_width"), 0),
		CropHeight:   getIntParam(q.Get("crop_height"), 0),
		Grayscale:    q.Get("grayscale") == "true" || q.Get("grayscale") == "1",
		InvertColors: q.Get("invert") == "true" || q.Get("invert") == "1",
		OutputFormat: q.Get("format"),
		Quality:      getIntParam(q.Get("quality"), 85),
	}

	if opts.OutputFormat == "" {
		opts.OutputFormat = "png"
	}

	transformed, err := processor.ApplyTransform(img, opts)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	encoded, err := processor.Encode(transformed, opts.OutputFormat, opts.Quality)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	contentType := "image/png"
	if opts.OutputFormat == "jpeg" || opts.OutputFormat == "jpg" {
		contentType = "image/jpeg"
	}

	w.Header().Set("Content-Type", contentType)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(encoded)
}

func getIntParam(val string, defaultVal int) int {
	if val == "" {
		return defaultVal
	}
	if parsed, err := strconv.Atoi(val); err == nil {
		return parsed
	}
	return defaultVal
}
