package main

// Fund mirrors a row in cnmv_funds: one per ISIN. CNMV's canonical key is
// the (Tipo, NumeroRegistro, NumeroCompartimento, NumeroClase) tuple, but
// ISIN is the stable public identifier and is unique per class.
type Fund struct {
	ISIN                      string  `json:"isin"`
	Tipo                      string  `json:"tipo"` // FI | FHF | SICAV | SHF
	NumeroRegistro            int64   `json:"numero_registro"`
	NumeroCompartimento       int64   `json:"numero_compartimento"`
	NumeroClase               int64   `json:"numero_clase"`
	Denominacion              string  `json:"denominacion"`
	DenominacionCompartimento string  `json:"denominacion_compartimento,omitempty"`
	DenominacionClase         string  `json:"denominacion_clase,omitempty"`
	IsETF                     bool    `json:"is_etf"`
	GestoraNombre             string  `json:"gestora_nombre,omitempty"`
	GestoraGrupo              string  `json:"gestora_grupo,omitempty"`
	DepositarioNombre         string  `json:"depositario_nombre,omitempty"`
	DepositarioGrupo          string  `json:"depositario_grupo,omitempty"`
	Currency                  string  `json:"currency"`
	VocacionInversora         string  `json:"vocacion_inversora,omitempty"` // populated from FONDTRIM (later phase)
	TER                       float64 `json:"ter,omitempty"`
	ComisionGestion           float64 `json:"comision_gestion,omitempty"`
	ComisionDeposito          float64 `json:"comision_deposito,omitempty"`
	LastSeenPeriod            string  `json:"last_seen_period,omitempty"` // YYYYMM
}

// FundSummary is the row shape returned by /api/v1/funds list endpoint —
// fund metadata joined with latest NAV/patrimonio + previous NAV for
// change computation.
type FundSummary struct {
	ISIN          string  `json:"isin"`
	Tipo          string  `json:"tipo"`
	Denominacion  string  `json:"denominacion"`
	GestoraNombre string  `json:"gestora_nombre,omitempty"`
	IsETF         bool    `json:"is_etf"`
	LatestNAV     float64 `json:"latest_nav"`
	LatestDate    string  `json:"latest_date,omitempty"`
	PrevNAV       float64 `json:"prev_nav,omitempty"`
	ChangePct     float64 `json:"change_pct,omitempty"`
	Patrimonio    float64 `json:"patrimonio,omitempty"`
	Participes    int64   `json:"participes,omitempty"`
}

// NAVObservation is a single (ISIN, date) data point.
type NAVObservation struct {
	ISIN       string
	Date       string // YYYY-MM-DD
	NAV        float64
	Participes int64
	Patrimonio float64
}

// NavPoint is one entry in a chart payload.
type NavPoint struct {
	Date string  `json:"date"`
	NAV  float64 `json:"nav"`
}

// HealthResponse — service-specific so the engine surfaces.
type HealthResponse struct {
	Status    string `json:"status"`
	Engine    string `json:"engine"`
	Timestamp string `json:"timestamp"`
}

// MonthlyZip is one entry from the CNMV listing page. URLs include an
// opaque token; do not try to predict them.
type MonthlyZip struct {
	Year  int
	Month int // 1..12
	URL   string
}
