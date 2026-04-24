package main

import (
	"encoding/csv"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

type ECBClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewECBClient() *ECBClient {
	return &ECBClient{
		baseURL: "https://data-api.ecb.europa.eu/service",
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *ECBClient) FetchDataflow(seriesID, dataflow, key, startPeriod string) ([]Observation, error) {
	url := fmt.Sprintf("%s/data/%s/%s?format=csvdata", c.baseURL, dataflow, key)
	if startPeriod != "" {
		url += "&startPeriod=" + startPeriod
	}

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("ecb request failed for %s: %w", seriesID, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ecb returned %d for %s", resp.StatusCode, seriesID)
	}

	reader := csv.NewReader(resp.Body)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("ecb csv parse failed for %s: %w", seriesID, err)
	}

	if len(records) < 2 {
		return nil, nil
	}

	// Find column indices for TIME_PERIOD and OBS_VALUE
	header := records[0]
	timeIdx, valueIdx := -1, -1
	for i, col := range header {
		switch col {
		case "TIME_PERIOD":
			timeIdx = i
		case "OBS_VALUE":
			valueIdx = i
		}
	}
	if timeIdx == -1 || valueIdx == -1 {
		return nil, fmt.Errorf("ecb csv missing TIME_PERIOD or OBS_VALUE columns for %s", seriesID)
	}

	var obs []Observation
	for _, row := range records[1:] {
		if len(row) <= timeIdx || len(row) <= valueIdx {
			continue
		}
		val, err := strconv.ParseFloat(row[valueIdx], 64)
		if err != nil {
			continue
		}
		date := normalizeECBDate(row[timeIdx])
		obs = append(obs, Observation{
			Source:   "ecb",
			SeriesID: seriesID,
			Value:    val,
			Date:     date,
		})
	}

	return obs, nil
}

// normalizeECBDate converts ECB date formats (YYYY-MM, YYYY-Q1, YYYY) to YYYY-MM-DD.
func normalizeECBDate(raw string) string {
	switch len(raw) {
	case 10: // YYYY-MM-DD
		return raw
	case 7: // YYYY-MM
		return raw + "-01"
	case 4: // YYYY
		return raw + "-01-01"
	default:
		return raw
	}
}
