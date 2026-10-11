package net

import (
	"io"
	"net/http"
)

// HTTPClient defines the interface for HTTP operations.
// Compatible with net/http.Client from stdlib.
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
	Get(url string) (*http.Response, error)
	Post(url string, contentType string, body io.Reader) (*http.Response, error)
}

// RecordedRequest stores details of a captured HTTP request.
type RecordedRequest struct {
	Method  string
	URL     string
	Headers http.Header
	Body    []byte
}
