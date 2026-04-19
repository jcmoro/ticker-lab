package main

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"
)

func handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, HealthResponse{
		Status:    "ok",
		Engine:    "go-crypto",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
}

func handleLatest(repo *Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		prices, err := repo.FindLatest(r.Context())
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, ProblemDetails{
				Type: "https://tickerlab.dev/problems/internal-error", Title: "Internal Server Error",
				Status: 500, Detail: "Failed to fetch prices", Code: "INTERNAL_ERROR",
			})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"date":   dateFromPrices(prices),
			"count":  len(prices),
			"prices": prices,
		})
	}
}

func handleHistory(repo *Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		coinID := r.PathValue("id")
		if coinID == "" {
			writeJSON(w, http.StatusBadRequest, ProblemDetails{
				Type: "https://tickerlab.dev/problems/bad-request", Title: "Bad Request",
				Status: 400, Detail: "Coin ID is required", Code: "MISSING_COIN_ID",
			})
			return
		}

		days := 90
		if d := r.URL.Query().Get("days"); d != "" {
			if parsed, err := strconv.Atoi(d); err == nil && parsed > 0 {
				days = parsed
			}
		}

		points, err := repo.FindHistory(r.Context(), coinID, days)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, ProblemDetails{
				Type: "https://tickerlab.dev/problems/internal-error", Title: "Internal Server Error",
				Status: 500, Detail: "Failed to fetch history", Code: "INTERNAL_ERROR",
			})
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"coin_id": coinID,
			"days":    days,
			"count":   len(points),
			"prices":  points,
		})
	}
}

func dateFromPrices(prices []CryptoPrice) string {
	if len(prices) > 0 {
		return prices[0].Date
	}
	return ""
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
