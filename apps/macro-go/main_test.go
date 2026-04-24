package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

func getTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("DATABASE_URL not set, skipping integration test")
	}
	pool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		t.Fatalf("Unable to connect to database: %v", err)
	}
	repo := NewRepository(pool)
	if err := repo.Migrate(context.Background()); err != nil {
		t.Fatalf("migrate failed: %v", err)
	}
	t.Cleanup(func() { pool.Close() })
	return pool
}

func TestHealthEndpoint(t *testing.T) {
	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	handleHealth(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var resp HealthResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode failed: %v", err)
	}

	if resp.Status != "ok" {
		t.Errorf("expected 'ok', got '%s'", resp.Status)
	}
	if resp.Engine != "go-macro" {
		t.Errorf("expected 'go-macro', got '%s'", resp.Engine)
	}
}

func TestCorsMiddleware(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /test", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := corsMiddleware(mux)

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Errorf("expected CORS '*', got '%s'", got)
	}
}

func TestIndicatorsEndpoint(t *testing.T) {
	pool := getTestPool(t)
	repo := NewRepository(pool)

	req := httptest.NewRequest("GET", "/api/v1/macro/indicators", nil)
	w := httptest.NewRecorder()

	handleIndicators(repo)(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Count      int         `json:"count"`
		Indicators []Indicator `json:"indicators"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode failed: %v", err)
	}
}

func TestHistoryEndpoint(t *testing.T) {
	pool := getTestPool(t)
	repo := NewRepository(pool)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/macro/{source}/{id}/history", handleHistory(repo))

	req := httptest.NewRequest("GET", "/api/v1/macro/fred/CPIAUCSL/history?days=365", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Source   string         `json:"source"`
		SeriesID string        `json:"series_id"`
		Days     int            `json:"days"`
		Count    int            `json:"count"`
		Points   []HistoryPoint `json:"points"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode failed: %v", err)
	}

	if resp.Source != "fred" {
		t.Errorf("expected 'fred', got '%s'", resp.Source)
	}
	if resp.SeriesID != "CPIAUCSL" {
		t.Errorf("expected 'CPIAUCSL', got '%s'", resp.SeriesID)
	}
	if resp.Days != 365 {
		t.Errorf("expected 365 days, got %d", resp.Days)
	}
}

func TestHistoryEndpoint_DefaultDays(t *testing.T) {
	pool := getTestPool(t)
	repo := NewRepository(pool)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/macro/{source}/{id}/history", handleHistory(repo))

	req := httptest.NewRequest("GET", "/api/v1/macro/ecb/ICP/history", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp struct {
		Days int `json:"days"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode failed: %v", err)
	}

	if resp.Days != 365 {
		t.Errorf("expected default 365 days, got %d", resp.Days)
	}
}

func TestSaveAndFindIndicators(t *testing.T) {
	pool := getTestPool(t)
	repo := NewRepository(pool)

	// Seed test series
	testSeries := []SeriesMeta{
		{Source: "test", SeriesID: "TEST_CPI", Name: "Test CPI", Freq: "monthly", Unit: "index", Category: "inflation"},
	}
	if err := repo.SeedSeries(context.Background(), testSeries); err != nil {
		t.Fatalf("seed failed: %v", err)
	}

	// Save observations
	obs := []Observation{
		{Source: "test", SeriesID: "TEST_CPI", Value: 100.0, Date: "2026-01-01"},
		{Source: "test", SeriesID: "TEST_CPI", Value: 101.5, Date: "2026-02-01"},
	}
	if err := repo.SaveObservations(context.Background(), obs); err != nil {
		t.Fatalf("save failed: %v", err)
	}

	// Find indicators
	indicators, err := repo.FindIndicators(context.Background(), "")
	if err != nil {
		t.Fatalf("find indicators failed: %v", err)
	}

	found := false
	for _, ind := range indicators {
		if ind.Source == "test" && ind.SeriesID == "TEST_CPI" {
			found = true
			if ind.LatestValue != 101.5 {
				t.Errorf("expected latest 101.5, got %f", ind.LatestValue)
			}
			if ind.PrevValue != 100.0 {
				t.Errorf("expected prev 100.0, got %f", ind.PrevValue)
			}
			if ind.Change != 1.5 {
				t.Errorf("expected change 1.5, got %f", ind.Change)
			}
		}
	}
	if !found {
		t.Error("TEST_CPI not found in indicators")
	}

	// Find history
	points, err := repo.FindHistory(context.Background(), "test", "TEST_CPI", 365)
	if err != nil {
		t.Fatalf("find history failed: %v", err)
	}
	if len(points) != 2 {
		t.Errorf("expected 2 history points, got %d", len(points))
	}

	// Cleanup
	_, _ = pool.Exec(context.Background(), "DELETE FROM macro_observations WHERE source = 'test'")
	_, _ = pool.Exec(context.Background(), "DELETE FROM macro_series WHERE source = 'test'")
}
