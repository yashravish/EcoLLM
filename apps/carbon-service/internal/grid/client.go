package grid

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
)

// staticIntensity is the fallback map (gCO2/kWh) from AGENT_KNOWLEDGE_BASE Layer 6.
var staticIntensity = map[string]float64{
	"US-EAST": 450,
	"US-WEST": 240,
	"EU-WEST": 300,
	"EU-NO":   40,
	"EU-DE":   400,
	"APAC-SG": 430,
	"COAL":    850,
	"HYDRO":   20,
}

const (
	cacheTTL      = time.Hour
	cacheKeyFmt   = "grid:%s"
	electricityMapsBaseURL = "https://api.electricitymap.org/v3/carbon-intensity/latest"
)

// Client fetches and caches grid carbon intensity.
type Client struct {
	rdb           *redis.Client
	apiKey        string
	defaultRegion string
	http          *http.Client
}

func New(rdb *redis.Client, apiKey, defaultRegion string) *Client {
	return &Client{
		rdb:           rdb,
		apiKey:        apiKey,
		defaultRegion: defaultRegion,
		http:          &http.Client{Timeout: 5 * time.Second},
	}
}

// Intensity returns (gCO2/kWh, source, error) for the given region.
// Source is "electricity_maps", "redis_cache", or "static_fallback".
func (c *Client) Intensity(ctx context.Context, region string) (float64, string, error) {
	key := fmt.Sprintf(cacheKeyFmt, region)

	// Try cache first
	val, err := c.rdb.Get(ctx, key).Float64()
	if err == nil {
		return val, "redis_cache", nil
	}

	// Try Electricity Maps API if key configured
	if c.apiKey != "" {
		intensity, err := c.fetchFromAPI(ctx, region)
		if err == nil {
			_ = c.rdb.Set(ctx, key, intensity, cacheTTL).Err()
			return intensity, "electricity_maps", nil
		}
		log.Warn().Err(err).Str("region", region).Msg("electricity maps fetch failed, falling back to static")
	}

	// Static fallback
	if v, ok := staticIntensity[region]; ok {
		_ = c.rdb.Set(ctx, key, v, cacheTTL).Err()
		return v, "static_fallback", nil
	}

	// Ultimate fallback: US average
	_ = c.rdb.Set(ctx, key, staticIntensity["US-EAST"], cacheTTL).Err()
	return staticIntensity["US-EAST"], "static_fallback", nil
}

type electricityMapsResponse struct {
	CarbonIntensity float64 `json:"carbonIntensity"`
}

func (c *Client) fetchFromAPI(ctx context.Context, region string) (float64, error) {
	url := fmt.Sprintf("%s?zone=%s", electricityMapsBaseURL, region)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("auth-token", c.apiKey)

	resp, err := c.http.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("electricity maps returned %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}

	var parsed electricityMapsResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return 0, err
	}
	return parsed.CarbonIntensity, nil
}
