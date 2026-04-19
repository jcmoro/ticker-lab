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
	if resp.Engine != "go-crypto" {
		t.Errorf("expected 'go-crypto', got '%s'", resp.Engine)
	}
}

func TestLatestEndpoint(t *testing.T) {
	pool := getTestPool(t)
	repo := NewRepository(pool)

	req := httptest.NewRequest("GET", "/api/v1/crypto/latest", nil)
	w := httptest.NewRecorder()

	handleLatest(repo)(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Count  int           `json:"count"`
		Date   string        `json:"date"`
		Prices []CryptoPrice `json:"prices"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode failed: %v", err)
	}

	if resp.Count == 0 {
		t.Skip("No crypto prices in database yet")
	}

	if resp.Count != len(resp.Prices) {
		t.Errorf("count %d != len(prices) %d", resp.Count, len(resp.Prices))
	}

	first := resp.Prices[0]
	if first.CoinID == "" || first.Symbol == "" || first.Name == "" {
		t.Error("expected non-empty coin metadata")
	}
	if first.PriceEUR <= 0 {
		t.Errorf("expected positive EUR price, got %f", first.PriceEUR)
	}
	if first.PriceUSD <= 0 {
		t.Errorf("expected positive USD price, got %f", first.PriceUSD)
	}
}

func TestHistoryEndpoint(t *testing.T) {
	pool := getTestPool(t)
	repo := NewRepository(pool)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/crypto/{id}/history", handleHistory(repo))

	req := httptest.NewRequest("GET", "/api/v1/crypto/bitcoin/history?days=30", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		CoinID string         `json:"coin_id"`
		Days   int            `json:"days"`
		Count  int            `json:"count"`
		Prices []HistoryPoint `json:"prices"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode failed: %v", err)
	}

	if resp.CoinID != "bitcoin" {
		t.Errorf("expected 'bitcoin', got '%s'", resp.CoinID)
	}
	if resp.Days != 30 {
		t.Errorf("expected 30 days, got %d", resp.Days)
	}
}

func TestHistoryEndpoint_DefaultDays(t *testing.T) {
	pool := getTestPool(t)
	repo := NewRepository(pool)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/crypto/{id}/history", handleHistory(repo))

	req := httptest.NewRequest("GET", "/api/v1/crypto/ethereum/history", nil)
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

	if resp.Days != 90 {
		t.Errorf("expected default 90 days, got %d", resp.Days)
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

func TestSaveAndFindLatest(t *testing.T) {
	pool := getTestPool(t)
	repo := NewRepository(pool)

	prices := []CryptoPrice{
		{CoinID: "test-coin", Symbol: "TST", Name: "Test Coin", PriceEUR: 123.45, PriceUSD: 145.67, MarketCap: 1000000, Change24h: 2.5, Date: "2026-04-19"},
	}

	if err := repo.Save(context.Background(), prices); err != nil {
		t.Fatalf("save failed: %v", err)
	}

	latest, err := repo.FindLatest(context.Background())
	if err != nil {
		t.Fatalf("find latest failed: %v", err)
	}

	found := false
	for _, p := range latest {
		if p.CoinID == "test-coin" {
			found = true
			if p.PriceEUR != 123.45 {
				t.Errorf("expected 123.45, got %f", p.PriceEUR)
			}
		}
	}
	if !found {
		t.Error("test-coin not found in latest results")
	}

	// Cleanup
	_, _ = pool.Exec(context.Background(), "DELETE FROM crypto_prices WHERE coin_id = 'test-coin'")
}
