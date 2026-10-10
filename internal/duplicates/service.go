package duplicates

import (
	"sync"
	"time"

	"github.com/afzalbek-dev/image-analysis-service/internal/domain"
	"github.com/afzalbek-dev/image-analysis-service/internal/hasher"
)

// Service provides thread-safe indexing and lookup of images for near-duplicate identification.
type Service struct {
	mu      sync.RWMutex
	records map[string]domain.DuplicateRecord
}

// NewService creates a new in-memory duplicate detection store.
func NewService() *Service {
	return &Service{
		records: make(map[string]domain.DuplicateRecord),
	}
}

// Register stores or updates an image fingerprint record in the registry.
func (s *Service) Register(id string, fp domain.HashFingerprint, meta domain.ImageMetadata) domain.DuplicateRecord {
	s.mu.Lock()
	defer s.mu.Unlock()

	rec := domain.DuplicateRecord{
		ID:          id,
		Fingerprint: fp,
		Metadata:    meta,
		IndexedAt:   time.Now().UTC(),
	}
	s.records[id] = rec
	return rec
}

// Check compares a query fingerprint against all indexed items, returning matches within threshold.
func (s *Service) Check(query domain.HashFingerprint, maxHammingDist int, minSimilarity float64) domain.DuplicateCheckResult {
	s.mu.RLock()
	defer s.mu.RUnlock()

	queryPHash, err := hasher.ParseHex(query.PHash)
	if err != nil {
		return domain.DuplicateCheckResult{IsDuplicate: false}
	}
	queryDHash, _ := hasher.ParseHex(query.DHash)

	var matches []domain.DuplicateMatch
	var bestMatch *domain.DuplicateMatch
	highestSim := -1.0

	for id, rec := range s.records {
		recPHash, err := hasher.ParseHex(rec.Fingerprint.PHash)
		if err != nil {
			continue
		}
		pDist := hasher.HammingDistance(queryPHash, recPHash)
		pSim := hasher.Similarity(queryPHash, recPHash)

		// Secondary check with dHash
		recDHash, _ := hasher.ParseHex(rec.Fingerprint.DHash)
		dDist := hasher.HammingDistance(queryDHash, recDHash)
		dSim := hasher.Similarity(queryDHash, recDHash)

		combinedSim := (pSim * 0.7) + (dSim * 0.3)
		effectiveDist := pDist
		if dDist < pDist {
			effectiveDist = (pDist + dDist) / 2
		}

		if effectiveDist <= maxHammingDist && combinedSim >= minSimilarity {
			match := domain.DuplicateMatch{
				ID:              id,
				SimilarityScore: combinedSim,
				HammingDistance: effectiveDist,
				HashType:        "perceptual_dct",
			}
			matches = append(matches, match)

			if combinedSim > highestSim {
				highestSim = combinedSim
				bm := match
				bestMatch = &bm
			}
		}
	}

	return domain.DuplicateCheckResult{
		IsDuplicate:    bestMatch != nil,
		BestMatch:      bestMatch,
		Matches:        matches,
		CandidateCount: len(s.records),
	}
}

// Get retrieves a record by identifier.
func (s *Service) Get(id string) (*domain.DuplicateRecord, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rec, found := s.records[id]
	if !found {
		return nil, false
	}
	return &rec, true
}

// Delete removes a record by ID.
func (s *Service) Delete(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, found := s.records[id]; !found {
		return false
	}
	delete(s.records, id)
	return true
}

// Count returns the number of indexed records.
func (s *Service) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.records)
}

// Clear removes all records from the registry.
func (s *Service) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.records = make(map[string]domain.DuplicateRecord)
}
