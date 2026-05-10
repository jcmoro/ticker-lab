package httpx

import "net/http"

// CORSMiddleware sets permissive CORS headers (Access-Control-Allow-Origin: *)
// and short-circuits OPTIONS preflight with 200.
//
// Permissive CORS is intentional for the Ticker Lab dev/personal-experiment
// scope. If a caller needs tighter origin/method allowlists in the future,
// add a configurable variant rather than changing this default.
func CORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}
