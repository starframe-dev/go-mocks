package net

import (
	"bytes"
	"io"
	"net/http"
	"sync"
)

// MockHTTPClient implements HTTPClient for unit testing with callbacks.
type MockHTTPClient struct {
	mu      sync.RWMutex
	history []RecordedRequest

	DoFunc   func(req *http.Request) (*http.Response, error)
	GetFunc  func(url string) (*http.Response, error)
	PostFunc func(url string, contentType string, body io.Reader) (*http.Response, error)
}

// NewMockHTTPClient creates a new mock HTTP client.
func NewMockHTTPClient() *MockHTTPClient {
	return &MockHTTPClient{}
}

// Do implements HTTPClient.Do.
func (m *MockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	m.mu.Lock()
	m.recordRequest(req)
	fn := m.DoFunc
	m.mu.Unlock()

	if fn != nil {
		return fn(req)
	}
	return defaultResponse(), nil
}

// Get implements HTTPClient.Get.
func (m *MockHTTPClient) Get(url string) (*http.Response, error) {
	m.mu.Lock()
	m.history = append(m.history, RecordedRequest{
		Method: http.MethodGet,
		URL:    url,
	})
	fn := m.GetFunc
	m.mu.Unlock()

	if fn != nil {
		return fn(url)
	}
	return defaultResponse(), nil
}

// Post implements HTTPClient.Post.
func (m *MockHTTPClient) Post(url string, contentType string, body io.Reader) (*http.Response, error) {
	var bodyBytes []byte
	if body != nil {
		bodyBytes, _ = io.ReadAll(body)
	}

	m.mu.Lock()
	m.history = append(m.history, RecordedRequest{
		Method:  http.MethodPost,
		URL:     url,
		Body:    bodyBytes,
		Headers: http.Header{"Content-Type": []string{contentType}},
	})
	fn := m.PostFunc
	m.mu.Unlock()

	if fn != nil {
		return fn(url, contentType, bytes.NewReader(bodyBytes))
	}
	return defaultResponse(), nil
}

// History returns a copy of all recorded requests.
func (m *MockHTTPClient) History() []RecordedRequest {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]RecordedRequest, len(m.history))
	copy(result, m.history)
	return result
}

// Reset clears the request history.
func (m *MockHTTPClient) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.history = nil
}

// recordRequest appends req to the history. The caller must hold m.mu.
func (m *MockHTTPClient) recordRequest(req *http.Request) {
	var bodyBytes []byte
	if req.Body != nil {
		bodyBytes, _ = io.ReadAll(req.Body)
		req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
	}

	m.history = append(m.history, RecordedRequest{
		Method:  req.Method,
		URL:     req.URL.String(),
		Headers: req.Header,
		Body:    bodyBytes,
	})
}

// defaultResponse returns a 200 response with a safe empty body.
func defaultResponse() *http.Response {
	return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}
}
