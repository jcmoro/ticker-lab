package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
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

	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "ingest":
			runIngest(repo, NewCNMVClient())
			return
		case "backfill":
			fromYear := 2020
			if len(os.Args) > 2 {
				if y, err := strconv.Atoi(os.Args[2]); err == nil {
					fromYear = y
				}
			}
			runBackfill(repo, NewCNMVClient(), fromYear)
			return
		}
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", handleHealth)
	mux.HandleFunc("GET /api/v1/funds", handleFunds(repo))
	mux.HandleFunc("GET /api/v1/funds/{isin}", handleFundDetail(repo))
	mux.HandleFunc("GET /api/v1/funds/{isin}/nav-observations", handleNavObservations(repo))

	handler := httpx.CORSMiddleware(mux)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8130"
	}
	log.Printf("CNMV Go listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, handler))
}

// runIngest downloads the current month + previous month and upserts.
// Idempotent against re-runs (ON CONFLICT updates).
func runIngest(repo *Repository, client *CNMVClient) {
	now := time.Now().UTC()
	current := monthOf(now)
	previous := monthOf(now.AddDate(0, -1, 0))

	for _, period := range []time.Time{previous, current} {
		if err := ingestPeriod(repo, client, period); err != nil {
			log.Printf("Period %04d-%02d failed: %v", period.Year(), period.Month(), err)
		}
	}
	fmt.Println("Done: CNMV ingest complete.")
}

// runBackfill iterates from fromYear-01 to now monthly.
func runBackfill(repo *Repository, client *CNMVClient, fromYear int) {
	now := time.Now().UTC()
	start := time.Date(fromYear, time.January, 1, 0, 0, 0, 0, time.UTC)
	for cur := start; !cur.After(now); cur = cur.AddDate(0, 1, 0) {
		if err := ingestPeriod(repo, client, cur); err != nil {
			log.Printf("Backfill %04d-%02d failed: %v", cur.Year(), cur.Month(), err)
		}
		time.Sleep(2 * time.Second) // be nice to CNMV
	}
	fmt.Println("Done: CNMV backfill complete.")
}

func ingestPeriod(repo *Repository, client *CNMVClient, period time.Time) error {
	year := period.Year()
	month := int(period.Month())
	periodTag := fmt.Sprintf("%04d%02d", year, month)

	log.Printf("[CNMV %04d-%02d] Listing year %d...", year, month, year)
	zips, err := client.ListMonthlyZips(year)
	if err != nil {
		return fmt.Errorf("list: %w", err)
	}

	var target *MonthlyZip
	for i := range zips {
		if zips[i].Month == month {
			target = &zips[i]
			break
		}
	}
	if target == nil {
		log.Printf("  No ZIP for %04d-%02d yet (may not be published)", year, month)
		return nil
	}

	log.Printf("  Downloading %s...", target.URL)
	registroXML, mensXML, err := client.DownloadZip(target.URL)
	if err != nil {
		return fmt.Errorf("download: %w", err)
	}

	funds, _, err := ParseRegistro(registroXML)
	if err != nil {
		return fmt.Errorf("parse registro: %w", err)
	}
	obs, err := ParseMens(mensXML)
	if err != nil {
		return fmt.Errorf("parse mens: %w", err)
	}

	log.Printf("  Parsed %d funds and %d nav observations", len(funds), len(obs))

	ctx := context.Background()
	if err := repo.UpsertFunds(ctx, funds, periodTag); err != nil {
		return fmt.Errorf("upsert funds: %w", err)
	}
	if err := repo.UpsertNAVs(ctx, obs); err != nil {
		return fmt.Errorf("upsert navs: %w", err)
	}
	log.Printf("  Saved.")
	return nil
}

func monthOf(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
}
