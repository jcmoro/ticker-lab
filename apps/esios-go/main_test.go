package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ticker-lab/httpx"
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
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp HealthResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Status != "ok" || resp.Engine != "go-esios" {
		t.Errorf("unexpected health: %+v", resp)
	}
}

func TestCorsMiddleware(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /test", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := httpx.CORSMiddleware(mux)

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Errorf("CORS origin = %q, want *", got)
	}
}

// TestFetchIndicator_FixtureParsing exercises esios.go against a recorded
// response payload. Verifies URL/header construction, decoding, and the
// defensive geo_id filter (REE sometimes returns all geos).
func TestFetchIndicator_FixtureParsing(t *testing.T) {
	fixture, err := os.ReadFile("testdata/indicator_1001.json")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	var seenAuth, seenAccept string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenAuth = r.Header.Get("x-api-key")
		seenAccept = r.Header.Get("Accept")
		if !strings.Contains(r.URL.RawQuery, "geo_ids[]=8741") {
			t.Errorf("missing geo_ids[]=8741 in query: %s", r.URL.RawQuery)
		}
		if !strings.Contains(r.URL.RawQuery, "time_trunc=hour") {
			t.Errorf("missing time_trunc=hour in query: %s", r.URL.RawQuery)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(fixture)
	}))
	t.Cleanup(server.Close)

	client := NewESIOSClient("test-key")
	client.baseURL = server.URL

	start := time.Date(2026, 5, 8, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 5, 9, 0, 0, 0, 0, time.UTC)
	obs, err := client.FetchIndicator(context.Background(), 1001, 8741, start, end, "hour")
	if err != nil {
		t.Fatalf("FetchIndicator: %v", err)
	}

	// Fixture has 3 entries for geo 8741 + 1 for geo 8742; client must filter.
	if len(obs) != 3 {
		t.Fatalf("len(obs) = %d, want 3 (after geo filter)", len(obs))
	}
	for _, o := range obs {
		if o.GeoID != 8741 {
			t.Errorf("geo filter missed entry: %+v", o)
		}
		if o.IndicatorID != 1001 {
			t.Errorf("indicator id = %d, want 1001", o.IndicatorID)
		}
	}
	if obs[0].Value != 87.43 || obs[2].Value != 91.20 {
		t.Errorf("values not decoded correctly: %+v", obs)
	}
	if seenAuth != "test-key" {
		t.Errorf("x-api-key header = %q, want test-key", seenAuth)
	}
	if !strings.Contains(seenAccept, "application/vnd.esios-api-v1+json") {
		t.Errorf("Accept header missing api-v1 vendor tag: %q", seenAccept)
	}
}

func TestFetchIndicator_NonOKStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	t.Cleanup(server.Close)

	client := NewESIOSClient("bad-key")
	client.baseURL = server.URL
	_, err := client.FetchIndicator(context.Background(), 1001, 8741, time.Now().Add(-time.Hour), time.Now(), "hour")
	if err == nil {
		t.Fatal("expected error for 401")
	}
	if !strings.Contains(err.Error(), "401") {
		t.Errorf("error should mention status: %v", err)
	}
}

func TestSeedAndFindIndicators(t *testing.T) {
	pool := getTestPool(t)
	repo := NewRepository(pool)

	test := []SeriesMeta{
		{IndicatorID: -1001, GeoID: 8741, Name: "Test PVPC", ShortName: "tPVPC", Category: "test_pricing", Unit: "EUR/MWh", GeoName: "España", Frequency: "hourly"},
	}
	if err := repo.SeedSeries(context.Background(), test); err != nil {
		t.Fatalf("seed: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), "DELETE FROM esios_observations WHERE indicator_id = -1001")
		_, _ = pool.Exec(context.Background(), "DELETE FROM esios_series WHERE indicator_id = -1001")
	})

	now := time.Now().UTC().Truncate(time.Hour)
	obs := []Observation{
		{IndicatorID: -1001, GeoID: 8741, Value: 80.0, DatetimeUTC: now.Add(-2 * time.Hour)},
		{IndicatorID: -1001, GeoID: 8741, Value: 85.5, DatetimeUTC: now.Add(-1 * time.Hour)},
	}
	if err := repo.SaveObservations(context.Background(), obs); err != nil {
		t.Fatalf("save: %v", err)
	}

	indicators, err := repo.FindIndicators(context.Background(), "test_pricing")
	if err != nil {
		t.Fatalf("find: %v", err)
	}
	if len(indicators) != 1 {
		t.Fatalf("len = %d, want 1", len(indicators))
	}
	got := indicators[0]
	if got.LatestValue != 85.5 {
		t.Errorf("LatestValue = %v, want 85.5", got.LatestValue)
	}
	if got.PrevValue != 80.0 {
		t.Errorf("PrevValue = %v, want 80.0", got.PrevValue)
	}
	if got.Change != 5.5 {
		t.Errorf("Change = %v, want 5.5", got.Change)
	}
}

