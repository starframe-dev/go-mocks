package net

import (
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMockHTTPClient_DefaultResponseBodyIsReadable(t *testing.T) {
	m := NewMockHTTPClient()
	req, err := http.NewRequest(http.MethodGet, "http://example.com", nil)
	require.NoError(t, err)

	responses := map[string]func() (*http.Response, error){
		"Do":   func() (*http.Response, error) { return m.Do(req) },
		"Get":  func() (*http.Response, error) { return m.Get("http://example.com") },
		"Post": func() (*http.Response, error) { return m.Post("http://example.com", "text/plain", nil) },
	}

	for name, call := range responses {
		t.Run(name, func(t *testing.T) {
			resp, err := call()
			require.NoError(t, err)

			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)
			assert.Empty(t, body)
			assert.NoError(t, resp.Body.Close())
		})
	}
}

func TestMockHTTPClient_CallbackCanReenterMock(t *testing.T) {
	m := NewMockHTTPClient()
	m.GetFunc = func(url string) (*http.Response, error) {
		// Would deadlock if the mock held its lock while running callbacks.
		_ = m.History()
		return defaultResponse(), nil
	}

	_, err := m.Get("http://example.com")

	require.NoError(t, err)
	assert.Len(t, m.History(), 1)
}

func TestInMemoryHTTPClient_InvalidURL(t *testing.T) {
	c := NewInMemoryHTTPClient()

	_, getErr := c.Get("http://bad host/%zz")
	_, postErr := c.Post("http://bad host/%zz", "text/plain", nil)

	assert.Error(t, getErr)
	assert.Error(t, postErr)
	assert.Empty(t, c.History())
}
