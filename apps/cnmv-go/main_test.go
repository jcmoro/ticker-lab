package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ticker-lab/httpx"
)

func getTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("DATABASE_URL not set, skipping integration test")
	}
	pool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		t.Fatalf("Unable to connect to database: %v", err)
	}
	repo := NewRepository(pool)
	if err := repo.Migrate(context.Background()); err != nil {
		t.Fatalf("migrate failed: %v", err)
	}
	t.Cleanup(func() { pool.Close() })
	return pool
}

func TestHealthEndpoint(t *testing.T) {
	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	handleHealth(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}
	var resp HealthResponse
	_ = json.NewDecoder(w.Body).Decode(&resp)
	if resp.Status != "ok" || resp.Engine != "go-cnmv" {
		t.Errorf("unexpected health: %+v", resp)
	}
}

func TestParseRegistro(t *testing.T) {
	data, err := os.ReadFile("testdata/FONDREGISTRO_sample.xml")
	if err != nil {
		t.Fatal(err)
	}
	funds, period, err := ParseRegistro(data)
	if err != nil {
		t.Fatalf("ParseRegistro: %v", err)
	}
	if period != "202511" {
		t.Errorf("period = %q, want 202511", period)
	}
	// Fixture has 3 classes: FONMARCH FI x2 + TEST SICAV x1.
	if len(funds) != 3 {
		t.Fatalf("len(funds) = %d, want 3", len(funds))
	}

	byISIN := map[string]Fund{}
	for _, f := range funds {
		byISIN[f.ISIN] = f
	}
	a := byISIN["ES0138841038"]
	if a.Tipo != "FI" || a.NumeroRegistro != 9 {
		t.Errorf("FONMARCH class A: %+v", a)
	}
	if a.GestoraNombre != "MARCH ASSET MANAGEMENT, S.G.I.I.C., S.A.U." {
		t.Errorf("gestora not parsed: %q", a.GestoraNombre)
	}
	if a.DepositarioGrupo != "BANCA MARCH" {
		t.Errorf("dep grupo: %q", a.DepositarioGrupo)
	}
	sicav := byISIN["ES0123456789"]
	if sicav.Tipo != "SICAV" || sicav.GestoraNombre != "TEST GESTORA SGIIC, S.A." {
		t.Errorf("SICAV row wrong: %+v", sicav)
	}
}

func TestParseMens(t *testing.T) {
	data, err := os.ReadFile("testdata/FONDMENS_sample.xml")
	if err != nil {
		t.Fatal(err)
	}
	obs, err := ParseMens(data)
	if err != nil {
		t.Fatalf("ParseMens: %v", err)
	}
	// Fixture has values for days 1, 3, 4, 5, 30 (day 2 is 0, day 31 is 0
	// AND November has only 30 days so day 31 must be dropped).
	if len(obs) != 5 {
		t.Fatalf("len(obs) = %d, want 5; got %+v", len(obs), obs)
	}
	for _, o := range obs {
		if o.ISIN != "ES0138841038" {
			t.Errorf("unexpected ISIN: %q", o.ISIN)
		}
		if o.Date == "2026-11-02" {
			t.Errorf("day 2 (VL=0) should be filtered")
		}
		if o.Date == "2025-11-31" {
			t.Errorf("day 31 in November should be filtered (month has 30 days)")
		}
	}
	if obs[0].Date != "2025-11-01" || obs[0].NAV != 30.5744 {
		t.Errorf("first obs = %+v", obs[0])
	}
	if obs[4].Date != "2025-11-30" || obs[4].NAV != 30.5611 {
		t.Errorf("last obs = %+v", obs[4])
	}
}

