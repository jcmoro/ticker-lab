package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// migrationLockKey is a per-service advisory-lock key. Different services
// use different keys so they don't block each other during concurrent boot.
const migrationLockKey int64 = 12003

// Migrate is idempotent: safe to call on every startup. An advisory lock
// serializes concurrent migration attempts (e.g. multiple service instances
// booting at once on Render).
func (r *Repository) Migrate(ctx context.Context) error {
	conn, err := r.pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("acquire connection: %w", err)
	}
	defer conn.Release()

	if _, err := conn.Exec(ctx, "SELECT pg_advisory_lock($1)", migrationLockKey); err != nil {
		return fmt.Errorf("acquire advisory lock: %w", err)
	}
	defer func() {
		_, _ = conn.Exec(context.Background(), "SELECT pg_advisory_unlock($1)", migrationLockKey)
	}()

	_, err = conn.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS esios_series (
			indicator_id   INTEGER NOT NULL,
			geo_id         INTEGER NOT NULL,
			name           VARCHAR(200) NOT NULL,
			short_name     VARCHAR(50)  NOT NULL,
			category       VARCHAR(50)  NOT NULL,
			unit           VARCHAR(20)  NOT NULL,
			geo_name       VARCHAR(50)  NOT NULL,
			frequency      VARCHAR(10)  NOT NULL DEFAULT 'hourly',
			last_synced_at TIMESTAMPTZ,
			PRIMARY KEY (indicator_id, geo_id)
		);

		CREATE TABLE IF NOT EXISTS esios_observations (
			id            BIGSERIAL    PRIMARY KEY,
			indicator_id  INTEGER      NOT NULL,
			geo_id        INTEGER      NOT NULL,
			value         NUMERIC(20,6) NOT NULL,
			datetime_utc  TIMESTAMPTZ  NOT NULL,
			created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
			UNIQUE (indicator_id, geo_id, datetime_utc)
		);

		CREATE INDEX IF NOT EXISTS idx_esios_obs_series
			ON esios_observations(indicator_id, geo_id, datetime_utc DESC);
		CREATE INDEX IF NOT EXISTS idx_esios_obs_dt
			ON esios_observations(datetime_utc DESC);
	`)
	return err
}

// SeedSeries upserts the static series catalog so handlers can JOIN against
// it. Existing rows keep their last_synced_at; only metadata fields are
// updated.
func (r *Repository) SeedSeries(ctx context.Context, series []SeriesMeta) error {
	if len(series) == 0 {
		return nil
	}
	var b strings.Builder
	b.WriteString(`INSERT INTO esios_series (indicator_id, geo_id, name, short_name, category, unit, geo_name, frequency) VALUES `)
	args := make([]any, 0, len(series)*8)
	for i, s := range series {
		if i > 0 {
			b.WriteString(", ")
		}
		n := i * 8
		fmt.Fprintf(&b, "($%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d)", n+1, n+2, n+3, n+4, n+5, n+6, n+7, n+8)
		args = append(args, s.IndicatorID, s.GeoID, s.Name, s.ShortName, s.Category, s.Unit, s.GeoName, s.Frequency)
	}
	b.WriteString(` ON CONFLICT (indicator_id, geo_id) DO UPDATE SET
		name = EXCLUDED.name,
		short_name = EXCLUDED.short_name,
		category = EXCLUDED.category,
		unit = EXCLUDED.unit,
		geo_name = EXCLUDED.geo_name,
		frequency = EXCLUDED.frequency`)
	_, err := r.pool.Exec(ctx, b.String(), args...)
	return err
}

func (r *Repository) SaveObservations(ctx context.Context, obs []Observation) error {
	if len(obs) == 0 {
		return nil
	}
	var b strings.Builder
	b.WriteString(`INSERT INTO esios_observations (indicator_id, geo_id, value, datetime_utc) VALUES `)
	args := make([]any, 0, len(obs)*4)
	for i, o := range obs {
		if i > 0 {
			b.WriteString(", ")
		}
		n := i * 4
		fmt.Fprintf(&b, "($%d, $%d, $%d, $%d)", n+1, n+2, n+3, n+4)
		args = append(args, o.IndicatorID, o.GeoID, o.Value, o.DatetimeUTC)
	}
	// EXCLUDED.value: re-runs of the same window must update, not noop.
	b.WriteString(` ON CONFLICT (indicator_id, geo_id, datetime_utc) DO UPDATE SET value = EXCLUDED.value`)
	_, err := r.pool.Exec(ctx, b.String(), args...)
	return err
}

func (r *Repository) UpdateLastSynced(ctx context.Context, indicatorID, geoID int, ts time.Time) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE esios_series SET last_synced_at = $3
		WHERE indicator_id = $1 AND geo_id = $2
	`, indicatorID, geoID, ts)
	return err
}

