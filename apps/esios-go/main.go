package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ticker-lab/httpx"
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
	if err := repo.SeedSeries(context.Background(), tier1Series); err != nil {
		log.Fatalf("Seed series failed: %v", err)
	}

	if len(os.Args) > 1 {
		apiKey := os.Getenv("ESIOS_API_KEY")
		switch os.Args[1] {
		case "ingest":
			if apiKey == "" {
				log.Fatal("ESIOS_API_KEY environment variable is required for ingest")
			}
			runIngest(repo, NewESIOSClient(apiKey))
			return
		case "backfill":
			if apiKey == "" {
				log.Fatal("ESIOS_API_KEY environment variable is required for backfill")
			}
			runBackfill(repo, NewESIOSClient(apiKey))
			return
		}
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", handleHealth)
	mux.HandleFunc("GET /api/v1/electricity/indicators", handleIndicators(repo))
	mux.HandleFunc("GET /api/v1/electricity/indicators/{indicator_id}/geos/{geo_id}/observations", handleObservations(repo))

	handler := httpx.CORSMiddleware(mux)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8120"
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := httpx.Run(ctx, ":"+port, handler, 15*time.Second); err != nil {
		slog.Error("server failed", "service", "esios-go", "err", err)
		os.Exit(1)
	}
}

// runIngest pulls observations since the last sync (or last 7 days if
// untracked) for every Tier 1 series. Per spec, a 2s sleep between calls
// is defensive against undocumented REE rate limits.
func runIngest(repo *Repository, client *ESIOSClient) {
	ctx := context.Background()
	now := time.Now().UTC()
	total := 0
	for i, s := range tier1Series {
		if i > 0 {
			time.Sleep(2 * time.Second)
		}

		last, _ := repo.GetLastSynced(ctx, s.IndicatorID, s.GeoID)
		start := last
		if start.IsZero() {
			start = now.Add(-7 * 24 * time.Hour)
		}

		log.Printf("[ESIOS %d/%d] %s (since %s)...", i+1, len(tier1Series), s.ShortName, start.Format(time.RFC3339))
		obs, err := client.FetchIndicator(ctx, s.IndicatorID, s.GeoID, start, now, "hour")
		if err != nil {
			log.Printf("  Error: %v (skipping)", err)
			continue
		}
		if len(obs) == 0 {
			log.Printf("  No new observations")
			continue
		}
		if err := repo.SaveObservations(ctx, obs); err != nil {
			log.Printf("  Save error: %v (skipping)", err)
			continue
		}
		_ = repo.UpdateLastSynced(ctx, s.IndicatorID, s.GeoID, obs[len(obs)-1].DatetimeUTC)
		log.Printf("  Saved %d observations", len(obs))
		total += len(obs)
	}
	fmt.Printf("Done: %d ESIOS observations ingested.\n", total)
}

// runBackfill walks from 2020-01-01 to now in monthly chunks. Chunking
// avoids any single request returning thousands of points (REE has no
// documented pagination on /indicators/{id}).
func runBackfill(repo *Repository, client *ESIOSClient) {
	ctx := context.Background()
	now := time.Now().UTC()
	earliest := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	total := 0

	for i, s := range tier1Series {
		log.Printf("=== Backfilling %s (indicator %d, geo %d) ===", s.ShortName, s.IndicatorID, s.GeoID)
		cur := earliest
		for cur.Before(now) {
			next := cur.AddDate(0, 1, 0)
			if next.After(now) {
				next = now
			}
			log.Printf("  %s … %s", cur.Format("2006-01-02"), next.Format("2006-01-02"))
			obs, err := client.FetchIndicator(ctx, s.IndicatorID, s.GeoID, cur, next, "hour")
			if err != nil {
				log.Printf("    Error: %v (skipping chunk)", err)
				cur = next
				time.Sleep(2 * time.Second)
				continue
			}
			if len(obs) > 0 {
				if err := repo.SaveObservations(ctx, obs); err != nil {
					log.Printf("    Save error: %v (skipping)", err)
				} else {
					total += len(obs)
				}
			}
			cur = next
			time.Sleep(2 * time.Second)
		}
		_ = repo.UpdateLastSynced(ctx, s.IndicatorID, s.GeoID, now)
		_ = i
	}
	fmt.Printf("Done: %d total observations backfilled.\n", total)
}
