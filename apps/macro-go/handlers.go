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
		Engine:    "go-macro",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
}

func handleIndicators(repo *Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		category := r.URL.Query().Get("category")

		indicators, err := repo.FindIndicators(r.Context(), category)
		if err != nil {
			httpx.WriteProblem(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to fetch indicators")
			return
		}

		httpx.WriteJSON(w, http.StatusOK, map[string]any{
			"count":      len(indicators),
			"indicators": indicators,
		})
	}
}

func handleHistory(repo *Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		source := r.PathValue("source")
		seriesID := r.PathValue("id")
		if source == "" || seriesID == "" {
			httpx.WriteProblem(w, http.StatusBadRequest, "MISSING_PARAMS", "Source and series ID are required")
			return
		}

		days := 365
		if d := r.URL.Query().Get("days"); d != "" {
			if parsed, err := strconv.Atoi(d); err == nil && parsed > 0 {
				days = parsed
			}
		}

		points, err := repo.FindHistory(r.Context(), source, seriesID, days)
		if err != nil {
			httpx.WriteProblem(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to fetch history")
			return
		}

		name, _ := repo.FindSeriesName(r.Context(), source, seriesID)

		httpx.WriteJSON(w, http.StatusOK, map[string]any{
			"source":    source,
			"series_id": seriesID,
			"name":      name,
			"days":      days,
			"count":     len(points),
			"points":    points,
		})
	}
}
