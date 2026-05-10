package main

import (
	"net/http"
	"strconv"
	"time"

	"github.com/ticker-lab/httpx"
)

func handleHealth(w http.ResponseWriter, _ *http.Request) {
	httpx.WriteJSON(w, http.StatusOK, HealthResponse{
		Status:    "ok",
		Engine:    "go-crypto",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
}

func handleLatest(repo *Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		prices, err := repo.FindLatest(r.Context())
		if err != nil {
			httpx.WriteProblem(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to fetch prices")
			return
		}
		httpx.WriteJSON(w, http.StatusOK, map[string]any{
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
			httpx.WriteProblem(w, http.StatusBadRequest, "MISSING_COIN_ID", "Coin ID is required")
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
			httpx.WriteProblem(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to fetch history")
			return
		}

		httpx.WriteJSON(w, http.StatusOK, map[string]any{
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
