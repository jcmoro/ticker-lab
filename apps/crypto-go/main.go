package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

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

	repo := NewRepository(pool)

	if err := repo.Migrate(context.Background()); err != nil {
		log.Fatalf("Migration failed: %v", err)
	}

	// Subcommands
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "ingest":
			runIngest(repo)
			return
		case "backfill":
			days := 365
			if len(os.Args) > 2 {
				if d, err := fmt.Sscanf(os.Args[2], "%d", &days); d == 0 || err != nil {
					days = 365
				}
			}
			runBackfill(repo, days)
			return
		}
	}

	// Default: serve HTTP
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", handleHealth)
	mux.HandleFunc("GET /api/v1/crypto/latest", handleLatest(repo))
	mux.HandleFunc("GET /api/v1/crypto/{id}/history", handleHistory(repo))

	handler := corsMiddleware(mux)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8090"
	}

	log.Printf("Crypto Go listening on :%s", port)
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

func runIngest(repo *Repository) {
	client := NewCoinGeckoClient()

	log.Printf("Fetching prices for %d coins from CoinGecko...", len(topCoins))
	prices, err := client.FetchPrices(topCoins)
	if err != nil {
		log.Fatalf("Fetch failed: %v", err)
	}

	log.Printf("Saving %d prices to database...", len(prices))
	if err := repo.Save(context.Background(), prices); err != nil {
		log.Fatalf("Save failed: %v", err)
	}

	fmt.Printf("Done: %d crypto prices ingested.\n", len(prices))
}

func runBackfill(repo *Repository, days int) {
	client := NewCoinGeckoClient()
	total := 0

	for i, coin := range topCoins {
		if i > 0 {
			log.Printf("  Waiting 15s (rate limit)...")
			time.Sleep(15 * time.Second)
		}

		log.Printf("[%d/%d] Fetching %s (%d days)...", i+1, len(topCoins), coin.Symbol, days)

		prices, err := client.FetchHistory(coin.ID, days)
		if err != nil {
			log.Printf("  Error: %v (skipping)", err)
			continue
		}

		if len(prices) > 0 {
			if err := repo.Save(context.Background(), prices); err != nil {
				log.Printf("  Save error: %v (skipping)", err)
				continue
			}
			log.Printf("  Saved %d data points", len(prices))
			total += len(prices)
		}
	}

	fmt.Printf("Done: %d total crypto data points backfilled.\n", total)
}