func TestSaveObservationsUpsertsValue(t *testing.T) {
	pool := getTestPool(t)
	repo := NewRepository(pool)

	test := []SeriesMeta{
		{IndicatorID: -1002, GeoID: 8741, Name: "Test Upsert", ShortName: "tUp", Category: "test_pricing", Unit: "MW", GeoName: "España", Frequency: "hourly"},
	}
	_ = repo.SeedSeries(context.Background(), test)
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), "DELETE FROM esios_observations WHERE indicator_id = -1002")
		_, _ = pool.Exec(context.Background(), "DELETE FROM esios_series WHERE indicator_id = -1002")
	})

	dt := time.Date(2099, 1, 1, 12, 0, 0, 0, time.UTC)
	if err := repo.SaveObservations(context.Background(), []Observation{
		{IndicatorID: -1002, GeoID: 8741, Value: 100.0, DatetimeUTC: dt},
	}); err != nil {
		t.Fatalf("first save: %v", err)
	}
	if err := repo.SaveObservations(context.Background(), []Observation{
		{IndicatorID: -1002, GeoID: 8741, Value: 200.0, DatetimeUTC: dt},
	}); err != nil {
		t.Fatalf("second save: %v", err)
	}

	points, err := repo.FindObservations(context.Background(), -1002, 8741, dt.Add(-time.Hour), dt.Add(time.Hour), 10)
	if err != nil {
		t.Fatalf("find: %v", err)
	}
	if len(points) != 1 {
		t.Fatalf("len = %d, want 1 (upsert)", len(points))
	}
	if points[0].Value != 200.0 {
		t.Errorf("value = %v, want 200.0 (upsert must update)", points[0].Value)
	}
}

func TestIndicatorsEndpointPagination(t *testing.T) {
	pool := getTestPool(t)
	repo := NewRepository(pool)

	// Seed 3 series so we can paginate at size=2.
	test := []SeriesMeta{
		{IndicatorID: -2001, GeoID: 8741, Name: "Test A", ShortName: "A", Category: "test_pag", Unit: "MW", GeoName: "España", Frequency: "hourly"},
		{IndicatorID: -2002, GeoID: 8741, Name: "Test B", ShortName: "B", Category: "test_pag", Unit: "MW", GeoName: "España", Frequency: "hourly"},
		{IndicatorID: -2003, GeoID: 8741, Name: "Test C", ShortName: "C", Category: "test_pag", Unit: "MW", GeoName: "España", Frequency: "hourly"},
	}
	_ = repo.SeedSeries(context.Background(), test)
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), "DELETE FROM esios_series WHERE category = 'test_pag'")
	})

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/electricity/indicators", handleIndicators(repo))

	// Page 1
	req := httptest.NewRequest("GET", "/api/v1/electricity/indicators?category=test_pag&page_size=2", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("page1 status = %d", w.Code)
	}
	var page1 struct {
		Indicators    []Indicator `json:"indicators"`
		NextPageToken string      `json:"next_page_token"`
		TotalSize     int64       `json:"total_size"`
	}
	if err := json.NewDecoder(w.Body).Decode(&page1); err != nil {
		t.Fatalf("page1 decode: %v", err)
	}
	if len(page1.Indicators) != 2 || page1.NextPageToken == "" || page1.TotalSize != 3 {
		t.Fatalf("page1 unexpected: %+v", page1)
	}

	// Page 2 via cursor
	req = httptest.NewRequest("GET", "/api/v1/electricity/indicators?category=test_pag&page_size=2&page_token="+page1.NextPageToken, nil)
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	var page2 struct {
		Indicators    []Indicator `json:"indicators"`
		NextPageToken string      `json:"next_page_token"`
	}
	_ = json.NewDecoder(w.Body).Decode(&page2)
	if len(page2.Indicators) != 1 {
		t.Fatalf("page2 len = %d, want 1", len(page2.Indicators))
	}
	if page2.NextPageToken != "" {
		t.Errorf("page2 should be terminal, got next_page_token=%q", page2.NextPageToken)
	}
}

func TestIndicatorsEndpoint_NegativePageSizeReturns400(t *testing.T) {
	pool := getTestPool(t)
	repo := NewRepository(pool)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/electricity/indicators", handleIndicators(repo))

	req := httptest.NewRequest("GET", "/api/v1/electricity/indicators?page_size=-1", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
	var pd httpx.ProblemDetails
	_ = json.NewDecoder(w.Body).Decode(&pd)
	if pd.Code != "INVALID_ARGUMENT" {
		t.Errorf("code = %q, want INVALID_ARGUMENT", pd.Code)
	}
}
