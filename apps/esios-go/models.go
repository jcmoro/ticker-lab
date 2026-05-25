package main

import "time"

// SeriesMeta is the metadata of a tracked ESIOS indicator. The (indicator_id,
// geo_id) pair uniquely identifies a series — the same indicator (e.g. PVPC
// 1001) can be tracked across multiple geographic zones (peninsula, Canarias).
type SeriesMeta struct {
	IndicatorID int    `json:"indicator_id"`
	GeoID       int    `json:"geo_id"`
	Name        string `json:"name"`
	ShortName   string `json:"short_name"`
	Category    string `json:"category"`
	Unit        string `json:"unit"`
	GeoName     string `json:"geo_name"`
	Frequency   string `json:"frequency"`
}

// Observation is a single hourly data point.
type Observation struct {
	IndicatorID int       `json:"indicator_id"`
	GeoID       int       `json:"geo_id"`
	Value       float64   `json:"value"`
	DatetimeUTC time.Time `json:"datetime_utc"`
}

// Indicator is a series with its latest value and previous value for change
// computation, used by the listing endpoint.
type Indicator struct {
	IndicatorID int       `json:"indicator_id"`
	GeoID       int       `json:"geo_id"`
	Name        string    `json:"name"`
	ShortName   string    `json:"short_name"`
	Category    string    `json:"category"`
	Unit        string    `json:"unit"`
	GeoName     string    `json:"geo_name"`
	Frequency   string    `json:"frequency"`
	LatestValue float64   `json:"latest_value"`
	LatestAt    time.Time `json:"latest_at"`
	PrevValue   float64   `json:"prev_value,omitempty"`
	Change      float64   `json:"change,omitempty"`
}

// HistoryPoint is one observation in a time-series chart payload.
type HistoryPoint struct {
	DatetimeUTC time.Time `json:"datetime_utc"`
	Value       float64   `json:"value"`
}

// HealthResponse is service-specific so the engine name surfaces.
type HealthResponse struct {
	Status    string `json:"status"`
	Engine    string `json:"engine"`
	Timestamp string `json:"timestamp"`
}

// Geo IDs are per-indicator: power demand/generation indicators use the
// "electric system" taxonomy (8741=España/Peninsula, 8742-8745=island
// systems), while wholesale market prices use the "country" taxonomy
// (1=Portugal, 2=Francia, 3=España). Each series in tier1Series carries
// its own GeoID.
const (
	GeoPeninsula = 8741 // electric system: España peninsular
	GeoCanarias  = 8742
	GeoBaleares  = 8743
	GeoCeuta     = 8744
	GeoMelilla   = 8745
	GeoCountryES = 3 // country-level España (used by SPOT market indicators)
)

// tier1Series — MVP set. Adding/quitting series or geos only requires
// editing this slice; no code changes elsewhere. Verified against the
// live ESIOS catalog on 2026-05-25 — names and geo taxonomies are not
// freely interchangeable across indicators.
var tier1Series = []SeriesMeta{
	{IndicatorID: 1001, GeoID: GeoPeninsula, Name: "PVPC 2.0TD", ShortName: "PVPC", Category: "pricing", Unit: "EUR/MWh", GeoName: "España", Frequency: "hourly"},
	{IndicatorID: 1293, GeoID: GeoPeninsula, Name: "Demanda real", ShortName: "Demanda", Category: "demand", Unit: "MW", GeoName: "España", Frequency: "hourly"},
	{IndicatorID: 600, GeoID: GeoCountryES, Name: "Precio mercado SPOT diario", ShortName: "SPOT", Category: "pricing", Unit: "EUR/MWh", GeoName: "España", Frequency: "hourly"},
	{IndicatorID: 10211, GeoID: GeoPeninsula, Name: "Precio horario final (suma componentes)", ShortName: "Precio final", Category: "pricing", Unit: "EUR/MWh", GeoName: "España", Frequency: "hourly"},
	{IndicatorID: 10355, GeoID: GeoPeninsula, Name: "Factor emisiones CO2 (tiempo real)", ShortName: "CO2", Category: "emissions", Unit: "tCO2/MWh", GeoName: "España", Frequency: "hourly"},
}
