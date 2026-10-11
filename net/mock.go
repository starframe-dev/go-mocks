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
	defer m.mu.Unlock()

	m.recordRequest(req)

	if m.DoFunc != nil {
		return m.DoFunc(req)
	}
	return &http.Response{StatusCode: 200, Body: io.NopCloser(io.NopCloser(nil))}, nil
}

// Get implements HTTPClient.Get.
func (m *MockHTTPClient) Get(url string) (*http.Response, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.history = append(m.history, RecordedRequest{
		Method: "GET",
		URL:    url,
	})

	if m.GetFunc != nil {
		return m.GetFunc(url)
	}
	return &http.Response{StatusCode: 200, Body: io.NopCloser(nil)}, nil
}

// Post implements HTTPClient.Post.
func (m *MockHTTPClient) Post(url string, contentType string, body io.Reader) (*http.Response, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var bodyBytes []byte
	if body != nil {
		bodyBytes, _ = io.ReadAll(body)
	}

	m.history = append(m.history, RecordedRequest{
		Method:  "POST",
		URL:     url,
		Body:    bodyBytes,
		Headers: http.Header{"Content-Type": []string{contentType}},
	})

	if m.PostFunc != nil {
		return m.PostFunc(url, contentType, io.NopCloser(bytes.NewReader(bodyBytes)))
	}
	return &http.Response{StatusCode: 200, Body: io.NopCloser(nil)}, nil
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