func TestParseMens_ShortMonthDropsExtraDays(t *testing.T) {
	// February 2025 (28 days). Day 29-31 should be filtered even if VL != 0.
	xml := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<FondMens>
  <FechaDatos>202502</FechaDatos>
  <Entidad><Tipo>FI</Tipo><NumeroRegistro>1</NumeroRegistro>
    <Compartimento><NumeroCompartimento>0</NumeroCompartimento>
      <Clase><NumeroClase>1</NumeroClase><ISIN>ES0000000001</ISIN>
        <VLDiario>
          <VL_Dia1>10.0</VL_Dia1>
          <VL_Dia28>11.0</VL_Dia28>
          <VL_Dia29>99.0</VL_Dia29>
          <VL_Dia30>99.0</VL_Dia30>
          <VL_Dia31>99.0</VL_Dia31>
        </VLDiario>
      </Clase>
    </Compartimento>
  </Entidad>
</FondMens>`)
	obs, err := ParseMens(xml)
	if err != nil {
		t.Fatal(err)
	}
	if len(obs) != 2 {
		t.Fatalf("len(obs) = %d, want 2 (days 29-31 must be dropped); got %+v", len(obs), obs)
	}
	if obs[0].Date != "2025-02-01" || obs[1].Date != "2025-02-28" {
		t.Errorf("dates wrong: %+v %+v", obs[0], obs[1])
	}
}

func TestListMonthlyZips_HTMLScraping(t *testing.T) {
	// Realistic snippet — same id/href/title shape as the live page.
	html := `<html><body>
<table>
<tr><td>Noviembre</td><td><a id="ctl00_ContentPrincipal_grdDescargas_ctl04_lnkZip" href="https://www.cnmv.es/webservices/verdocumento/ver?e=AAA111" title="Noviembre" target="_blank">ZIP</a></td></tr>
<tr><td>Octubre</td><td><a id="ctl00_ContentPrincipal_grdDescargas_ctl05_lnkZip" href="https://www.cnmv.es/webservices/verdocumento/ver?e=BBB222" title="Octubre" target="_blank">ZIP</a></td></tr>
<tr><td>Septiembre</td><td><a id="ctl00_ContentPrincipal_grdDescargas_ctl06_lnkZip" href="https://www.cnmv.es/webservices/verdocumento/ver?e=CCC333" title="Septiembre" target="_blank">ZIP</a></td></tr>
</table>
</body></html>`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("ano") != "2025" {
			t.Errorf("missing ano=2025: %s", r.URL.RawQuery)
		}
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(html))
	}))
	t.Cleanup(server.Close)

	client := NewCNMVClient()
	client.listURL = server.URL

	zips, err := client.ListMonthlyZips(2025)
	if err != nil {
		t.Fatalf("ListMonthlyZips: %v", err)
	}
	if len(zips) != 3 {
		t.Fatalf("len(zips) = %d, want 3", len(zips))
	}

	byMonth := map[int]string{}
	for _, z := range zips {
		byMonth[z.Month] = z.URL
	}
	if !contains(byMonth[11], "AAA111") {
		t.Errorf("November URL wrong: %q", byMonth[11])
	}
	if !contains(byMonth[10], "BBB222") {
		t.Errorf("October URL wrong: %q", byMonth[10])
	}
	if !contains(byMonth[9], "CCC333") {
		t.Errorf("September URL wrong: %q", byMonth[9])
	}
}

func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && (len(s) >= len(substr)) && (indexOf(s, substr) >= 0)
}

func indexOf(s, substr string) int {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

func TestUpsertAndFind(t *testing.T) {
	pool := getTestPool(t)
	repo := NewRepository(pool)

	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), "DELETE FROM cnmv_nav_observations WHERE isin LIKE 'TS%'")
		_, _ = pool.Exec(context.Background(), "DELETE FROM cnmv_funds WHERE isin LIKE 'TS%'")
	})

	funds := []Fund{
		{ISIN: "TS0000000001", Tipo: "FI", NumeroRegistro: 1, Denominacion: "TEST FUND ONE, FI", GestoraNombre: "TEST GESTORA"},
		{ISIN: "TS0000000002", Tipo: "SICAV", NumeroRegistro: 2, Denominacion: "TEST SICAV TWO", GestoraNombre: "TEST GESTORA"},
	}
	if err := repo.UpsertFunds(context.Background(), funds, "209912"); err != nil {
		t.Fatalf("upsert funds: %v", err)
	}

	obs := []NAVObservation{
		{ISIN: "TS0000000001", Date: "2099-12-01", NAV: 10.0, Participes: 100, Patrimonio: 1000.0},
		{ISIN: "TS0000000001", Date: "2099-12-02", NAV: 11.0, Participes: 100, Patrimonio: 1100.0},
		{ISIN: "TS0000000002", Date: "2099-12-01", NAV: 50.0, Participes: 50, Patrimonio: 2500.0},
	}
	if err := repo.UpsertNAVs(context.Background(), obs); err != nil {
		t.Fatalf("upsert navs: %v", err)
	}

	// Idempotent / upsert: re-running with different NAV must update.
	if err := repo.UpsertNAVs(context.Background(), []NAVObservation{
		{ISIN: "TS0000000001", Date: "2099-12-02", NAV: 12.5, Participes: 100, Patrimonio: 1250.0},
	}); err != nil {
		t.Fatalf("upsert navs (update): %v", err)
	}

	result, total, err := repo.FindFunds(context.Background(), FundQuery{
		Gestora: "TEST GESTORA",
		Limit:   10,
	})
	if err != nil {
		t.Fatalf("find: %v", err)
	}
	if total < 2 {
		t.Errorf("total < 2: %d", total)
	}
	if len(result) < 2 {
		t.Fatalf("found < 2 funds: %d", len(result))
	}

	byISIN := map[string]FundSummary{}
	for _, r := range result {
		byISIN[r.ISIN] = r
	}
	fund1 := byISIN["TS0000000001"]
	if fund1.LatestNAV != 12.5 {
		t.Errorf("LatestNAV = %v, want 12.5 (upserted)", fund1.LatestNAV)
	}
	if fund1.PrevNAV != 10.0 {
		t.Errorf("PrevNAV = %v, want 10.0", fund1.PrevNAV)
	}
	if fund1.ChangePct == 0 {
		t.Errorf("ChangePct should be ~25%%, got %v", fund1.ChangePct)
	}
}

func TestFundDetailEndpoint_InvalidISIN(t *testing.T) {
	pool := getTestPool(t)
	repo := NewRepository(pool)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/funds/{isin}", handleFundDetail(repo))

	req := httptest.NewRequest("GET", "/api/v1/funds/NOT-AN-ISIN", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
	var pd httpx.ProblemDetails
	_ = json.NewDecoder(w.Body).Decode(&pd)
	if pd.Code != "INVALID_ARGUMENT" {
		t.Errorf("code = %q", pd.Code)
	}
}

func TestFundsEndpoint_PaginationCursor(t *testing.T) {
	pool := getTestPool(t)
	repo := NewRepository(pool)

	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), "DELETE FROM cnmv_funds WHERE isin LIKE 'TP%'")
	})

	// Seed 3 funds matching our query.
	funds := []Fund{
		{ISIN: "TP0000000001", Tipo: "FI", NumeroRegistro: 1, Denominacion: "PAGINATE A", GestoraNombre: "PAG GESTORA"},
		{ISIN: "TP0000000002", Tipo: "FI", NumeroRegistro: 2, Denominacion: "PAGINATE B", GestoraNombre: "PAG GESTORA"},
		{ISIN: "TP0000000003", Tipo: "FI", NumeroRegistro: 3, Denominacion: "PAGINATE C", GestoraNombre: "PAG GESTORA"},
	}
	_ = repo.UpsertFunds(context.Background(), funds, "209912")

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/funds", handleFunds(repo))

	// Page 1
	req := httptest.NewRequest("GET", "/api/v1/funds?gestora=PAG%20GESTORA&page_size=2", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	var page1 struct {
		Funds         []FundSummary `json:"funds"`
		NextPageToken string        `json:"next_page_token"`
		TotalSize     int64         `json:"total_size"`
	}
	if err := json.NewDecoder(w.Body).Decode(&page1); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(page1.Funds) != 2 {
		t.Fatalf("page1 funds = %d, want 2", len(page1.Funds))
	}
	if page1.NextPageToken == "" {
		t.Fatal("expected next_page_token")
	}
	if page1.TotalSize != 3 {
		t.Errorf("total_size = %d, want 3", page1.TotalSize)
	}

	// Page 2
	req = httptest.NewRequest("GET", "/api/v1/funds?gestora=PAG%20GESTORA&page_size=2&page_token="+page1.NextPageToken, nil)
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	var page2 struct {
		Funds         []FundSummary `json:"funds"`
		NextPageToken string        `json:"next_page_token"`
	}
	if err := json.NewDecoder(w.Body).Decode(&page2); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(page2.Funds) != 1 {
		t.Errorf("page2 funds = %d, want 1", len(page2.Funds))
	}
	if page2.NextPageToken != "" {
		t.Errorf("page2 next_page_token should be empty (terminal), got %q", page2.NextPageToken)
	}
}
