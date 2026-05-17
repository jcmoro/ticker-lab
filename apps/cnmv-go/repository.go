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

// migrationLockKey is a per-service advisory-lock key. Different services
// use different keys so they don't block each other during concurrent boot.
const migrationLockKey int64 = 12004

// Migrate is idempotent. An advisory lock serializes concurrent migration
// attempts (e.g. multiple service instances booting at once on Render).
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
		CREATE TABLE IF NOT EXISTS cnmv_funds (
			isin                       CHAR(12) PRIMARY KEY,
			tipo                       VARCHAR(8)   NOT NULL,
			numero_registro            BIGINT       NOT NULL,
			numero_compartimento       BIGINT       NOT NULL DEFAULT 0,
			numero_clase               BIGINT       NOT NULL DEFAULT 0,
			denominacion               VARCHAR(200) NOT NULL,
			denominacion_compartimento VARCHAR(200),
			denominacion_clase         VARCHAR(200),
			is_etf                     BOOLEAN      NOT NULL DEFAULT FALSE,
			gestora_nombre             VARCHAR(200),
			gestora_grupo              VARCHAR(200),
			depositario_nombre         VARCHAR(200),
			depositario_grupo          VARCHAR(200),
			currency                   CHAR(3)      NOT NULL DEFAULT 'EUR',
			vocacion_inversora         VARCHAR(50),
			ter                        NUMERIC(8, 4),
			comision_gestion           NUMERIC(8, 4),
			comision_deposito          NUMERIC(8, 4),
			last_seen_period           CHAR(6),
			created_at                 TIMESTAMP DEFAULT NOW(),
			updated_at                 TIMESTAMP DEFAULT NOW()
		);

		CREATE INDEX IF NOT EXISTS idx_cnmv_funds_tipo     ON cnmv_funds(tipo);
		CREATE INDEX IF NOT EXISTS idx_cnmv_funds_gestora  ON cnmv_funds(gestora_nombre);

		CREATE TABLE IF NOT EXISTS cnmv_nav_observations (
			id         BIGSERIAL PRIMARY KEY,
			isin       CHAR(12)       NOT NULL,
			date       DATE           NOT NULL,
			nav        NUMERIC(20, 6) NOT NULL,
			participes BIGINT,
			patrimonio NUMERIC(20, 2),
			created_at TIMESTAMP DEFAULT NOW(),
			UNIQUE (isin, date)
		);

		CREATE INDEX IF NOT EXISTS idx_cnmv_nav_isin_date ON cnmv_nav_observations(isin, date DESC);
		CREATE INDEX IF NOT EXISTS idx_cnmv_nav_date      ON cnmv_nav_observations(date);
	`)
	return err
}

const fundChunkSize = 500

func (r *Repository) UpsertFunds(ctx context.Context, funds []Fund, period string) error {
	for start := 0; start < len(funds); start += fundChunkSize {
		end := start + fundChunkSize
		if end > len(funds) {
			end = len(funds)
		}
		if err := r.upsertFundsChunk(ctx, funds[start:end], period); err != nil {
			return err
		}
	}
	return nil
}

func (r *Repository) upsertFundsChunk(ctx context.Context, funds []Fund, period string) error {
	if len(funds) == 0 {
		return nil
	}
	var b strings.Builder
	b.WriteString(`INSERT INTO cnmv_funds (
		isin, tipo, numero_registro, numero_compartimento, numero_clase,
		denominacion, denominacion_compartimento, denominacion_clase,
		is_etf, gestora_nombre, gestora_grupo,
		depositario_nombre, depositario_grupo, currency, last_seen_period
	) VALUES `)

	args := make([]any, 0, len(funds)*15)
	for i, f := range funds {
		if i > 0 {
			b.WriteString(", ")
		}
		n := i * 15
		fmt.Fprintf(&b,
			"($%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d)",
			n+1, n+2, n+3, n+4, n+5, n+6, n+7, n+8, n+9, n+10, n+11, n+12, n+13, n+14, n+15,
		)
		currency := f.Currency
		if currency == "" {
			currency = "EUR"
		}
		args = append(args,
			f.ISIN, f.Tipo, f.NumeroRegistro, f.NumeroCompartimento, f.NumeroClase,
			f.Denominacion, nullIfEmpty(f.DenominacionCompartimento), nullIfEmpty(f.DenominacionClase),
			f.IsETF, nullIfEmpty(f.GestoraNombre), nullIfEmpty(f.GestoraGrupo),
			nullIfEmpty(f.DepositarioNombre), nullIfEmpty(f.DepositarioGrupo), currency, period,
		)
	}
	b.WriteString(` ON CONFLICT (isin) DO UPDATE SET
		denominacion               = EXCLUDED.denominacion,
		denominacion_compartimento = EXCLUDED.denominacion_compartimento,
		denominacion_clase         = EXCLUDED.denominacion_clase,
		is_etf                     = EXCLUDED.is_etf,
		gestora_nombre             = EXCLUDED.gestora_nombre,
		gestora_grupo              = EXCLUDED.gestora_grupo,
		depositario_nombre         = EXCLUDED.depositario_nombre,
		depositario_grupo          = EXCLUDED.depositario_grupo,
		last_seen_period           = EXCLUDED.last_seen_period,
		updated_at                 = NOW()`)
	_, err := r.pool.Exec(ctx, b.String(), args...)
	return err
}

const navChunkSize = 1000

func (r *Repository) UpsertNAVs(ctx context.Context, obs []NAVObservation) error {
	for start := 0; start < len(obs); start += navChunkSize {
		end := start + navChunkSize
		if end > len(obs) {
			end = len(obs)
		}
		if err := r.upsertNAVsChunk(ctx, obs[start:end]); err != nil {
			return err
		}
	}
	return nil
}

func (r *Repository) upsertNAVsChunk(ctx context.Context, obs []NAVObservation) error {
	if len(obs) == 0 {
		return nil
	}
	var b strings.Builder
	b.WriteString(`INSERT INTO cnmv_nav_observations (isin, date, nav, participes, patrimonio) VALUES `)
	args := make([]any, 0, len(obs)*5)
	for i, o := range obs {
		if i > 0 {
			b.WriteString(", ")
		}
		n := i * 5
		fmt.Fprintf(&b, "($%d, $%d, $%d, $%d, $%d)", n+1, n+2, n+3, n+4, n+5)
		args = append(args, o.ISIN, o.Date, o.NAV, o.Participes, o.Patrimonio)
	}
	// EXCLUDED.value so re-ingesting the same month corrects late revisions.
	b.WriteString(` ON CONFLICT (isin, date) DO UPDATE SET
		nav        = EXCLUDED.nav,
		participes = EXCLUDED.participes,
		patrimonio = EXCLUDED.patrimonio`)
	_, err := r.pool.Exec(ctx, b.String(), args...)
	return err
}

// FundQuery scopes the /api/v1/funds list endpoint.
type FundQuery struct {
	Tipo    string // FI | SICAV | ...
	Gestora string // ILIKE %gestora%
	Q       string // ILIKE %denominacion%
	Limit   int    // page_size + 1 (caller asks for one extra to detect "more")
	Offset  int    // decoded from cursor
}

func (r *Repository) FindFunds(ctx context.Context, q FundQuery) ([]FundSummary, int64, error) {
	// Total count is computed in a separate query so total_size in the
	// AIP-158 response is exact, not an estimate.
	var total int64
	if err := r.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM cnmv_funds f
		WHERE (NULLIF($1, '') IS NULL OR f.tipo = $1)
		  AND (NULLIF($2, '') IS NULL OR f.gestora_nombre ILIKE '%' || $2 || '%')
		  AND (NULLIF($3, '') IS NULL OR f.denominacion ILIKE '%' || $3 || '%')
	`, q.Tipo, q.Gestora, q.Q).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := r.pool.Query(ctx, `
		SELECT f.isin, f.tipo, f.denominacion,
		       COALESCE(f.gestora_nombre, '') AS gestora_nombre,
		       f.is_etf,
		       COALESCE(o.nav::float8, 0)        AS latest_nav,
		       COALESCE(o.date::text, '')        AS latest_date,
		       COALESCE(o.patrimonio::float8, 0) AS patrimonio,
		       COALESCE(o.participes, 0)         AS participes,
		       COALESCE(prev.nav::float8, 0)     AS prev_nav
		FROM cnmv_funds f
		LEFT JOIN LATERAL (
			SELECT nav, date, patrimonio, participes
			FROM cnmv_nav_observations
			WHERE isin = f.isin
			ORDER BY date DESC LIMIT 1
		) o ON TRUE
		LEFT JOIN LATERAL (
			SELECT nav FROM cnmv_nav_observations
			WHERE isin = f.isin AND date < o.date
			ORDER BY date DESC LIMIT 1
		) prev ON TRUE
		WHERE (NULLIF($1, '') IS NULL OR f.tipo = $1)
		  AND (NULLIF($2, '') IS NULL OR f.gestora_nombre ILIKE '%' || $2 || '%')
		  AND (NULLIF($3, '') IS NULL OR f.denominacion ILIKE '%' || $3 || '%')
		ORDER BY o.patrimonio DESC NULLS LAST, f.isin
		LIMIT $4 OFFSET $5
	`, q.Tipo, q.Gestora, q.Q, q.Limit, q.Offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var out []FundSummary
	for rows.Next() {
		var fs FundSummary
		if err := rows.Scan(
			&fs.ISIN, &fs.Tipo, &fs.Denominacion, &fs.GestoraNombre, &fs.IsETF,
			&fs.LatestNAV, &fs.LatestDate, &fs.Patrimonio, &fs.Participes, &fs.PrevNAV,
		); err != nil {
			return nil, 0, err
		}
		if fs.PrevNAV > 0 {
			fs.ChangePct = ((fs.LatestNAV - fs.PrevNAV) / fs.PrevNAV) * 100
		}
		out = append(out, fs)
	}
	return out, total, rows.Err()
}

