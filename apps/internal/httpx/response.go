package httpx

import (
	"encoding/json"
	"net/http"
)

// ProblemDetails is the RFC 9457 / AIP-193 error envelope used across all
// Ticker Lab services. Code values use SCREAMING_SNAKE_CASE per the
// api-design-standards.md mapping (INVALID_ARGUMENT, NOT_FOUND, etc.).
type ProblemDetails struct {
	Type   string `json:"type"`
	Title  string `json:"title"`
	Status int    `json:"status"`
	Detail string `json:"detail"`
	Code   string `json:"code"`
}

// problemTypeBaseURI is used to build the type URI per CLAUDE.md docs
// standards. Services may override by constructing ProblemDetails directly.
const problemTypeBaseURI = "https://tickerlab.dev/problems/"

// WriteJSON serializes v to the response with the given status. Falls back
// to a 500 ProblemDetails if marshaling fails.
func WriteJSON(w http.ResponseWriter, status int, v any) {
	body, err := json.Marshal(v)
	if err != nil {
		w.Header().Set("Content-Type", "application/problem+json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"type":"` + problemTypeBaseURI + `internal-server-error","title":"Internal Server Error","status":500,"detail":"response serialization failed","code":"INTERNAL"}`))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(body)
}

// WriteProblem emits an application/problem+json response with the given
// HTTP status, AIP-193 code, and human-readable title/detail.
//
// Title is omitted from the argument list when it can be derived from the
// status code via http.StatusText. Pass code in SCREAMING_SNAKE_CASE.
func WriteProblem(w http.ResponseWriter, status int, code, detail string) {
	title := http.StatusText(status)
	body, err := json.Marshal(ProblemDetails{
		Type:   problemTypeBaseURI + slugify(title),
		Title:  title,
		Status: status,
		Detail: detail,
		Code:   code,
	})
	if err != nil {
		w.Header().Set("Content-Type", "application/problem+json")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	_, _ = w.Write(body)
}

// slugify lowercases and replaces spaces with hyphens; suitable for the
// short URI fragment after problemTypeBaseURI.
func slugify(s string) string {
	out := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c == ' ':
			out = append(out, '-')
		case c >= 'A' && c <= 'Z':
			out = append(out, c+('a'-'A'))
		default:
			out = append(out, c)
		}
	}
	return string(out)
}
