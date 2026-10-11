package net

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
)

// registeredResponse stores a pre-configured response for a given method + URL.
type registeredResponse struct {
	statusCode int
	body       []byte
	headers    http.Header
}

// InMemoryHTTPClient implements HTTPClient with pre-registered responses.
type InMemoryHTTPClient struct {
	mu        sync.RWMutex
	responses map[string]registeredResponse
	history   []RecordedRequest
}

// NewInMemoryHTTPClient creates a new in-memory HTTP client.
func NewInMemoryHTTPClient() *InMemoryHTTPClient {
	return &InMemoryHTTPClient{
		responses: make(map[string]registeredResponse),
	}
}

// Register associates a response with a method + URL pattern.
func (c *InMemoryHTTPClient) Register(method, url string, statusCode int, body []byte, headers http.Header) {
	c.mu.Lock()
	defer c.mu.Unlock()
	key := method + " " + url
	c.responses[key] = registeredResponse{
		statusCode: statusCode,
		body:       body,
		headers:    headers,
	}
}

// Do implements HTTPClient.Do.
func (c *InMemoryHTTPClient) Do(req *http.Request) (*http.Response, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	var bodyBytes []byte
	if req.Body != nil {
		bodyBytes, _ = io.ReadAll(req.Body)
	}

	c.history = append(c.history, RecordedRequest{
		Method:  req.Method,
		URL:     req.URL.String(),
		Headers: req.Header,
		Body:    bodyBytes,
	})

	key := req.Method + " " + req.URL.String()
	resp, ok := c.responses[key]
	if !ok {
		return nil, fmt.Errorf("request not registered: %s", key)
	}

	headers := make(http.Header)
	for k, v := range resp.headers {
		headers[k] = v
	}

	return &http.Response{
		StatusCode: resp.statusCode,
		Body:       io.NopCloser(bytes.NewReader(resp.body)),
		Header:     headers,
	}, nil
}

// Get implements HTTPClient.Get.
func (c *InMemoryHTTPClient) Get(urlStr string) (*http.Response, error) {
	u, _ := url.Parse(urlStr)
	return c.Do(&http.Request{Method: "GET", URL: u})
}

// Post implements HTTPClient.Post.
func (c *InMemoryHTTPClient) Post(urlStr string, contentType string, body io.Reader) (*http.Response, error) {
	u, _ := url.Parse(urlStr)
	var bodyBytes []byte
	if body != nil {
		bodyBytes, _ = io.ReadAll(body)
	}
	return c.Do(&http.Request{
		Method: "POST",
		URL:    u,
		Header: http.Header{"Content-Type": []string{contentType}},
		Body:   io.NopCloser(bytes.NewReader(bodyBytes)),
	})
}

// History returns a copy of all recorded requests.
func (c *InMemoryHTTPClient) History() []RecordedRequest {
	c.mu.RLock()
	defer c.mu.RUnlock()
	result := make([]RecordedRequest, len(c.history))
	copy(result, c.history)
	return result
}