func (r *Repository) FindFundByISIN(ctx context.Context, isin string) (*Fund, error) {
	var f Fund
	var denCompart, denClase, gestNombre, gestGrupo, depNombre, depGrupo *string
	err := r.pool.QueryRow(ctx, `
		SELECT isin, tipo, numero_registro, numero_compartimento, numero_clase,
		       denominacion, denominacion_compartimento, denominacion_clase,
		       is_etf, gestora_nombre, gestora_grupo,
		       depositario_nombre, depositario_grupo, currency
		FROM cnmv_funds WHERE isin = $1
	`, isin).Scan(
		&f.ISIN, &f.Tipo, &f.NumeroRegistro, &f.NumeroCompartimento, &f.NumeroClase,
		&f.Denominacion, &denCompart, &denClase,
		&f.IsETF, &gestNombre, &gestGrupo,
		&depNombre, &depGrupo, &f.Currency,
	)
	if err != nil {
		return nil, err
	}
	if denCompart != nil {
		f.DenominacionCompartimento = *denCompart
	}
	if denClase != nil {
		f.DenominacionClase = *denClase
	}
	if gestNombre != nil {
		f.GestoraNombre = *gestNombre
	}
	if gestGrupo != nil {
		f.GestoraGrupo = *gestGrupo
	}
	if depNombre != nil {
		f.DepositarioNombre = *depNombre
	}
	if depGrupo != nil {
		f.DepositarioGrupo = *depGrupo
	}
	return &f, nil
}

// FindNAVObservations returns observations in [start, end], ascending.
// Caller passes limit+1 to detect more pages.
func (r *Repository) FindNAVObservations(ctx context.Context, isin, start, end string, limit int) ([]NavPoint, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT date::text, nav::float8
		FROM cnmv_nav_observations
		WHERE isin = $1
		  AND (NULLIF($2, '') IS NULL OR date >= $2::date)
		  AND (NULLIF($3, '') IS NULL OR date <= $3::date)
		ORDER BY date ASC
		LIMIT $4
	`, isin, start, end, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []NavPoint
	for rows.Next() {
		var p NavPoint
		if err := rows.Scan(&p.Date, &p.NAV); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// CountNAVObservations returns the total number of NAV observations
// matching the (isin, [start, end]) filter, ignoring any pagination
// cursor. Used to populate total_size on the first page response.
func (r *Repository) CountNAVObservations(ctx context.Context, isin, start, end string) (int64, error) {
	var total int64
	err := r.pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM cnmv_nav_observations
		WHERE isin = $1
		  AND (NULLIF($2, '') IS NULL OR date >= $2::date)
		  AND (NULLIF($3, '') IS NULL OR date <= $3::date)
	`, isin, start, end).Scan(&total)
	return total, err
}

func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}
