package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type ConversionResponse struct {
	From   string  `json:"from"`
	To     string  `json:"to"`
	Amount float64 `json:"amount"`
	Rate   float64 `json:"rate"`
	Result float64 `json:"result"`
	Date   string  `json:"date"`
	Engine string  `json:"engine"`
}

type HealthResponse struct {
	Status    string `json:"status"`
	Engine    string `json:"engine"`
	Timestamp string `json:"timestamp"`
}

type ProblemDetails struct {
	Type   string `json:"type"`
	Title  string `json:"title"`
	Status int    `json:"status"`
	Detail string `json:"detail"`
	Code   string `json:"code"`
}

type rateRow struct {
	Currency string
	Rate     float64
	Date     string
}

func main() {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL environment variable is required")
	}

	pool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v", err)
	}
	defer pool.Close()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/go/convert", handleConvert(pool))
	mux.HandleFunc("GET /health", handleHealth)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	handler := corsMiddleware(mux)

	log.Printf("Converter Go listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, handler))
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, HealthResponse{
		Status:    "ok",
		Engine:    "go",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
}

func handleConvert(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		from := r.URL.Query().Get("from")
		to := r.URL.Query().Get("to")
		amountStr := r.URL.Query().Get("amount")

		if from == "" || to == "" {
			writeJSON(w, http.StatusBadRequest, ProblemDetails{
				Type:   "https://tickerlab.dev/problems/bad-request",
				Title:  "Bad Request",
				Status: 400,
				Detail: "Both 'from' and 'to' query parameters are required",
				Code:   "MISSING_PARAMETERS",
			})
			return
		}

		amount := 1.0
		if amountStr != "" {
			if parsed, err := strconv.ParseFloat(amountStr, 64); err == nil {
				amount = parsed
			}
		}

		rates, err := getLatestRates(r.Context(), pool)
		if err != nil {
			log.Printf("Database error: %v", err)
			writeJSON(w, http.StatusInternalServerError, ProblemDetails{
				Type:   "https://tickerlab.dev/problems/internal-server-error",
				Title:  "Internal Server Error",
				Status: 500,
				Detail: "An unexpected error occurred",
				Code:   "INTERNAL_ERROR",
			})
			return
		}

		if len(rates) == 0 {
			writeJSON(w, http.StatusNotFound, ProblemDetails{
				Type:   "https://tickerlab.dev/problems/not-found",
				Title:  "Not Found",
				Status: 404,
				Detail: "No exchange rates available",
				Code:   "RATES_NOT_FOUND",
			})
			return
		}

		rateMap := make(map[string]float64)
		rateMap["EUR"] = 1.0
		date := rates[0].Date
		for _, row := range rates {
			rateMap[row.Currency] = row.Rate
		}

		fromRate, fromOk := rateMap[from]
		toRate, toOk := rateMap[to]

		if !fromOk || !toOk {
			writeJSON(w, http.StatusNotFound, ProblemDetails{
				Type:   "https://tickerlab.dev/problems/not-found",
				Title:  "Not Found",
				Status: 404,
				Detail: fmt.Sprintf("Cannot convert %s to %s. Currency not available.", from, to),
				Code:   "CURRENCY_NOT_FOUND",
			})
			return
		}

		rate := toRate / fromRate
		result := math.Round(amount*rate*100) / 100
		roundedRate := math.Round(rate*1_000_000) / 1_000_000

		writeJSON(w, http.StatusOK, ConversionResponse{
			From:   from,
			To:     to,
			Amount: amount,
			Rate:   roundedRate,
			Result: result,
			Date:   date,
			Engine: "go",
		})
	}
}

func getLatestRates(ctx context.Context, pool *pgxpool.Pool) ([]rateRow, error) {
	query := `
		SELECT quote_currency, rate::float8, date::text
		FROM exchange_rates
		WHERE base_currency = 'EUR'
		  AND date = (SELECT MAX(date) FROM exchange_rates WHERE base_currency = 'EUR')
		ORDER BY quote_currency
	`
	rows, err := pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rates []rateRow
	for rows.Next() {
		var r rateRow
		if err := rows.Scan(&r.Currency, &r.Rate, &r.Date); err != nil {
			return nil, err
		}
		rates = append(rates, r)
	}
	return rates, rows.Err()
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
