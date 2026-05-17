package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"math"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ticker-lab/httpx"
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

	handler := httpx.CORSMiddleware(mux)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := httpx.Run(ctx, ":"+port, handler, 15*time.Second); err != nil {
		slog.Error("server failed", "service", "converter-go", "err", err)
		os.Exit(1)
	}
}

func handleHealth(w http.ResponseWriter, _ *http.Request) {
	httpx.WriteJSON(w, http.StatusOK, HealthResponse{
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
			httpx.WriteProblem(w, http.StatusBadRequest, "MISSING_PARAMETERS",
				"Both 'from' and 'to' query parameters are required")
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
			httpx.WriteProblem(w, http.StatusInternalServerError, "INTERNAL_ERROR",
				"An unexpected error occurred")
			return
		}

		if len(rates) == 0 {
			httpx.WriteProblem(w, http.StatusNotFound, "RATES_NOT_FOUND",
				"No exchange rates available")
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
			httpx.WriteProblem(w, http.StatusNotFound, "CURRENCY_NOT_FOUND",
				fmt.Sprintf("Cannot convert %s to %s. Currency not available.", from, to))
			return
		}

		rate := toRate / fromRate
		result := math.Round(amount*rate*100) / 100
		roundedRate := math.Round(rate*1_000_000) / 1_000_000

		httpx.WriteJSON(w, http.StatusOK, ConversionResponse{
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
