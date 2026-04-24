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

	allSeries := append(fredSeries, ecbSeries...)
	if err := repo.SeedSeries(context.Background(), allSeries); err != nil {
		log.Fatalf("Seed series failed: %v", err)
	}

	// Subcommands
	if len(os.Args) > 1 {
		fredAPIKey := os.Getenv("FRED_API_KEY")

		switch os.Args[1] {
		case "ingest":
			if fredAPIKey == "" {
				log.Fatal("FRED_API_KEY environment variable is required for ingest")
			}
			runIngestFRED(repo, NewFREDClient(fredAPIKey))
			return
		case "ingest-ecb":
			runIngestECB(repo, NewECBClient())
			return
		case "backfill":
			if fredAPIKey == "" {
				log.Fatal("FRED_API_KEY environment variable is required for backfill")
			}
			runBackfill(repo, NewFREDClient(fredAPIKey), NewECBClient())
			return
		}
	}

	// Default: serve HTTP
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", handleHealth)
	mux.HandleFunc("GET /api/v1/macro/indicators", handleIndicators(repo))
	mux.HandleFunc("GET /api/v1/macro/{source}/{id}/history", handleHistory(repo))

	handler := corsMiddleware(mux)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8110"
	}

	log.Printf("Macro Go listening on :%s", port)
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

func runIngestFRED(repo *Repository, client *FREDClient) {
	total := 0
	for i, s := range fredSeries {
		if i > 0 {
			time.Sleep(500 * time.Millisecond)
		}

		lastDate, _ := repo.GetLastSyncDate(context.Background(), s.Source, s.SeriesID)
		log.Printf("[FRED %d/%d] Fetching %s (since %s)...", i+1, len(fredSeries), s.SeriesID, lastDate)

		obs, err := client.FetchObservations(s.SeriesID, lastDate)
		if err != nil {
			log.Printf("  Error: %v (skipping)", err)
			continue
		}

		if len(obs) > 0 {
			if err := repo.SaveObservations(context.Background(), obs); err != nil {
				log.Printf("  Save error: %v (skipping)", err)
				continue
			}
			_ = repo.UpdateLastSynced(context.Background(), s.Source, s.SeriesID)
			log.Printf("  Saved %d observations", len(obs))
			total += len(obs)
		}
	}
	fmt.Printf("Done: %d FRED observations ingested.\n", total)
}

func runIngestECB(repo *Repository, client *ECBClient) {
	total := 0
	for i, s := range ecbSeries {
		if i > 0 {
			time.Sleep(1 * time.Second)
		}

		df, ok := ecbDataflows[s.SeriesID]
		if !ok {
			log.Printf("  No ECB dataflow config for %s (skipping)", s.SeriesID)
			continue
		}

		lastDate, _ := repo.GetLastSyncDate(context.Background(), s.Source, s.SeriesID)
		startPeriod := ""
		if lastDate != "" && len(lastDate) >= 7 {
			startPeriod = lastDate[:7]
		}

		log.Printf("[ECB %d/%d] Fetching %s (since %s)...", i+1, len(ecbSeries), s.SeriesID, startPeriod)

		obs, err := client.FetchDataflow(s.SeriesID, df.Dataflow, df.Key, startPeriod)
		if err != nil {
			log.Printf("  Error: %v (skipping)", err)
			continue
		}

		if len(obs) > 0 {
			if err := repo.SaveObservations(context.Background(), obs); err != nil {
				log.Printf("  Save error: %v (skipping)", err)
				continue
			}
			_ = repo.UpdateLastSynced(context.Background(), s.Source, s.SeriesID)
			log.Printf("  Saved %d observations", len(obs))
			total += len(obs)
		}
	}
	fmt.Printf("Done: %d ECB observations ingested.\n", total)
}

func runBackfill(repo *Repository, fredClient *FREDClient, ecbClient *ECBClient) {
	log.Println("=== Backfilling FRED series ===")
	total := 0
	for i, s := range fredSeries {
		if i > 0 {
			time.Sleep(500 * time.Millisecond)
		}
		log.Printf("[FRED %d/%d] Backfilling %s...", i+1, len(fredSeries), s.SeriesID)

		obs, err := fredClient.FetchObservations(s.SeriesID, "2000-01-01")
		if err != nil {
			log.Printf("  Error: %v (skipping)", err)
			continue
		}

		if len(obs) > 0 {
			if err := repo.SaveObservations(context.Background(), obs); err != nil {
				log.Printf("  Save error: %v (skipping)", err)
				continue
			}
			_ = repo.UpdateLastSynced(context.Background(), s.Source, s.SeriesID)
			log.Printf("  Saved %d observations", len(obs))
			total += len(obs)
		}
	}

	log.Println("=== Backfilling ECB series ===")
	for i, s := range ecbSeries {
		if i > 0 {
			time.Sleep(1 * time.Second)
		}

		df, ok := ecbDataflows[s.SeriesID]
		if !ok {
			continue
		}

		log.Printf("[ECB %d/%d] Backfilling %s...", i+1, len(ecbSeries), s.SeriesID)

		obs, err := ecbClient.FetchDataflow(s.SeriesID, df.Dataflow, df.Key, "2000-01")
		if err != nil {
			log.Printf("  Error: %v (skipping)", err)
			continue
		}

		if len(obs) > 0 {
			if err := repo.SaveObservations(context.Background(), obs); err != nil {
				log.Printf("  Save error: %v (skipping)", err)
				continue
			}
			_ = repo.UpdateLastSynced(context.Background(), s.Source, s.SeriesID)
			log.Printf("  Saved %d observations", len(obs))
			total += len(obs)
		}
	}

	fmt.Printf("Done: %d total observations backfilled.\n", total)
}
