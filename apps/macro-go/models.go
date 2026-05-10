package main

// SeriesMeta defines a tracked macro indicator series.
type SeriesMeta struct {
	Source   string `json:"source"`
	SeriesID string `json:"series_id"`
	Name     string `json:"name"`
	Freq     string `json:"frequency"`
	Unit     string `json:"unit"`
	Category string `json:"category"`
}

// Observation is a single data point from FRED or ECB.
type Observation struct {
	Source   string  `json:"source"`
	SeriesID string  `json:"series_id"`
	Value    float64 `json:"value"`
	Date     string  `json:"date"`
}

// Indicator is a series with its latest and previous values (for the listing endpoint).
type Indicator struct {
	Source      string  `json:"source"`
	SeriesID    string  `json:"series_id"`
	Name        string  `json:"name"`
	Category    string  `json:"category"`
	Unit        string  `json:"unit"`
	Freq        string  `json:"frequency"`
	LatestValue float64 `json:"latest_value"`
	LatestDate  string  `json:"latest_date"`
	PrevValue   float64 `json:"prev_value,omitempty"`
	Change      float64 `json:"change,omitempty"`
}

// HistoryPoint is a single point in a time series chart.
type HistoryPoint struct {
	Date  string  `json:"date"`
	Value float64 `json:"value"`
}

type HealthResponse struct {
	Status    string `json:"status"`
	Engine    string `json:"engine"`
	Timestamp string `json:"timestamp"`
}

// FRED series
var fredSeries = []SeriesMeta{
	// Tier 1
	{Source: "fred", SeriesID: "CPIAUCSL", Name: "CPI (All Urban Consumers)", Freq: "monthly", Unit: "index", Category: "inflation"},
	{Source: "fred", SeriesID: "UNRATE", Name: "Unemployment Rate", Freq: "monthly", Unit: "percent", Category: "employment"},
	{Source: "fred", SeriesID: "FEDFUNDS", Name: "Federal Funds Rate", Freq: "monthly", Unit: "percent", Category: "interest_rates"},
	{Source: "fred", SeriesID: "DGS10", Name: "10-Year Treasury Yield", Freq: "daily", Unit: "percent", Category: "interest_rates"},
	{Source: "fred", SeriesID: "GDPC1", Name: "Real GDP", Freq: "quarterly", Unit: "billions_usd", Category: "gdp"},
	// Tier 2
	{Source: "fred", SeriesID: "DGS2", Name: "2-Year Treasury Yield", Freq: "daily", Unit: "percent", Category: "interest_rates"},
	{Source: "fred", SeriesID: "T10Y2Y", Name: "10Y-2Y Treasury Spread", Freq: "daily", Unit: "percent", Category: "interest_rates"},
	{Source: "fred", SeriesID: "PCEPI", Name: "PCE Price Index", Freq: "monthly", Unit: "index", Category: "inflation"},
	{Source: "fred", SeriesID: "CPILFESL", Name: "Core CPI (ex Food & Energy)", Freq: "monthly", Unit: "index", Category: "inflation"},
	{Source: "fred", SeriesID: "PAYEMS", Name: "Nonfarm Payrolls", Freq: "monthly", Unit: "thousands", Category: "employment"},
	{Source: "fred", SeriesID: "JTSJOL", Name: "Job Openings (JOLTS)", Freq: "monthly", Unit: "thousands", Category: "employment"},
	{Source: "fred", SeriesID: "M2SL", Name: "M2 Money Supply", Freq: "monthly", Unit: "billions_usd", Category: "monetary"},
	// Tier 3 — Markets & Risk
	{Source: "fred", SeriesID: "VIXCLS", Name: "VIX Volatility Index", Freq: "daily", Unit: "index", Category: "markets"},
	{Source: "fred", SeriesID: "BAMLH0A0HYM2", Name: "High Yield Bond Spread", Freq: "daily", Unit: "percent", Category: "markets"},
	{Source: "fred", SeriesID: "DCOILWTICO", Name: "WTI Crude Oil", Freq: "daily", Unit: "usd_barrel", Category: "commodities"},
	{Source: "fred", SeriesID: "GOLDAMGBD228NLBM", Name: "Gold Price (London)", Freq: "daily", Unit: "usd_oz", Category: "commodities"},
	// Tier 3 — Consumer & Housing
	{Source: "fred", SeriesID: "CSUSHPINSA", Name: "Case-Shiller Home Price Index", Freq: "monthly", Unit: "index", Category: "housing"},
	{Source: "fred", SeriesID: "MORTGAGE30US", Name: "30-Year Mortgage Rate", Freq: "weekly", Unit: "percent", Category: "housing"},
	{Source: "fred", SeriesID: "UMCSENT", Name: "Consumer Sentiment (UMich)", Freq: "monthly", Unit: "index", Category: "consumer"},
	{Source: "fred", SeriesID: "RSAFS", Name: "Retail Sales", Freq: "monthly", Unit: "millions_usd", Category: "consumer"},
	{Source: "fred", SeriesID: "INDPRO", Name: "Industrial Production Index", Freq: "monthly", Unit: "index", Category: "production"},
}

