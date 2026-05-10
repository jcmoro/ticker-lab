package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// ESIOSClient consumes the REE e·sios API.
//
// Auth uses the modern x-api-key header. Tokens are personal and requested
// via email to consultasios@ree.es. See docs/esios-integration.md.
type ESIOSClient struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

func NewESIOSClient(apiKey string) *ESIOSClient {
	return &ESIOSClient{
		baseURL:    "https://api.esios.ree.es",
		apiKey:     apiKey,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// indicatorResponse mirrors the slice of /indicators/{id} we actually use.
// ESIOS returns more metadata; we only decode what we persist.
type indicatorResponse struct {
	Indicator struct {
		ID     int    `json:"id"`
		Name   string `json:"name"`
		Values []struct {
			Value       float64   `json:"value"`
			DatetimeUTC time.Time `json:"datetime_utc"`
			GeoID       int       `json:"geo_id"`
		} `json:"values"`
	} `json:"indicator"`
}

// FetchIndicator pulls observations for a single indicator + geo within a
// time range. timeTrunc is the ESIOS aggregation level ("hour", "day", ...).
//
// REE occasionally ignores geo_ids[] and returns all geos; we re-filter
// client-side as a defense in depth.
func (c *ESIOSClient) FetchIndicator(
	ctx context.Context,
	indicatorID, geoID int,
	startDate, endDate time.Time,
	timeTrunc string,
) ([]Observation, error) {
	url := fmt.Sprintf(
		"%s/indicators/%d?start_date=%s&end_date=%s&time_trunc=%s&geo_ids[]=%d",
		c.baseURL,
		indicatorID,
		startDate.UTC().Format("2006-01-02T15:04:05"),
		endDate.UTC().Format("2006-01-02T15:04:05"),
		timeTrunc,
		geoID,
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("esios build request for %d: %w", indicatorID, err)
	}
	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("Accept", "application/json; application/vnd.esios-api-v1+json")
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("esios request failed for %d: %w", indicatorID, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("esios returned %d for indicator %d", resp.StatusCode, indicatorID)
	}

	var data indicatorResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("esios decode failed for %d: %w", indicatorID, err)
	}

	obs := make([]Observation, 0, len(data.Indicator.Values))
	for _, v := range data.Indicator.Values {
		// Defensive: REE sometimes returns all geos despite the filter.
		if v.GeoID != geoID {
			continue
		}
		obs = append(obs, Observation{
			IndicatorID: indicatorID,
			GeoID:       geoID,
			Value:       v.Value,
			DatetimeUTC: v.DatetimeUTC.UTC(),
		})
	}
	return obs, nil
}
