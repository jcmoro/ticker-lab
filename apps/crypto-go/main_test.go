package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"sync"
	"testing"

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

	handler := httpx.CORSMiddleware(mux)

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

	// Use a far-future date so this row is guaranteed to be MAX(date) in
	// FindLatest regardless of whatever real data the dev DB already holds.
	const testDate = "2099-12-31"
	ctx := context.Background()
	t.Cleanup(func() {
		_, _ = pool.Exec(ctx, "DELETE FROM crypto_prices WHERE coin_id = 'test-coin' OR date = $1", testDate)
	})

	prices := []CryptoPrice{
		{CoinID: "test-coin", Symbol: "TST", Name: "Test Coin", PriceEUR: 123.45, PriceUSD: 145.67, MarketCap: 1000000, Change24h: 2.5, Date: testDate},
	}

	if err := repo.Save(ctx, prices); err != nil {
		t.Fatalf("save failed: %v", err)
	}

	latest, err := repo.FindLatest(ctx)
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
}

// TestLatestEndpoint_OrderingByMarketCap seeds three test prices at the
// canonical far-future date and asserts the handler returns them ordered
// by market_cap_eur DESC (per repository contract). Other rows that happen
// to share the same MAX(date) are tolerated by filtering on the test prefix.
func TestLatestEndpoint_OrderingByMarketCap(t *testing.T) {
	pool := getTestPool(t)
	repo := NewRepository(pool)

	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), "DELETE FROM crypto_prices WHERE coin_id LIKE 'integ-ord-%'")
	})

	prices := []CryptoPrice{
		{CoinID: "integ-ord-small", Symbol: "SML", Name: "Small Cap", PriceEUR: 10, PriceUSD: 11, MarketCap: 100, Date: "2099-12-31"},
		{CoinID: "integ-ord-large", Symbol: "LRG", Name: "Large Cap", PriceEUR: 30, PriceUSD: 33, MarketCap: 300, Date: "2099-12-31"},
		{CoinID: "integ-ord-mid", Symbol: "MID", Name: "Mid Cap", PriceEUR: 20, PriceUSD: 22, MarketCap: 200, Date: "2099-12-31"},
	}
	if err := repo.Save(context.Background(), prices); err != nil {
		t.Fatalf("seed: %v", err)
	}

	req := httptest.NewRequest("GET", "/api/v1/crypto/latest", nil)
	w := httptest.NewRecorder()
	handleLatest(repo)(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", w.Code, w.Body.String())
	}

	var resp struct {
		Prices []CryptoPrice `json:"prices"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}

	// Extract only our seeded rows in the order returned.
	var seen []string
	for _, p := range resp.Prices {
		if len(p.CoinID) >= 10 && p.CoinID[:10] == "integ-ord-" {
			seen = append(seen, p.CoinID)
		}
	}
	want := []string{"integ-ord-large", "integ-ord-mid", "integ-ord-small"}
	if len(seen) != len(want) {
		t.Fatalf("seeded %d rows, response surfaced %d: %v", len(want), len(seen), seen)
	}
	for i := range want {
		if seen[i] != want[i] {
			t.Errorf("position %d: got %q, want %q (full order: %v)", i, seen[i], want[i], seen)
		}
	}
}

// TestHistoryEndpoint_FiltersByDays asserts the days param actually restricts
// the returned points to [today - days, today].
func TestHistoryEndpoint_FiltersByDays(t *testing.T) {
	pool := getTestPool(t)
	repo := NewRepository(pool)

	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), "DELETE FROM crypto_prices WHERE coin_id = 'integ-hist-coin'")
	})

	// Use CURRENT_DATE - N so the days filter (date >= CURRENT_DATE - days)
	// is exercised. 50/25/5 days ago lets us split: days=30 → 2 rows.
	_, err := pool.Exec(context.Background(), `
		INSERT INTO crypto_prices (coin_id, symbol, name, price_eur, price_usd, market_cap_eur, date)
		VALUES
			('integ-hist-coin', 'IHC', 'Integ Hist Coin', 1.0, 1.1, 1000, CURRENT_DATE - 50),
			('integ-hist-coin', 'IHC', 'Integ Hist Coin', 1.5, 1.6, 1500, CURRENT_DATE - 25),
			('integ-hist-coin', 'IHC', 'Integ Hist Coin', 2.0, 2.1, 2000, CURRENT_DATE - 5)
	`)
	if err != nil {
		t.Fatalf("seed: %v", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/crypto/{id}/history", handleHistory(repo))

	cases := []struct {
		days    int
		wantLen int
	}{
		{days: 30, wantLen: 2},
		{days: 60, wantLen: 3},
		{days: 7, wantLen: 1},
	}
	for _, c := range cases {
		req := httptest.NewRequest("GET", "/api/v1/crypto/integ-hist-coin/history?days="+strconv.Itoa(c.days), nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("days=%d status = %d", c.days, w.Code)
		}
		var resp struct {
			Count  int            `json:"count"`
			Prices []HistoryPoint `json:"prices"`
		}
		_ = json.NewDecoder(w.Body).Decode(&resp)
		if resp.Count != c.wantLen {
			t.Errorf("days=%d count = %d, want %d (prices=%v)", c.days, resp.Count, c.wantLen, resp.Prices)
		}
	}
}

// TestHistoryEndpoint_InvalidDaysDefaultsTo90 asserts the handler's
// "Atoi err == nil && parsed > 0" guard falls back to 90 for garbage input.
func TestHistoryEndpoint_InvalidDaysDefaultsTo90(t *testing.T) {
	pool := getTestPool(t)
	repo := NewRepository(pool)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/crypto/{id}/history", handleHistory(repo))

	for _, raw := range []string{"abc", "-5", "0"} {
		req := httptest.NewRequest("GET", "/api/v1/crypto/bitcoin/history?days="+raw, nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("days=%q status = %d", raw, w.Code)
		}
		var resp struct {
			Days int `json:"days"`
		}
		_ = json.NewDecoder(w.Body).Decode(&resp)
		if resp.Days != 90 {
			t.Errorf("days=%q: got %d, want 90 (default)", raw, resp.Days)
		}
	}
}

// TestHistoryEndpoint_UnknownCoinReturnsEmpty confirms that an unknown
// coin_id is not an error — handler returns 200 with count=0.
func TestHistoryEndpoint_UnknownCoinReturnsEmpty(t *testing.T) {
	pool := getTestPool(t)
	repo := NewRepository(pool)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/crypto/{id}/history", handleHistory(repo))

	req := httptest.NewRequest("GET", "/api/v1/crypto/this-coin-does-not-exist/history?days=30", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	var resp struct {
		CoinID string         `json:"coin_id"`
		Count  int            `json:"count"`
		Prices []HistoryPoint `json:"prices"`
	}
	_ = json.NewDecoder(w.Body).Decode(&resp)
	if resp.Count != 0 {
		t.Errorf("count = %d, want 0", resp.Count)
	}
	if len(resp.Prices) != 0 {
		t.Errorf("prices len = %d, want 0", len(resp.Prices))
	}
}

// TestMigrate_ConcurrentSafe runs Migrate from N goroutines simultaneously
// against a shared pool. The advisory lock should serialize them so all
// callers succeed without errors and the schema lands in its final state.
func TestMigrate_ConcurrentSafe(t *testing.T) {
	pool := getTestPool(t)

	const n = 8
	var wg sync.WaitGroup
	errs := make(chan error, n)
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			errs <- NewRepository(pool).Migrate(context.Background())
		}()
	}
	wg.Wait()
	close(errs)

	for err := range errs {
		if err != nil {
			t.Errorf("concurrent Migrate failed: %v", err)
		}
	}
}