// ECB series
var ecbSeries = []SeriesMeta{
	// Tier 1
	{Source: "ecb", SeriesID: "ICP", Name: "HICP (Eurozone Inflation)", Freq: "monthly", Unit: "percent", Category: "inflation"},
	{Source: "ecb", SeriesID: "FM_MRR", Name: "ECB Main Refinancing Rate", Freq: "monthly", Unit: "percent", Category: "interest_rates"},
	// Tier 2
	{Source: "ecb", SeriesID: "EST", Name: "Euro Short-Term Rate (ESTR)", Freq: "daily", Unit: "percent", Category: "interest_rates"},
}

// ECB dataflow configuration: maps series_id to dataflow + key for the ECB API.
var ecbDataflows = map[string]struct {
	Dataflow string
	Key      string
}{
	"ICP":    {Dataflow: "ICP", Key: "M.U2.N.000000.4.ANR"},
	"FM_MRR": {Dataflow: "FM", Key: "B.U2.EUR.4F.KR.MRR_FR.LEV"},
	"EST":    {Dataflow: "EST", Key: "B.EU000A2X2A25.WT"},
}

// BdE series — Spain-specific rates not present in ECB SDW (IRPH, MIBOR,
// NEDR/TEDR for Spanish banks, plus mortgage-reference Euribor).
// See docs/bde-integration.md.
var bdeSeries = []SeriesMeta{
	// Tier 1 — Mortgage reference rates (monthly)
	{Source: "bde", SeriesID: "D_1NBAF472", Name: "Euribor 12m (referencia hipotecaria ES)", Freq: "monthly", Unit: "percent", Category: "spanish_rates"},
	{Source: "bde", SeriesID: "D_1NBAE972", Name: "Euribor 6m (referencia hipotecaria ES)", Freq: "monthly", Unit: "percent", Category: "spanish_rates"},
	{Source: "bde", SeriesID: "D_1NBAD972", Name: "Euribor 3m (referencia hipotecaria ES)", Freq: "monthly", Unit: "percent", Category: "spanish_rates"},
	{Source: "bde", SeriesID: "D_1NBAC972", Name: "Euribor 1m (referencia hipotecaria ES)", Freq: "monthly", Unit: "percent", Category: "spanish_rates"},
	{Source: "bde", SeriesID: "D_1T9H0000", Name: "IRPH (Tipo medio préstamos hipotecarios)", Freq: "monthly", Unit: "percent", Category: "spanish_rates"},
	{Source: "bde", SeriesID: "D_1T9H0011", Name: "IRS 5 años (referencia hipotecaria ES)", Freq: "monthly", Unit: "percent", Category: "spanish_rates"},
	// Tier 2 — Euribor daily
	{Source: "bde", SeriesID: "D_DNBAF172", Name: "Euribor 12m (diario)", Freq: "daily", Unit: "percent", Category: "spanish_rates"},
	// Tier 3 — TIPI: rates applied by Spanish credit institutions (NEDR/TEDR, Spain-only)
	{Source: "bde", SeriesID: "DN_1TI2T0135", Name: "Préstamos hogares — vivienda (TEDR ES)", Freq: "monthly", Unit: "percent", Category: "spanish_rates"},
	{Source: "bde", SeriesID: "DN_1TI2T0138", Name: "Préstamos hogares — consumo (TEDR ES)", Freq: "monthly", Unit: "percent", Category: "spanish_rates"},
	{Source: "bde", SeriesID: "DN_1TI2T0144", Name: "Préstamos sociedades no financ. (TEDR ES)", Freq: "monthly", Unit: "percent", Category: "spanish_rates"},
}

// bdeDefaultRange maps frequency to the `rango` enum the BdE API expects.
// BdE rejects mismatched combinations (e.g. "12M" against a monthly series)
// with errNum 412, so the mapping is strict.
var bdeDefaultRange = map[string]string{
	"daily":     "36M",
	"monthly":   "60M",
	"quarterly": "MAX",
}
