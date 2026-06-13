package labelparser

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"log"
	"sync"
	"time"
)

type ScanCache interface {
	Get(imageHash string) (ScanLabelResult, bool)
	Set(imageHash string, result ScanLabelResult)
}

type ScanPipeline struct {
	parser   Parser
	detector SymbolDetector
	cache    ScanCache
}

func NewScanPipeline(parser Parser, detector SymbolDetector, cache ScanCache) ScanPipeline {
	return ScanPipeline{
		parser:   parser,
		detector: detector,
		cache:    cache,
	}
}

func (p ScanPipeline) ParseLabel(ctx context.Context, input ParseLabelInput) (ParseLabelResult, error) {
	if p.parser == nil {
		return ParseLabelResult{}, ErrProviderUnavailable
	}
	return p.parser.ParseLabel(ctx, input)
}

func (p ScanPipeline) ScanLabel(ctx context.Context, input ScanLabelInput) (ScanLabelResult, error) {
	if p.parser == nil {
		return ScanLabelResult{}, ErrProviderUnavailable
	}

	imageHash := input.ImageHash
	if imageHash == "" {
		imageHash = hashScanImage(input.ParseLabelInput)
		input.ImageHash = imageHash
	}

	if p.cache != nil && imageHash != "" {
		if result, ok := p.cache.Get(imageHash); ok {
			result.CacheHit = true
			result.ImageHash = imageHash
			result.Route = "cache"
			if result.Provider == "" {
				result.Provider = "cache"
			}
			if result.PaidFallbackUsed {
				result.FallbackCallsAvoided = 1
			}
			result.RoutingReasons = appendUniqueString(result.RoutingReasons, "image_hash_cache_hit")
			return normalizeScanLabelResult(result), nil
		}
	}

	if p.detector != nil {
		detections, err := p.detector.DetectSymbols(ctx, input)
		if err != nil {
			log.Printf("scan-label symbol detector failed: %v", err)
			input.DetectedSymbols = normalizeSymbolDetections(input.DetectedSymbols)
		} else {
			input.DetectedSymbols = mergeSymbolDetections(input.DetectedSymbols, detections)
		}
	}

	result, err := scanWithParser(ctx, p.parser, input)
	if err != nil {
		return ScanLabelResult{}, err
	}

	result.ImageHash = imageHash
	result.CacheHit = false
	result.SymbolDetections = mergeSymbolDetections(input.DetectedSymbols, result.SymbolDetections)
	result.RoutingReasons = appendUniqueString(result.RoutingReasons, "image_hash_cache_miss")
	result = normalizeScanLabelResult(result)

	if p.cache != nil && imageHash != "" {
		p.cache.Set(imageHash, result)
	}

	return result, nil
}

func scanWithParser(ctx context.Context, parser Parser, input ScanLabelInput) (ScanLabelResult, error) {
	if scanner, ok := parser.(Scanner); ok {
		result, err := scanner.ScanLabel(ctx, input)
		if err != nil {
			return ScanLabelResult{}, err
		}
		return normalizeScanLabelResult(result), nil
	}

	result, err := parser.ParseLabel(ctx, input.ParseLabelInput)
	if err != nil {
		return ScanLabelResult{}, err
	}
	return scanFromParseLabelResult(result, careRuleEvidence{}, "legacy parser", 0), nil
}

func hashScanImage(input ParseLabelInput) string {
	if len(input.Content) == 0 {
		return ""
	}

	hash := sha256.New()
	_, _ = hash.Write([]byte(input.MIMEType))
	_, _ = hash.Write([]byte{0})
	_, _ = hash.Write(input.Content)
	return hex.EncodeToString(hash.Sum(nil))
}

func mergeSymbolDetections(first []SymbolDetection, second []SymbolDetection) []SymbolDetection {
	merged := normalizeSymbolDetections(first)
	for _, detection := range normalizeSymbolDetections(second) {
		merged = appendUniqueSymbolDetection(merged, detection)
	}
	if merged == nil {
		return []SymbolDetection{}
	}
	return merged
}

type MemoryScanCache struct {
	mu         sync.Mutex
	ttl        time.Duration
	maxEntries int
	entries    map[string]memoryScanCacheEntry
	hits       int
	misses     int
	avoided    int
}

type memoryScanCacheEntry struct {
	result   ScanLabelResult
	storedAt time.Time
}

func NewMemoryScanCache(ttl time.Duration, maxEntries int) *MemoryScanCache {
	if ttl <= 0 {
		ttl = 24 * time.Hour
	}
	if maxEntries <= 0 {
		maxEntries = 512
	}

	return &MemoryScanCache{
		ttl:        ttl,
		maxEntries: maxEntries,
		entries:    map[string]memoryScanCacheEntry{},
	}
}

func (c *MemoryScanCache) Get(imageHash string) (ScanLabelResult, bool) {
	if c == nil || imageHash == "" {
		return ScanLabelResult{}, false
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	entry, ok := c.entries[imageHash]
	if !ok {
		c.misses++
		return ScanLabelResult{}, false
	}
	if time.Since(entry.storedAt) > c.ttl {
		delete(c.entries, imageHash)
		c.misses++
		return ScanLabelResult{}, false
	}

	c.hits++
	if entry.result.PaidFallbackUsed {
		c.avoided++
	}
	return cloneScanResult(entry.result), true
}

func (c *MemoryScanCache) Set(imageHash string, result ScanLabelResult) {
	if c == nil || imageHash == "" {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if len(c.entries) >= c.maxEntries {
		c.evictOldest()
	}

	result.CacheHit = false
	result.FallbackCallsAvoided = 0
	c.entries[imageHash] = memoryScanCacheEntry{
		result:   cloneScanResult(result),
		storedAt: time.Now(),
	}
}

func (c *MemoryScanCache) evictOldest() {
	var oldestKey string
	var oldestTime time.Time
	for key, entry := range c.entries {
		if oldestKey == "" || entry.storedAt.Before(oldestTime) {
			oldestKey = key
			oldestTime = entry.storedAt
		}
	}
	if oldestKey != "" {
		delete(c.entries, oldestKey)
	}
}

type ScanCacheStats struct {
	Hits                 int `json:"hits"`
	Misses               int `json:"misses"`
	FallbackCallsAvoided int `json:"fallback_calls_avoided"`
	Entries              int `json:"entries"`
}

func (c *MemoryScanCache) Stats() ScanCacheStats {
	if c == nil {
		return ScanCacheStats{}
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	return ScanCacheStats{
		Hits:                 c.hits,
		Misses:               c.misses,
		FallbackCallsAvoided: c.avoided,
		Entries:              len(c.entries),
	}
}

func cloneScanResult(result ScanLabelResult) ScanLabelResult {
	result.UncertainFields = append([]string(nil), result.UncertainFields...)
	result.SymbolDetections = append([]SymbolDetection(nil), result.SymbolDetections...)
	result.RoutingReasons = append([]string(nil), result.RoutingReasons...)
	return result
}