func (r *Repository) GetLastSynced(ctx context.Context, indicatorID, geoID int) (time.Time, error) {
	var ts *time.Time
	err := r.pool.QueryRow(ctx, `
		SELECT last_synced_at FROM esios_series
		WHERE indicator_id = $1 AND geo_id = $2
	`, indicatorID, geoID).Scan(&ts)
	if err != nil {
		return time.Time{}, err
	}
	if ts == nil {
		return time.Time{}, nil
	}
	return *ts, nil
}

// FindIndicators returns the catalog with the latest value + previous value
// for change computation.
func (r *Repository) FindIndicators(ctx context.Context, category string) ([]Indicator, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT s.indicator_id, s.geo_id, s.name, s.short_name, s.category, s.unit, s.geo_name, s.frequency,
		       COALESCE(o.value, 0)        AS latest_value,
		       COALESCE(o.datetime_utc, '1970-01-01'::timestamptz) AS latest_at,
		       COALESCE(prev.value, 0)     AS prev_value
		FROM esios_series s
		LEFT JOIN LATERAL (
			SELECT value, datetime_utc FROM esios_observations
			WHERE indicator_id = s.indicator_id AND geo_id = s.geo_id
			ORDER BY datetime_utc DESC LIMIT 1
		) o ON TRUE
		LEFT JOIN LATERAL (
			SELECT value FROM esios_observations
			WHERE indicator_id = s.indicator_id AND geo_id = s.geo_id
			  AND datetime_utc < o.datetime_utc
			ORDER BY datetime_utc DESC LIMIT 1
		) prev ON TRUE
		WHERE ($1 = '' OR s.category = $1)
		ORDER BY s.category, s.name
	`, category)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Indicator
	for rows.Next() {
		var ind Indicator
		if err := rows.Scan(
			&ind.IndicatorID, &ind.GeoID, &ind.Name, &ind.ShortName,
			&ind.Category, &ind.Unit, &ind.GeoName, &ind.Frequency,
			&ind.LatestValue, &ind.LatestAt, &ind.PrevValue,
		); err != nil {
			return nil, err
		}
		if ind.PrevValue != 0 {
			ind.Change = ind.LatestValue - ind.PrevValue
		}
		out = append(out, ind)
	}
	return out, rows.Err()
}

// FindObservations returns observations within [start, end], ordered
// ascending. Caller is responsible for clamping the range and applying
// pagination on top.
func (r *Repository) FindObservations(
	ctx context.Context,
	indicatorID, geoID int,
	start, end time.Time,
	limit int,
) ([]HistoryPoint, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT datetime_utc, value::float8
		FROM esios_observations
		WHERE indicator_id = $1 AND geo_id = $2
		  AND datetime_utc BETWEEN $3 AND $4
		ORDER BY datetime_utc ASC
		LIMIT $5
	`, indicatorID, geoID, start, end, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []HistoryPoint
	for rows.Next() {
		var p HistoryPoint
		if err := rows.Scan(&p.DatetimeUTC, &p.Value); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// CountObservations returns the total number of observations matching the
// (indicator_id, geo_id, [start, end]) filter, ignoring any pagination
// cursor. Used to populate total_size on the first page response.
func (r *Repository) CountObservations(
	ctx context.Context,
	indicatorID, geoID int,
	start, end time.Time,
) (int64, error) {
	var total int64
	err := r.pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM esios_observations
		WHERE indicator_id = $1 AND geo_id = $2
		  AND datetime_utc BETWEEN $3 AND $4
	`, indicatorID, geoID, start, end).Scan(&total)
	return total, err
}
