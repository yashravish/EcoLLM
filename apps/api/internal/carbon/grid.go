package carbon

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Regional carbon intensity reference values (gCO2/kWh).
// Source: EPA eGRID + Electricity Maps averages (see AGENT_KNOWLEDGE_BASE.md Layer 6).
var regionalIntensity = map[string]float64{
	"US-EAST":    450, // EPA eGRID average
	"US-WEST":    240, // High renewables (CA/PNW mix)
	"US-CENTRAL": 520,
	"EU-WEST":    300, // European average
	"EU-DE":      370, // Germany (coal + gas mix)
	"EU-NO":      40,  // Norway (hydro-dominant)
	"COAL":       850, // Worst-case coal region
}

const (
	defaultIntensity   = 450 // US average fallback per architecture spec
	gridCacheTTL       = time.Hour
	gridCachePrefix    = "grid:"
	electricityMapsURL = "https://api.electricitymap.org/v3/carbon-intensity/latest"
)

// GridIntensityForRegion returns the static carbon intensity for a given region code.
// Falls back to US average if the region is unrecognized.
func GridIntensityForRegion(region string) float64 {
	if intensity, ok := regionalIntensity[region]; ok {
		return intensity
	}
	return defaultIntensity
}

// GridCache is a minimal key-value cache interface used by GridService.
// Implement it with a Redis adapter (see database.NewGridCacheAdapter) or
// a no-op for testing.
type GridCache interface {
	Get(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
}

// GridService fetches grid carbon intensity with a three-tier priority:
//  1. GridCache (Redis via adapter, key: "grid:{region}", TTL: 1h)
//  2. Electricity Maps API v3
//  3. Static regional fallback map
//
// All paths return a valid GridData — GetIntensity never errors.
type GridService struct {
	cache   GridCache  // nil disables caching
	apiKey  string     // empty disables Electricity Maps API calls
	httpCli *http.Client
}

func NewGridService(cache GridCache, apiKey string) *GridService {
	return &GridService{
		cache:  cache,
		apiKey: apiKey,
		httpCli: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

// GetIntensity returns grid carbon intensity for the given region.
// It tries cache → Electricity Maps API → static fallback in order.
func (g *GridService) GetIntensity(ctx context.Context, region string) GridData {
	now := time.Now()

	if g.cache != nil {
		if cached, ok := g.fromCache(ctx, region); ok {
			return cached
		}
	}

	if g.apiKey != "" {
		if live, ok := g.fromAPI(ctx, region); ok {
			if g.cache != nil {
				g.writeCache(ctx, region, live)
			}
			return live
		}
	}

	return GridData{
		Region:        region,
		IntensityGCO2: GridIntensityForRegion(region),
		DataSource:    "static",
		UpdatedAt:     now,
		ExpiresAt:     now.Add(gridCacheTTL),
	}
}

func (g *GridService) fromCache(ctx context.Context, region string) (GridData, bool) {
	raw, err := g.cache.Get(ctx, gridCachePrefix+region)
	if err != nil {
		return GridData{}, false
	}
	var d GridData
	if err := json.Unmarshal(raw, &d); err != nil {
		return GridData{}, false
	}
	if time.Now().After(d.ExpiresAt) {
		return GridData{}, false
	}
	return d, true
}

func (g *GridService) fromAPI(ctx context.Context, region string) (GridData, bool) {
	url := fmt.Sprintf("%s?zone=%s", electricityMapsURL, region)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return GridData{}, false
	}
	req.Header.Set("auth-token", g.apiKey)

	resp, err := g.httpCli.Do(req)
	if err != nil {
		return GridData{}, false
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return GridData{}, false
	}

	var payload struct {
		Zone            string  `json:"zone"`
		CarbonIntensity float64 `json:"carbonIntensity"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return GridData{}, false
	}
	if payload.CarbonIntensity <= 0 {
		return GridData{}, false
	}

	now := time.Now()
	return GridData{
		Region:        region,
		IntensityGCO2: payload.CarbonIntensity,
		DataSource:    "electricity_maps",
		UpdatedAt:     now,
		ExpiresAt:     now.Add(gridCacheTTL),
	}, true
}

func (g *GridService) writeCache(ctx context.Context, region string, d GridData) {
	raw, err := json.Marshal(d)
	if err != nil {
		return
	}
	_ = g.cache.Set(ctx, gridCachePrefix+region, raw, gridCacheTTL)
}
