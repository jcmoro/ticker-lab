package httpx

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestParsePagination_DefaultsWhenEmpty(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/x", nil)
	got, err := ParsePagination(r, 0, 0)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if got.Size != DefaultPageSize {
		t.Errorf("Size = %d, want %d", got.Size, DefaultPageSize)
	}
	if got.Token != "" {
		t.Errorf("Token = %q, want empty", got.Token)
	}
}

func TestParsePagination_PageSizeParsed(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/x?page_size=50", nil)
	got, err := ParsePagination(r, 0, 0)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if got.Size != 50 {
		t.Errorf("Size = %d, want 50", got.Size)
	}
}

func TestParsePagination_ClampsToMax(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/x?page_size=9999", nil)
	got, err := ParsePagination(r, 0, 0)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if got.Size != MaxPageSize {
		t.Errorf("Size = %d, want %d", got.Size, MaxPageSize)
	}
}

func TestParsePagination_HonorsCustomMax(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/x?page_size=9999", nil)
	got, err := ParsePagination(r, 0, 200)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if got.Size != 200 {
		t.Errorf("Size = %d, want 200", got.Size)
	}
}

func TestParsePagination_ZeroUsesDefault(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/x?page_size=0", nil)
	got, err := ParsePagination(r, 0, 0)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if got.Size != DefaultPageSize {
		t.Errorf("Size = %d, want %d", got.Size, DefaultPageSize)
	}
}

func TestParsePagination_NegativeIsError(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/x?page_size=-1", nil)
	_, err := ParsePagination(r, 0, 0)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var pe *PaginationError
	if !errors.As(err, &pe) {
		t.Fatalf("error is not PaginationError: %T", err)
	}
	if pe.Status != http.StatusBadRequest || pe.Code != "INVALID_ARGUMENT" {
		t.Errorf("unexpected error fields: status=%d code=%q", pe.Status, pe.Code)
	}
}

func TestParsePagination_NonNumericIsError(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/x?page_size=abc", nil)
	_, err := ParsePagination(r, 0, 0)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var pe *PaginationError
	if !errors.As(err, &pe) {
		t.Fatalf("error is not PaginationError: %T", err)
	}
}

func TestParsePagination_ExtractsToken(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/x?page_token=abc123", nil)
	got, err := ParsePagination(r, 0, 0)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if got.Token != "abc123" {
		t.Errorf("Token = %q, want abc123", got.Token)
	}
}

func TestCursor_RoundTrip(t *testing.T) {
	type cursor struct {
		LastID   string `json:"last_id"`
		LastDate string `json:"last_date"`
	}
	in := cursor{LastID: "ES0138841038", LastDate: "2026-05-08"}
	token, err := EncodeCursor(in)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	if strings.ContainsAny(token, "/+=") {
		t.Errorf("token %q contains URL-unsafe characters", token)
	}
	var out cursor
	if err := DecodeCursor(token, &out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if out != in {
		t.Errorf("round-trip mismatch: got %+v want %+v", out, in)
	}
}

func TestDecodeCursor_EmptyTokenIsNoOp(t *testing.T) {
	var out struct{ X string }
	if err := DecodeCursor("", &out); err != nil {
		t.Errorf("unexpected err: %v", err)
	}
	if out.X != "" {
		t.Errorf("dst was modified: %+v", out)
	}
}

func TestDecodeCursor_MalformedBase64(t *testing.T) {
	var out struct{}
	err := DecodeCursor("not!valid!base64", &out)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var pe *PaginationError
	if !errors.As(err, &pe) {
		t.Fatalf("error is not PaginationError: %T", err)
	}
}

func TestDecodeCursor_ValidBase64InvalidJSON(t *testing.T) {
	// "not json" base64-encoded
	corrupt := "bm90IGpzb24"
	var out struct{}
	err := DecodeCursor(corrupt, &out)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var pe *PaginationError
	if !errors.As(err, &pe) {
		t.Fatalf("error is not PaginationError: %T", err)
	}
}

func TestPage_EmbeddedInServiceResponse(t *testing.T) {
	type fund struct {
		ISIN string `json:"isin"`
	}
	type response struct {
		Funds []fund `json:"funds"`
		Page
	}

	total := int64(3112)
	r := response{
		Funds: []fund{{ISIN: "ES0138841038"}},
		Page:  NewPage("next-token", &total),
	}

	raw, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	got := string(raw)

	for _, want := range []string{
		`"funds":[{"isin":"ES0138841038"}]`,
		`"next_page_token":"next-token"`,
		`"total_size":3112`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("response missing %q\nfull: %s", want, got)
		}
	}
}

func TestPage_OmitsTotalSizeWhenNil(t *testing.T) {
	type response struct {
		Items []int `json:"items"`
		Page
	}
	r := response{Items: []int{1, 2}, Page: NewPage("", nil)}
	raw, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	got := string(raw)
	if strings.Contains(got, "total_size") {
		t.Errorf("expected total_size to be omitted, got: %s", got)
	}
	if !strings.Contains(got, `"next_page_token":""`) {
		t.Errorf("expected empty next_page_token, got: %s", got)
	}
}
