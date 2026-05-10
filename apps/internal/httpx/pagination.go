// Package httpx provides shared HTTP utilities for the Ticker Lab Go services.
//
// pagination.go implements Google AIP-158 (page_size / page_token / next_page_token).
// See docs/api-design-standards.md §1.3 for the binding rules.
package httpx

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
)

const (
	// DefaultPageSize is the page size used when the client omits page_size.
	DefaultPageSize = 100
	// MaxPageSize is the upper bound; larger requests are clamped to this value.
	MaxPageSize = 500
)

// PageParams is the parsed pagination query.
type PageParams struct {
	Size  int
	Token string
}

// PaginationError is returned for malformed pagination input.
// Status is always 400 INVALID_ARGUMENT per AIP-193.
type PaginationError struct {
	Message string
	Code    string // INVALID_ARGUMENT
	Status  int    // 400
}

func (e *PaginationError) Error() string { return e.Message }

func newPaginationError(msg string) *PaginationError {
	return &PaginationError{
		Message: msg,
		Code:    "INVALID_ARGUMENT",
		Status:  http.StatusBadRequest,
	}
}

// ParsePagination reads page_size and page_token from r.URL.Query().
// defaultSize and maxSize use the package defaults when zero.
func ParsePagination(r *http.Request, defaultSize, maxSize int) (PageParams, error) {
	if defaultSize <= 0 {
		defaultSize = DefaultPageSize
	}
	if maxSize <= 0 {
		maxSize = MaxPageSize
	}

	q := r.URL.Query()
	pageSize := defaultSize

	if raw := q.Get("page_size"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil {
			return PageParams{}, newPaginationError(
				fmt.Sprintf("page_size must be an integer, got %q", raw),
			)
		}
		if n < 0 {
			return PageParams{}, newPaginationError(
				fmt.Sprintf("page_size must be non-negative, got %d", n),
			)
		}
		switch {
		case n == 0:
			pageSize = defaultSize
		case n > maxSize:
			pageSize = maxSize
		default:
			pageSize = n
		}
	}

	return PageParams{Size: pageSize, Token: q.Get("page_token")}, nil
}

// EncodeCursor serializes a payload to an opaque, URL-safe page token.
// Clients must not parse the result; APIs may change the encoding.
func EncodeCursor(payload any) (string, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

// DecodeCursor parses an opaque token into dst.
// Empty token is treated as "no cursor" and returns nil without touching dst.
func DecodeCursor(token string, dst any) error {
	if token == "" {
		return nil
	}
	raw, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return newPaginationError("page_token is malformed")
	}
	if err := json.Unmarshal(raw, dst); err != nil {
		return newPaginationError("page_token is malformed")
	}
	return nil
}

// Page is the AIP-132/158 pagination envelope. Embed it in service response
// structs alongside the items field; the JSON tags are already correct:
//
//	type ListFundsResponse struct {
//	    Funds []Fund `json:"funds"`
//	    httpx.Page
//	}
//
// AIP-132 names the items field after the resource (e.g. "funds",
// "indicators"); each service owns that field. Page only carries
// next_page_token and the optional total_size.
type Page struct {
	NextPageToken string `json:"next_page_token"`
	TotalSize     *int64 `json:"total_size,omitempty"`
}

// NewPage constructs a Page. Pass nil for totalSize to omit it from JSON
// (AIP-158 leaves total_size optional).
func NewPage(nextPageToken string, totalSize *int64) Page {
	return Page{NextPageToken: nextPageToken, TotalSize: totalSize}
}
