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

type ProblemDetails struct {
	Type   string `json:"type"`
	Title  string `json:"title"`
	Status int    `json:"status"`
	Detail string `json:"detail"`
	Code   string `json:"code"`
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
	{Source: "fred", SeriesID: "PAYEMS", Name: "Nonfarm Payrolls", Freq: "monthly", Unit: "thousands", Category: "employment"},
	{Source: "fred", SeriesID: "M2SL", Name: "M2 Money Supply", Freq: "monthly", Unit: "billions_usd", Category: "monetary"},
	{Source: "fred", SeriesID: "CSUSHPINSA", Name: "Case-Shiller Home Price Index", Freq: "monthly", Unit: "index", Category: "housing"},
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
