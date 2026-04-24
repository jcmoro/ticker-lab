package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

type FREDClient struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

func NewFREDClient(apiKey string) *FREDClient {
	return &FREDClient{
		baseURL: "https://api.stlouisfed.org/fred",
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

type fredObservationsResponse struct {
	Observations []struct {
		Date  string `json:"date"`
		Value string `json:"value"`
	} `json:"observations"`
}

func (c *FREDClient) FetchObservations(seriesID, startDate string) ([]Observation, error) {
	url := fmt.Sprintf("%s/series/observations?series_id=%s&api_key=%s&file_type=json",
		c.baseURL, seriesID, c.apiKey)

	if startDate != "" {
		url += "&observation_start=" + startDate
	}

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("fred request failed for %s: %w", seriesID, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fred returned %d for %s", resp.StatusCode, seriesID)
	}

	var data fredObservationsResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("fred decode failed for %s: %w", seriesID, err)
	}

	var obs []Observation
	for _, o := range data.Observations {
		if o.Value == "." {
			continue
		}
		val, err := strconv.ParseFloat(o.Value, 64)
		if err != nil {
			continue
		}
		obs = append(obs, Observation{
			Source:   "fred",
			SeriesID: seriesID,
			Value:    val,
			Date:     o.Date,
		})
	}

	return obs, nil
}
