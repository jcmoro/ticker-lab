package httpx

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWriteJSON_Success(t *testing.T) {
	w := httptest.NewRecorder()
	payload := map[string]any{"hello": "world"}
	WriteJSON(w, http.StatusCreated, payload)

	if w.Code != http.StatusCreated {
		t.Errorf("status = %d, want %d", w.Code, http.StatusCreated)
	}
	if got := w.Header().Get("Content-Type"); got != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", got)
	}
	var out map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatalf("body not valid JSON: %v", err)
	}
	if out["hello"] != "world" {
		t.Errorf("body = %v, want hello=world", out)
	}
}

func TestWriteJSON_MarshalFailureFallsBackToProblem(t *testing.T) {
	w := httptest.NewRecorder()
	// Channels cannot be marshaled to JSON.
	WriteJSON(w, http.StatusOK, make(chan int))

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", w.Code)
	}
	if got := w.Header().Get("Content-Type"); got != "application/problem+json" {
		t.Errorf("Content-Type = %q, want application/problem+json", got)
	}
	var pd ProblemDetails
	if err := json.Unmarshal(w.Body.Bytes(), &pd); err != nil {
		t.Fatalf("body not valid problem JSON: %v", err)
	}
	if pd.Code != "INTERNAL" || pd.Status != 500 {
		t.Errorf("unexpected problem fields: %+v", pd)
	}
}

func TestWriteProblem_BasicShape(t *testing.T) {
	w := httptest.NewRecorder()
	WriteProblem(w, http.StatusBadRequest, "INVALID_ARGUMENT", "page_size must be non-negative")

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
	if got := w.Header().Get("Content-Type"); got != "application/problem+json" {
		t.Errorf("Content-Type = %q, want application/problem+json", got)
	}
	var pd ProblemDetails
	if err := json.Unmarshal(w.Body.Bytes(), &pd); err != nil {
		t.Fatalf("body not valid JSON: %v", err)
	}
	if pd.Status != 400 || pd.Code != "INVALID_ARGUMENT" {
		t.Errorf("status/code = %d/%q, want 400/INVALID_ARGUMENT", pd.Status, pd.Code)
	}
	if pd.Title != "Bad Request" {
		t.Errorf("title = %q, want Bad Request", pd.Title)
	}
	if pd.Detail != "page_size must be non-negative" {
		t.Errorf("detail = %q", pd.Detail)
	}
	if pd.Type != "https://tickerlab.dev/problems/bad-request" {
		t.Errorf("type = %q", pd.Type)
	}
}

func TestWriteProblem_NotFoundSlug(t *testing.T) {
	w := httptest.NewRecorder()
	WriteProblem(w, http.StatusNotFound, "NOT_FOUND", "no such fund")
	var pd ProblemDetails
	_ = json.Unmarshal(w.Body.Bytes(), &pd)
	if pd.Type != "https://tickerlab.dev/problems/not-found" {
		t.Errorf("type = %q", pd.Type)
	}
}
