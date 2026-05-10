package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// BDEClient consumes the Banco de España statistical web service (BIEST).
//
// Real API URL is app.bde.es/bierest/resources/srdatosapp (the public REST
// endpoint), NOT bde.es/webbe/... (the web front-end). Responses are gzip
// JSON; Go's default Transport handles decompression transparently.
//
// See docs/bde-integration.md for series catalog and rationale.
type BDEClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewBDEClient() *BDEClient {
	return &BDEClient{
		baseURL: "https://app.bde.es/bierest/resources/srdatosapp",
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// bdeSeriesResponse mirrors a single element of the array returned by
// /listaSeries. Valores is RawMessage so we can tolerate null and any
// stray sentinel ("_", "null") without bailing on json.Unmarshal.
type bdeSeriesResponse struct {
	Serie       string            `json:"serie"`
	Descripcion string            `json:"descripcion"`
	ErrNum      *int              `json:"errNum,omitempty"`
	ErrMsgUsr   string            `json:"errMsgUsr,omitempty"`
	Fechas      []string          `json:"fechas"`
	Valores     []json.RawMessage `json:"valores"`
}

// FetchSeries calls /listaSeries for a single series with the given rango.
//
// `rango` is an enum (not arbitrary dates) — typical values: "30M", "60M",
// "MAX" for monthly series; "3M", "12M", "36M" for daily. Empty omits the
// param and uses the API default.
func (c *BDEClient) FetchSeries(seriesID, rango string) ([]Observation, error) {
	url := fmt.Sprintf("%s/listaSeries?idioma=es&series=%s", c.baseURL, seriesID)
	if rango != "" {
		url += "&rango=" + rango
	}

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("bde build request failed for %s: %w", seriesID, err)
	}
	// app.bde.es drops requests without a recognizable User-Agent (returns
	// EOF mid-handshake). A browser-like UA is required.
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; ticker-lab-macro-go/1.0; +https://tickerlab.dev)")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("bde request failed for %s: %w", seriesID, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bde returned %d for %s", resp.StatusCode, seriesID)
	}

	var data []bdeSeriesResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("bde decode failed for %s: %w", seriesID, err)
	}
	if len(data) == 0 {
		return nil, fmt.Errorf("bde returned empty array for %s", seriesID)
	}

	s := data[0]
	if s.ErrNum != nil {
		return nil, fmt.Errorf("bde error %d for %s: %s", *s.ErrNum, seriesID, s.ErrMsgUsr)
	}
	if len(s.Fechas) != len(s.Valores) {
		return nil, fmt.Errorf("bde fechas/valores length mismatch for %s: %d vs %d",
			seriesID, len(s.Fechas), len(s.Valores))
	}

	obs := make([]Observation, 0, len(s.Fechas))
	for i, f := range s.Fechas {
		if isBDEGap(s.Valores[i]) {
			continue
		}
		var v float64
		if err := json.Unmarshal(s.Valores[i], &v); err != nil {
			continue
		}
		date := normalizeBDEDate(f)
		if date == "" {
			continue
		}
		obs = append(obs, Observation{
			Source:   "bde",
			SeriesID: seriesID,
			Value:    v,
			Date:     date,
		})
	}
	return obs, nil
}

// isBDEGap reports whether a raw values cell represents a missing
// observation. BdE uses null most commonly; "_" / "null" string sentinels
// have been observed historically.
func isBDEGap(raw json.RawMessage) bool {
	v := strings.TrimSpace(string(raw))
	switch v {
	case "", "null", `"_"`, `"null"`, `""`:
		return true
	}
	return false
}

// normalizeBDEDate truncates ISO 8601 timestamps ("2026-04-01T08:15:00Z")
// to YYYY-MM-DD. The API embeds offsets that vary with DST; truncation
// avoids that whole class of bug.
func normalizeBDEDate(raw string) string {
	if len(raw) < 10 {
		return ""
	}
	return raw[:10]
}
