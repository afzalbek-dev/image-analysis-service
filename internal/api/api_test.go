package api

import (
	"bytes"
	"encoding/json"
	"image"
	"image/color"
	_ "image/png"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/afzalbek-dev/image-analysis-service/internal/config"
	"github.com/afzalbek-dev/image-analysis-service/internal/domain"
	"github.com/afzalbek-dev/image-analysis-service/internal/duplicates"
	"github.com/afzalbek-dev/image-analysis-service/testutil"
)

func setupTestServer() (*Server, http.Handler) {
	cfg := &config.Config{
		Port:                     "8080",
		MaxUploadSizeMB:          10,
		SimilarityThreshold:      0.85,
		HammingDistanceThreshold: 10,
		Environment:              "test",
	}
	dupSvc := duplicates.NewService()
	srv := NewServer(cfg, dupSvc)
	return srv, srv.SetupRouter()
}

func createMultipartRequest(url string, fileFieldName string, fileName string, fileBytes []byte) (*http.Request, error) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	part, err := writer.CreateFormFile(fileFieldName, fileName)
	if err != nil {
		return nil, err
	}
	if _, err := io.Copy(part, bytes.NewReader(fileBytes)); err != nil {
		return nil, err
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}

	req := httptest.NewRequest(http.MethodPost, url, &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	return req, nil
}

func TestHealthEndpoint(t *testing.T) {
	_, router := setupTestServer()

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode json: %v", err)
	}
	if resp["status"] != "ok" {
		t.Errorf("expected status ok, got %v", resp["status"])
	}
	if resp["service"] != "image-analysis-service" {
		t.Errorf("expected service image-analysis-service, got %v", resp["service"])
	}
}

func TestAnalyzeEndpoint(t *testing.T) {
	_, router := setupTestServer()

	sampleImg := testutil.CreateCheckerboardImage(120, 80, 10)
	pngBytes, err := testutil.EncodeToPNG(sampleImg)
	if err != nil {
		t.Fatalf("encode error: %v", err)
	}

	req, err := createMultipartRequest("/api/v1/analyze", "file", "test.png", pngBytes)
	if err != nil {
		t.Fatalf("request creation error: %v", err)
	}

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d, body: %s", rec.Code, rec.Body.String())
	}

	var res domain.AnalysisResult
	if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to parse analysis result: %v", err)
	}

	if res.Metadata.Width != 120 || res.Metadata.Height != 80 {
		t.Errorf("expected 120x80, got %dx%d", res.Metadata.Width, res.Metadata.Height)
	}
	if res.Fingerprint.PHash == "" {
		t.Errorf("expected non-empty pHash")
	}
	if len(res.Color.DominantColors) == 0 {
		t.Errorf("expected dominant colors")
	}
}

func TestDuplicatesFlow(t *testing.T) {
	_, router := setupTestServer()

	img := testutil.CreateCheckerboardImage(100, 100, 10)
	pngBytes, _ := testutil.EncodeToPNG(img)

	// 1. Register image
	reqReg, _ := createMultipartRequest("/api/v1/duplicates/register?id=img-orig", "file", "board.png", pngBytes)
	recReg := httptest.NewRecorder()
	router.ServeHTTP(recReg, reqReg)

	if recReg.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d: %s", recReg.Code, recReg.Body.String())
	}

	var regRecord domain.DuplicateRecord
	if err := json.Unmarshal(recReg.Body.Bytes(), &regRecord); err != nil {
		t.Fatalf("parse record error: %v", err)
	}
	if regRecord.ID != "img-orig" {
		t.Errorf("expected id img-orig, got %s", regRecord.ID)
	}

	// 2. Check identical image
	reqCheck, _ := createMultipartRequest("/api/v1/duplicates/check", "file", "grad_copy.png", pngBytes)
	recCheck := httptest.NewRecorder()
	router.ServeHTTP(recCheck, reqCheck)

	if recCheck.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", recCheck.Code)
	}

	var checkResult domain.DuplicateCheckResult
	if err := json.Unmarshal(recCheck.Body.Bytes(), &checkResult); err != nil {
		t.Fatalf("parse check result error: %v", err)
	}
	if !checkResult.IsDuplicate {
		t.Errorf("expected duplicate match for identical image")
	}
	if checkResult.BestMatch == nil || checkResult.BestMatch.ID != "img-orig" {
		t.Errorf("expected match with img-orig")
	}

	// 3. Check dissimilar image
	diffImg := testutil.CreateSolidImage(100, 100, color.RGBA{R: 255, G: 255, B: 255, A: 255})
	diffBytes, _ := testutil.EncodeToPNG(diffImg)
	reqDiff, _ := createMultipartRequest("/api/v1/duplicates/check", "file", "white.png", diffBytes)
	recDiff := httptest.NewRecorder()
	router.ServeHTTP(recDiff, reqDiff)

	var diffResult domain.DuplicateCheckResult
	_ = json.Unmarshal(recDiff.Body.Bytes(), &diffResult)
	if diffResult.IsDuplicate {
		t.Errorf("white image should not duplicate gradient image")
	}
}

func TestTransformEndpoint(t *testing.T) {
	_, router := setupTestServer()

	img := testutil.CreateCheckerboardImage(200, 200, 20)
	pngBytes, _ := testutil.EncodeToPNG(img)

	req, _ := createMultipartRequest("/api/v1/process/transform?width=100&height=100&grayscale=true&format=png", "file", "board.png", pngBytes)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d, body: %s", rec.Code, rec.Body.String())
	}
	if rec.Header().Get("Content-Type") != "image/png" {
		t.Errorf("expected image/png content type, got %s", rec.Header().Get("Content-Type"))
	}

	decoded, format, err := image.Decode(rec.Body)
	if err != nil {
		t.Fatalf("failed to decode response image: %v", err)
	}
	if format != "png" {
		t.Errorf("expected format png, got %s", format)
	}
	if decoded.Bounds().Dx() != 100 || decoded.Bounds().Dy() != 100 {
		t.Errorf("expected 100x100 bounds, got %dx%d", decoded.Bounds().Dx(), decoded.Bounds().Dy())
	}
}
