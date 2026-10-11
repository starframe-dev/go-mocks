package contracts

import (
	"io"
	"net/http"

	"github.com/starframe-dev/go-mocks/net"
)

// NetContract defines the expected interface for HTTP client implementations.
// Compatible with net/http.Client from stdlib.
type NetContract interface {
	// Do sends an HTTP request and returns an HTTP response.
	Do(req *http.Request) (*http.Response, error)

	// Get sends a GET request to the specified URL.
	Get(url string) (*http.Response, error)

	// Post sends a POST request to the specified URL.
	Post(url string, contentType string, body io.Reader) (*http.Response, error)
}

// AssertNetContract compiles at compile time that MockHTTPClient implements NetContract.
var _ NetContract = (*net.MockHTTPClient)(nil)
