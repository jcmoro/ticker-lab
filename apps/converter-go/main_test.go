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
	t.Cleanup(func() { pool.Close() })
	return pool
}

func TestHealthEndpoint(t *testing.T) {
	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	handleHealth(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var resp HealthResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Status != "ok" {
		t.Errorf("expected status 'ok', got '%s'", resp.Status)
	}
	if resp.Engine != "go" {
		t.Errorf("expected engine 'go', got '%s'", resp.Engine)
	}
	if resp.Timestamp == "" {
		t.Error("expected non-empty timestamp")
	}
}

func TestConvertEndpoint_MissingParams(t *testing.T) {
	pool := getTestPool(t)
	handler := handleConvert(pool)

	tests := []struct {
		name  string
		query string
	}{
		{"missing both", ""},
		{"missing to", "?from=EUR"},
		{"missing from", "?to=USD"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/v1/go/convert"+tc.query, nil)
			w := httptest.NewRecorder()

			handler(w, req)

			if w.Code != http.StatusBadRequest {
				t.Errorf("expected 400, got %d", w.Code)
			}
		})
	}
}

func TestConvertEndpoint_EURtoUSD(t *testing.T) {
	pool := getTestPool(t)
	handler := handleConvert(pool)

	req := httptest.NewRequest("GET", "/api/v1/go/convert?from=EUR&to=USD&amount=100", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp ConversionResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}

	if resp.From != "EUR" {
		t.Errorf("expected from 'EUR', got '%s'", resp.From)
	}
	if resp.To != "USD" {
		t.Errorf("expected to 'USD', got '%s'", resp.To)
	}
	if resp.Amount != 100 {
		t.Errorf("expected amount 100, got %f", resp.Amount)
	}
	if resp.Rate <= 0 {
		t.Errorf("expected positive rate, got %f", resp.Rate)
	}
	if resp.Result <= 0 {
		t.Errorf("expected positive result, got %f", resp.Result)
	}
	if resp.Engine != "go" {
		t.Errorf("expected engine 'go', got '%s'", resp.Engine)
	}
	if resp.Date == "" {
		t.Error("expected non-empty date")
	}
}

func TestConvertEndpoint_CrossRate(t *testing.T) {
	pool := getTestPool(t)
	handler := handleConvert(pool)

	req := httptest.NewRequest("GET", "/api/v1/go/convert?from=GBP&to=JPY&amount=1000", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp ConversionResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}

	if resp.From != "GBP" || resp.To != "JPY" {
		t.Errorf("expected GBP->JPY, got %s->%s", resp.From, resp.To)
	}
	// GBP->JPY via EUR should be > 200 (historically ~180-220)
	if resp.Rate < 100 || resp.Rate > 400 {
		t.Errorf("cross-rate GBP/JPY seems wrong: %f", resp.Rate)
	}
	if resp.Result < 100000 {
		t.Errorf("1000 GBP in JPY should be > 100000, got %f", resp.Result)
	}
}

func TestConvertEndpoint_SameCurrency(t *testing.T) {
	pool := getTestPool(t)
	handler := handleConvert(pool)

	req := httptest.NewRequest("GET", "/api/v1/go/convert?from=EUR&to=EUR&amount=50", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp ConversionResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}

	if resp.Rate != 1 {
		t.Errorf("expected rate 1 for same currency, got %f", resp.Rate)
	}
	if resp.Result != 50 {
		t.Errorf("expected result 50 for same currency, got %f", resp.Result)
	}
}

func TestConvertEndpoint_UnknownCurrency(t *testing.T) {
	pool := getTestPool(t)
	handler := handleConvert(pool)

	req := httptest.NewRequest("GET", "/api/v1/go/convert?from=EUR&to=XYZ&amount=100", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestConvertEndpoint_DefaultAmount(t *testing.T) {
	pool := getTestPool(t)
	handler := handleConvert(pool)

	req := httptest.NewRequest("GET", "/api/v1/go/convert?from=EUR&to=USD", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp ConversionResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}

	if resp.Amount != 1 {
		t.Errorf("expected default amount 1, got %f", resp.Amount)
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
		t.Errorf("expected CORS origin '*', got '%s'", got)
	}

	// OPTIONS preflight
	req = httptest.NewRequest("OPTIONS", "/test", nil)
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for OPTIONS, got %d", w.Code)
	}
}
