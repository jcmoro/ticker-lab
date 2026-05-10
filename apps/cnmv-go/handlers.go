package main

import (
	"errors"
	"net/http"
	"regexp"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/ticker-lab/httpx"
)

var isinPattern = regexp.MustCompile(`^[A-Z]{2}[A-Z0-9]{9}[0-9]$`)

func handleHealth(w http.ResponseWriter, _ *http.Request) {
	httpx.WriteJSON(w, http.StatusOK, HealthResponse{
		Status:    "ok",
		Engine:    "go-cnmv",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
}

// handleFunds serves GET /api/v1/funds?tipo=&gestora=&q=&page_size=&page_token=
func handleFunds(repo *Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		page, err := httpx.ParsePagination(r, 50, 500)
		if err != nil {
			writePaginationError(w, err)
			return
		}
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

		q := FundQuery{
			Tipo:    r.URL.Query().Get("tipo"),
			Gestora: r.URL.Query().Get("gestora"),
			Q:       r.URL.Query().Get("q"),
			Limit:   page.Size + 1,
			Offset:  offset,
		}

		funds, total, err := repo.FindFunds(r.Context(), q)
		if err != nil {
			httpx.WriteProblem(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to fetch funds")
			return
		}

		nextToken := ""
		if len(funds) > page.Size {
			funds = funds[:page.Size]
			nextToken, _ = httpx.EncodeCursor(map[string]int{"offset": offset + page.Size})
		}

		httpx.WriteJSON(w, http.StatusOK, struct {
			Funds []FundSummary `json:"funds"`
			httpx.Page
		}{
			Funds: funds,
			Page:  httpx.NewPage(nextToken, &total),
		})
	}
}

// handleFundDetail serves GET /api/v1/funds/{isin}
func handleFundDetail(repo *Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		isin := r.PathValue("isin")
		if !isinPattern.MatchString(isin) {
			httpx.WriteProblem(w, http.StatusBadRequest, "INVALID_ARGUMENT", "isin does not match ISO 6166 (^[A-Z]{2}[A-Z0-9]{9}[0-9]$)")
			return
		}
		fund, err := repo.FindFundByISIN(r.Context(), isin)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				httpx.WriteProblem(w, http.StatusNotFound, "NOT_FOUND", "no fund with that ISIN")
				return
			}
			httpx.WriteProblem(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to fetch fund")
			return
		}
		httpx.WriteJSON(w, http.StatusOK, fund)
	}
}

// handleNavObservations serves the AIP-122 nested resource:
// GET /api/v1/funds/{isin}/nav-observations?start_date=&end_date=&page_size=&page_token=
func handleNavObservations(repo *Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		isin := r.PathValue("isin")
		if !isinPattern.MatchString(isin) {
			httpx.WriteProblem(w, http.StatusBadRequest, "INVALID_ARGUMENT", "isin does not match ISO 6166")
			return
		}

		page, err := httpx.ParsePagination(r, 365, 3650)
		if err != nil {
			writePaginationError(w, err)
			return
		}

		start := r.URL.Query().Get("start_date")
		end := r.URL.Query().Get("end_date")
		// Default range: last 365 days.
		if end == "" {
			end = time.Now().UTC().Format("2006-01-02")
		}
		if start == "" {
			start = time.Now().UTC().AddDate(-1, 0, 0).Format("2006-01-02")
		}

		// page_token, when present, encodes the last seen date — caller
		// advances by overriding start_date with the next day.
		if page.Token != "" {
			var cursor struct {
				After string `json:"after"`
			}
			if err := httpx.DecodeCursor(page.Token, &cursor); err != nil {
				writePaginationError(w, err)
				return
			}
			// Next day after the cursor anchor.
			t, perr := time.Parse("2006-01-02", cursor.After)
			if perr != nil {
				httpx.WriteProblem(w, http.StatusBadRequest, "INVALID_ARGUMENT", "page_token is malformed")
				return
			}
			start = t.AddDate(0, 0, 1).Format("2006-01-02")
		}

		points, err := repo.FindNAVObservations(r.Context(), isin, start, end, page.Size+1)
		if err != nil {
			httpx.WriteProblem(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to fetch nav observations")
			return
		}

		nextToken := ""
		if len(points) > page.Size {
			points = points[:page.Size]
			nextToken, _ = httpx.EncodeCursor(map[string]string{"after": points[len(points)-1].Date})
		}

		httpx.WriteJSON(w, http.StatusOK, struct {
			ISIN   string     `json:"isin"`
			Points []NavPoint `json:"points"`
			httpx.Page
		}{
			ISIN:   isin,
			Points: points,
			Page:   httpx.NewPage(nextToken, nil),
		})
	}
}

func writePaginationError(w http.ResponseWriter, err error) {
	var pe *httpx.PaginationError
	if errors.As(err, &pe) {
		httpx.WriteProblem(w, pe.Status, pe.Code, pe.Message)
		return
	}
	httpx.WriteProblem(w, http.StatusBadRequest, "INVALID_ARGUMENT", err.Error())
}
