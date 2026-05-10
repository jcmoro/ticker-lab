package main

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/ticker-lab/httpx"
)

func handleHealth(w http.ResponseWriter, _ *http.Request) {
	httpx.WriteJSON(w, http.StatusOK, HealthResponse{
		Status:    "ok",
		Engine:    "go-esios",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
}

// handleIndicators serves GET /api/v1/electricity/indicators with AIP-158
// pagination. The `category` filter is a simple equality match.
func handleIndicators(repo *Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		page, err := httpx.ParsePagination(r, 100, 500)
		if err != nil {
			writePaginationError(w, err)
			return
		}

		category := r.URL.Query().Get("category")
		all, err := repo.FindIndicators(r.Context(), category)
		if err != nil {
			httpx.WriteProblem(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to fetch indicators")
			return
		}

		// In-memory pagination: there are at most ~14 series across all
		// geos and categories; a cursor-DB pagination would be overkill.
		offset := 0
		if page.Token != "" {
			var cursor struct {
				Offset int `json:"offset"`
			}
			if err := httpx.DecodeCursor(page.Token, &cursor); err != nil {
				writePaginationError(w, err)
				return
			}
			offset = cursor.Offset
		}

		end := offset + page.Size
		if end > len(all) {
			end = len(all)
		}
		items := all[offset:end]

		nextToken := ""
		if end < len(all) {
			nextToken, _ = httpx.EncodeCursor(map[string]int{"offset": end})
		}
		total := int64(len(all))

		httpx.WriteJSON(w, http.StatusOK, struct {
			Indicators []Indicator `json:"indicators"`
			httpx.Page
		}{
			Indicators: items,
			Page:       httpx.NewPage(nextToken, &total),
		})
	}
}

// handleObservations serves the AIP-122 nested resource path:
// /api/v1/electricity/indicators/{indicator_id}/geos/{geo_id}/observations
// with start_date, end_date and pagination params.
func handleObservations(repo *Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		indicatorID, err := strconv.Atoi(r.PathValue("indicator_id"))
		if err != nil {
			httpx.WriteProblem(w, http.StatusBadRequest, "INVALID_ARGUMENT", "indicator_id must be an integer")
			return
		}
		geoID, err := strconv.Atoi(r.PathValue("geo_id"))
		if err != nil {
			httpx.WriteProblem(w, http.StatusBadRequest, "INVALID_ARGUMENT", "geo_id must be an integer")
			return
		}

		page, err := httpx.ParsePagination(r, 168, 8760)
		if err != nil {
			writePaginationError(w, err)
			return
		}

		end := time.Now().UTC()
		start := end.Add(-7 * 24 * time.Hour)
		if v := r.URL.Query().Get("start_date"); v != "" {
			t, err := parseDate(v)
			if err != nil {
				httpx.WriteProblem(w, http.StatusBadRequest, "INVALID_ARGUMENT", "start_date must be RFC3339 or YYYY-MM-DD")
				return
			}
			start = t
		}
		if v := r.URL.Query().Get("end_date"); v != "" {
			t, err := parseDate(v)
			if err != nil {
				httpx.WriteProblem(w, http.StatusBadRequest, "INVALID_ARGUMENT", "end_date must be RFC3339 or YYYY-MM-DD")
				return
			}
			end = t
		}

		// page_token, when present, supersedes start_date — it points
		// just past the previous page's last observation.
		if page.Token != "" {
			var cursor struct {
				After string `json:"after"`
			}
			if err := httpx.DecodeCursor(page.Token, &cursor); err != nil {
				writePaginationError(w, err)
				return
			}
			t, err := time.Parse(time.RFC3339, cursor.After)
			if err != nil {
				httpx.WriteProblem(w, http.StatusBadRequest, "INVALID_ARGUMENT", "page_token cursor is malformed")
				return
			}
			// Add 1ns so the cursor is exclusive of its anchor.
			start = t.Add(time.Nanosecond)
		}

		// Pull one extra row so we can detect "more available" without a
		// second query.
		points, err := repo.FindObservations(r.Context(), indicatorID, geoID, start, end, page.Size+1)
		if err != nil {
			httpx.WriteProblem(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to fetch observations")
			return
		}

		nextToken := ""
		if len(points) > page.Size {
			points = points[:page.Size]
			last := points[len(points)-1].DatetimeUTC
			nextToken, _ = httpx.EncodeCursor(map[string]string{"after": last.Format(time.RFC3339Nano)})
		}

		httpx.WriteJSON(w, http.StatusOK, struct {
			IndicatorID int            `json:"indicator_id"`
			GeoID       int            `json:"geo_id"`
			Points      []HistoryPoint `json:"points"`
			httpx.Page
		}{
			IndicatorID: indicatorID,
			GeoID:       geoID,
			Points:      points,
			Page:        httpx.NewPage(nextToken, nil),
		})
	}
}

// parseDate accepts either RFC3339 or a bare YYYY-MM-DD (assumed UTC start
// of day). Hourly time-series users will typically pass RFC3339; SSR pages
// pass dates.
func parseDate(s string) (time.Time, error) {
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t.UTC(), nil
	}
	if t, err := time.Parse("2006-01-02", s); err == nil {
		return t.UTC(), nil
	}
	return time.Time{}, errors.New("unrecognized date format")
}

func writePaginationError(w http.ResponseWriter, err error) {
	var pe *httpx.PaginationError
	if errors.As(err, &pe) {
		httpx.WriteProblem(w, pe.Status, pe.Code, pe.Message)
		return
	}
	httpx.WriteProblem(w, http.StatusBadRequest, "INVALID_ARGUMENT", err.Error())
}
