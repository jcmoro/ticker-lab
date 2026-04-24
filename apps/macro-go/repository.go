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
		CREATE TABLE IF NOT EXISTS macro_series (
			source       VARCHAR(10) NOT NULL,
			series_id    VARCHAR(50) NOT NULL,
			name         VARCHAR(200) NOT NULL,
			frequency    VARCHAR(10) NOT NULL,
			unit         VARCHAR(50) DEFAULT '',
			category     VARCHAR(50) NOT NULL,
			last_synced  TIMESTAMP,
			PRIMARY KEY (source, series_id)
		);

		CREATE TABLE IF NOT EXISTS macro_observations (
			id           SERIAL PRIMARY KEY,
			source       VARCHAR(10) NOT NULL,
			series_id    VARCHAR(50) NOT NULL,
			value        NUMERIC(20, 6) NOT NULL,
			date         DATE NOT NULL,
			created_at   TIMESTAMP DEFAULT NOW(),
			UNIQUE(source, series_id, date)
		);

		CREATE INDEX IF NOT EXISTS idx_macro_obs_series ON macro_observations(source, series_id, date);
		CREATE INDEX IF NOT EXISTS idx_macro_obs_date ON macro_observations(date);
	`)
	return err
}

func (r *Repository) SeedSeries(ctx context.Context, series []SeriesMeta) error {
	for _, s := range series {
		_, err := r.pool.Exec(ctx, `
			INSERT INTO macro_series (source, series_id, name, frequency, unit, category)
			VALUES ($1, $2, $3, $4, $5, $6)
			ON CONFLICT (source, series_id) DO UPDATE SET
				name = EXCLUDED.name,
				frequency = EXCLUDED.frequency,
				unit = EXCLUDED.unit,
				category = EXCLUDED.category
		`, s.Source, s.SeriesID, s.Name, s.Freq, s.Unit, s.Category)
		if err != nil {
			return fmt.Errorf("seed series %s/%s: %w", s.Source, s.SeriesID, err)
		}
	}
	return nil
}

func dedupObs(obs []Observation) []Observation {
	seen := make(map[string]bool)
	result := make([]Observation, 0, len(obs))
	for _, o := range obs {
		key := o.Source + "|" + o.SeriesID + "|" + o.Date
		if !seen[key] {
			seen[key] = true
			result = append(result, o)
		}
	}
	return result
}

func (r *Repository) SaveObservations(ctx context.Context, obs []Observation) error {
	obs = dedupObs(obs)
	if len(obs) == 0 {
		return nil
	}

	var b strings.Builder
	b.WriteString(`INSERT INTO macro_observations (source, series_id, value, date) VALUES `)

	args := make([]any, 0, len(obs)*4)
	for i, o := range obs {
		if i > 0 {
			b.WriteString(", ")
		}
		n := i * 4
		fmt.Fprintf(&b, "($%d, $%d, $%d, $%d)", n+1, n+2, n+3, n+4)
		args = append(args, o.Source, o.SeriesID, o.Value, o.Date)
	}

	b.WriteString(` ON CONFLICT (source, series_id, date) DO UPDATE SET value = EXCLUDED.value`)

	_, err := r.pool.Exec(ctx, b.String(), args...)
	return err
}

func (r *Repository) UpdateLastSynced(ctx context.Context, source, seriesID string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE macro_series SET last_synced = NOW()
		WHERE source = $1 AND series_id = $2
	`, source, seriesID)
	return err
}

func (r *Repository) GetLastSyncDate(ctx context.Context, source, seriesID string) (string, error) {
	var date *string
	err := r.pool.QueryRow(ctx, `
		SELECT MAX(date::text) FROM macro_observations
		WHERE source = $1 AND series_id = $2
	`, source, seriesID).Scan(&date)
	if err != nil {
		return "", err
	}
	if date == nil {
		return "", nil
	}
	return *date, nil
}

func (r *Repository) FindIndicators(ctx context.Context, category string) ([]Indicator, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT s.source, s.series_id, s.name, s.category, s.unit, s.frequency,
		       o.value AS latest_value, o.date::text AS latest_date,
		       COALESCE(prev.value, 0) AS prev_value
		FROM macro_series s
		LEFT JOIN LATERAL (
			SELECT value, date FROM macro_observations
			WHERE source = s.source AND series_id = s.series_id
			ORDER BY date DESC LIMIT 1
		) o ON true
		LEFT JOIN LATERAL (
			SELECT value FROM macro_observations
			WHERE source = s.source AND series_id = s.series_id AND date < o.date
			ORDER BY date DESC LIMIT 1
		) prev ON true
		WHERE ($1 = '' OR s.category = $1)
		ORDER BY s.category, s.name
	`, category)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var indicators []Indicator
	for rows.Next() {
		var ind Indicator
		var latestValue, prevValue *float64
		var latestDate *string
		if err := rows.Scan(&ind.Source, &ind.SeriesID, &ind.Name, &ind.Category, &ind.Unit, &ind.Freq,
			&latestValue, &latestDate, &prevValue); err != nil {
			return nil, err
		}
		if latestValue != nil {
			ind.LatestValue = *latestValue
		}
		if latestDate != nil {
			ind.LatestDate = *latestDate
		}
		if prevValue != nil {
			ind.PrevValue = *prevValue
		}
		if ind.PrevValue != 0 {
			ind.Change = ind.LatestValue - ind.PrevValue
		}
		indicators = append(indicators, ind)
	}
	return indicators, rows.Err()
}

func (r *Repository) FindHistory(ctx context.Context, source, seriesID string, days int) ([]HistoryPoint, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT date::text, value::float8
		FROM macro_observations
		WHERE source = $1 AND series_id = $2 AND date >= CURRENT_DATE - $3::int
		ORDER BY date ASC
	`, source, seriesID, days)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var points []HistoryPoint
	for rows.Next() {
		var p HistoryPoint
		if err := rows.Scan(&p.Date, &p.Value); err != nil {
			return nil, err
		}
		points = append(points, p)
	}
	return points, rows.Err()
}

func (r *Repository) FindSeriesName(ctx context.Context, source, seriesID string) (string, error) {
	var name string
	err := r.pool.QueryRow(ctx, `
		SELECT name FROM macro_series WHERE source = $1 AND series_id = $2
	`, source, seriesID).Scan(&name)
	if err != nil {
		return seriesID, nil
	}
	return name, nil
}
