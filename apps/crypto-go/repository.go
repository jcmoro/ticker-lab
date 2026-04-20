package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) Migrate(ctx context.Context) error {
	_, err := r.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS crypto_prices (
			id SERIAL PRIMARY KEY,
			coin_id VARCHAR(50) NOT NULL,
			symbol VARCHAR(20) NOT NULL,
			name VARCHAR(100) NOT NULL,
			price_eur NUMERIC(20, 8) NOT NULL,
			price_usd NUMERIC(20, 8) NOT NULL,
			market_cap_eur NUMERIC(24, 2) DEFAULT 0,
			change_24h NUMERIC(10, 4) DEFAULT 0,
			date DATE NOT NULL,
			created_at TIMESTAMP DEFAULT NOW(),
			UNIQUE(coin_id, date)
		);
		CREATE INDEX IF NOT EXISTS idx_crypto_prices_coin_date ON crypto_prices(coin_id, date);
	`)
	return err
}

func dedup(prices []CryptoPrice) []CryptoPrice {
	seen := make(map[string]bool)
	result := make([]CryptoPrice, 0, len(prices))
	for _, p := range prices {
		key := p.CoinID + "|" + p.Date
		if !seen[key] {
			seen[key] = true
			result = append(result, p)
		}
	}
	return result
}

func (r *Repository) Save(ctx context.Context, prices []CryptoPrice) error {
	prices = dedup(prices)
	if len(prices) == 0 {
		return nil
	}

	var b strings.Builder
	b.WriteString(`INSERT INTO crypto_prices (coin_id, symbol, name, price_eur, price_usd, market_cap_eur, change_24h, date) VALUES `)

	args := make([]any, 0, len(prices)*8)
	for i, p := range prices {
		if i > 0 {
			b.WriteString(", ")
		}
		n := i * 8
		fmt.Fprintf(&b, "($%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d)", n+1, n+2, n+3, n+4, n+5, n+6, n+7, n+8)
		args = append(args, p.CoinID, p.Symbol, p.Name, p.PriceEUR, p.PriceUSD, p.MarketCap, p.Change24h, p.Date)
	}

	b.WriteString(` ON CONFLICT (coin_id, date) DO UPDATE SET
		price_eur = EXCLUDED.price_eur,
		price_usd = CASE WHEN EXCLUDED.price_usd > 0 THEN EXCLUDED.price_usd ELSE crypto_prices.price_usd END,
		market_cap_eur = CASE WHEN EXCLUDED.market_cap_eur > 0 THEN EXCLUDED.market_cap_eur ELSE crypto_prices.market_cap_eur END,
		change_24h = CASE WHEN EXCLUDED.change_24h != 0 THEN EXCLUDED.change_24h ELSE crypto_prices.change_24h END`)

	_, err := r.pool.Exec(ctx, b.String(), args...)
	return err
}

func (r *Repository) FindLatest(ctx context.Context) ([]CryptoPrice, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT coin_id, symbol, name, price_eur::float8, price_usd::float8,
		       market_cap_eur::float8, change_24h::float8, date::text
		FROM crypto_prices
		WHERE date = (SELECT MAX(date) FROM crypto_prices)
		ORDER BY market_cap_eur DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var prices []CryptoPrice
	for rows.Next() {
		var p CryptoPrice
		if err := rows.Scan(&p.CoinID, &p.Symbol, &p.Name, &p.PriceEUR, &p.PriceUSD, &p.MarketCap, &p.Change24h, &p.Date); err != nil {
			return nil, err
		}
		prices = append(prices, p)
	}
	return prices, rows.Err()
}

func (r *Repository) FindHistory(ctx context.Context, coinID string, days int) ([]HistoryPoint, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT date::text, price_eur::float8
		FROM crypto_prices
		WHERE coin_id = $1 AND date >= CURRENT_DATE - $2::int
		ORDER BY date ASC
	`, coinID, days)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var points []HistoryPoint
	for rows.Next() {
		var p HistoryPoint
		if err := rows.Scan(&p.Date, &p.Price); err != nil {
			return nil, err
		}
		points = append(points, p)
	}
	return points, rows.Err()
}
