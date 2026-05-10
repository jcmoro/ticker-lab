package httpx

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCORSMiddleware_SetsHeaders(t *testing.T) {
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	r := httptest.NewRequest(http.MethodGet, "/x", nil)
	w := httptest.NewRecorder()
	CORSMiddleware(next).ServeHTTP(w, r)

	if !called {
		t.Fatal("next handler was not called for GET")
	}
	headers := []struct{ name, want string }{
		{"Access-Control-Allow-Origin", "*"},
		{"Access-Control-Allow-Methods", "GET, OPTIONS"},
		{"Access-Control-Allow-Headers", "Content-Type"},
	}
	for _, h := range headers {
		if got := w.Header().Get(h.name); got != h.want {
			t.Errorf("%s = %q, want %q", h.name, got, h.want)
		}
	}
}

func TestCORSMiddleware_OptionsShortCircuit(t *testing.T) {
	called := false
	next := http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		called = true
	})

	r := httptest.NewRequest(http.MethodOptions, "/x", nil)
	w := httptest.NewRecorder()
	CORSMiddleware(next).ServeHTTP(w, r)

	if called {
		t.Error("next handler should NOT be called for OPTIONS")
	}
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Errorf("Origin header missing on preflight: %q", got)
	}
}
