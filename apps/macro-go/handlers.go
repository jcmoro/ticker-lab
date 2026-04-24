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
		Engine:    "go-macro",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
}

func handleIndicators(repo *Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		category := r.URL.Query().Get("category")

		indicators, err := repo.FindIndicators(r.Context(), category)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, ProblemDetails{
				Type: "https://tickerlab.dev/problems/internal-error", Title: "Internal Server Error",
				Status: 500, Detail: "Failed to fetch indicators", Code: "INTERNAL_ERROR",
			})
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
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
			writeJSON(w, http.StatusBadRequest, ProblemDetails{
				Type: "https://tickerlab.dev/problems/bad-request", Title: "Bad Request",
				Status: 400, Detail: "Source and series ID are required", Code: "MISSING_PARAMS",
			})
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
			writeJSON(w, http.StatusInternalServerError, ProblemDetails{
				Type: "https://tickerlab.dev/problems/internal-error", Title: "Internal Server Error",
				Status: 500, Detail: "Failed to fetch history", Code: "INTERNAL_ERROR",
			})
			return
		}

		name, _ := repo.FindSeriesName(r.Context(), source, seriesID)

		writeJSON(w, http.StatusOK, map[string]any{
			"source":    source,
			"series_id": seriesID,
			"name":      name,
			"days":      days,
			"count":     len(points),
			"points":    points,
		})
	}
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
