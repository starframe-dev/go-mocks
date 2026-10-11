package net

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// Phase 1: MockHTTPClient unit tests
// ============================================================================

func TestMockHTTPClient_Do(t *testing.T) {
	m := NewMockHTTPClient()
	m.DoFunc = func(req *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("ok"))}, nil
	}

	req, _ := http.NewRequest("GET", "http://example.com", nil)
	resp, err := m.Do(req)

	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
}

func TestMockHTTPClient_DoFuncError(t *testing.T) {
	m := NewMockHTTPClient()
	m.DoFunc = func(req *http.Request) (*http.Response, error) {
		return nil, assert.AnError
	}

	req, _ := http.NewRequest("GET", "http://example.com", nil)
	_, err := m.Do(req)

	assert.ErrorIs(t, err, assert.AnError)
}

func TestMockHTTPClient_Get(t *testing.T) {
	m := NewMockHTTPClient()
	m.GetFunc = func(url string) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("get-response"))}, nil
	}

	resp, err := m.Get("http://example.com")

	require.NoError(t, err)
	body, _ := io.ReadAll(resp.Body)
	assert.Equal(t, "get-response", string(body))
}

func TestMockHTTPClient_Post(t *testing.T) {
	m := NewMockHTTPClient()
	m.PostFunc = func(url, contentType string, body io.Reader) (*http.Response, error) {
		return &http.Response{
			StatusCode: 201,
			Body:       io.NopCloser(strings.NewReader("created")),
		}, nil
	}

	resp, err := m.Post("http://example.com", "application/json", strings.NewReader(`{}`))

	require.NoError(t, err)
	assert.Equal(t, 201, resp.StatusCode)
}

func TestMockHTTPClient_History(t *testing.T) {
	m := NewMockHTTPClient()
	m.DoFunc = func(req *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(""))}, nil
	}

	req, _ := http.NewRequest("POST", "http://api.example.com/users", strings.NewReader(`{"name":"Alice"}`))
	req.Header.Set("Content-Type", "application/json")
	m.Do(req)

	history := m.History()
	require.Len(t, history, 1)
	assert.Equal(t, "POST", history[0].Method)
	assert.Equal(t, "http://api.example.com/users", history[0].URL)
	assert.Equal(t, "application/json", history[0].Headers.Get("Content-Type"))
	assert.Equal(t, `{"name":"Alice"}`, string(history[0].Body))
}

func TestMockHTTPClient_HistoryEmpty(t *testing.T) {
	m := NewMockHTTPClient()
	assert.Empty(t, m.History())
}

func TestMockHTTPClient_Reset(t *testing.T) {
	m := NewMockHTTPClient()
	m.DoFunc = func(req *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(""))}, nil
	}

	req, _ := http.NewRequest("GET", "http://example.com", nil)
	m.Do(req)
	require.Len(t, m.History(), 1)

	m.Reset()
	assert.Empty(t, m.History())
}

func TestMockHTTPClient_ConcurrentAccess(t *testing.T) {
	m := NewMockHTTPClient()
	m.DoFunc = func(req *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(""))}, nil
	}

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				req, _ := http.NewRequest("GET", "http://example.com", nil)
				m.Do(req)
				m.History()
			}
		}()
	}
	wg.Wait()
	// If we get here without panic, concurrency is safe
	assert.Len(t, m.History(), 1000)
}

// ============================================================================
// Phase 2: InMemoryHTTPClient unit tests
// ============================================================================

func TestInMemoryHTTPClient_RegisterAndDo(t *testing.T) {
	c := NewInMemoryHTTPClient()
	c.Register("GET", "http://api.example.com/users/1", 200, []byte(`{"id":1,"name":"Alice"}`), http.Header{"Content-Type": []string{"application/json"}})

	req, _ := http.NewRequest("GET", "http://api.example.com/users/1", nil)
	resp, err := c.Do(req)

	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
	body, _ := io.ReadAll(resp.Body)
	assert.JSONEq(t, `{"id":1,"name":"Alice"}`, string(body))
	assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))
}

func TestInMemoryHTTPClient_DoNotFound(t *testing.T) {
	c := NewInMemoryHTTPClient()

	req, _ := http.NewRequest("GET", "http://api.example.com/unknown", nil)
	_, err := c.Do(req)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not registered")
}

func TestInMemoryHTTPClient_Get(t *testing.T) {
	c := NewInMemoryHTTPClient()
	c.Register("GET", "http://api.example.com/users", 200, []byte(`[]`), nil)

	resp, err := c.Get("http://api.example.com/users")

	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
}

func TestInMemoryHTTPClient_Post(t *testing.T) {
	c := NewInMemoryHTTPClient()
	c.Register("POST", "http://api.example.com/users", 201, []byte(`{"id":42}`), nil)

	resp, err := c.Post("http://api.example.com/users", "application/json", strings.NewReader(`{"name":"Bob"}`))

	require.NoError(t, err)
	assert.Equal(t, 201, resp.StatusCode)
}

func TestInMemoryHTTPClient_History(t *testing.T) {
	c := NewInMemoryHTTPClient()
	c.Register("GET", "http://api.example.com/users", 200, []byte(`[]`), nil)

	c.Get("http://api.example.com/users")

	history := c.History()
	require.Len(t, history, 1)
	assert.Equal(t, "GET", history[0].Method)
	assert.Equal(t, "http://api.example.com/users", history[0].URL)
}

func TestInMemoryHTTPClient_MultipleEndpoints(t *testing.T) {
	c := NewInMemoryHTTPClient()
	c.Register("GET", "http://api.example.com/users", 200, []byte(`[{"id":1}]`), nil)
	c.Register("GET", "http://api.example.com/posts", 200, []byte(`[{"title":"hello"}]`), nil)

	resp1, _ := c.Get("http://api.example.com/users")
	body1, _ := io.ReadAll(resp1.Body)
	assert.Contains(t, string(body1), `"id":1`)

	resp2, _ := c.Get("http://api.example.com/posts")
	body2, _ := io.ReadAll(resp2.Body)
	assert.Contains(t, string(body2), `"title":"hello"`)
}

func TestInMemoryHTTPClient_ConcurrentAccess(t *testing.T) {
	c := NewInMemoryHTTPClient()
	c.Register("GET", "http://api.example.com/users", 200, []byte(`[]`), nil)

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				c.Get("http://api.example.com/users")
				c.History()
			}
		}()
	}
	wg.Wait()
	assert.Len(t, c.History(), 1000)
}

// ============================================================================
// Phase 3: Example Service integration tests
// ============================================================================

// UserService demonstrates how to use HTTPClient mock in real code.
type UserService struct {
	client HTTPClient
	baseURL string
}

func NewUserService(client HTTPClient, baseURL string) *UserService {
	return &UserService{client: client, baseURL: baseURL}
}

func (s *UserService) FetchUser(id int) (*User, error) {
	url := s.baseURL + "/users/" + string(rune('0'+id))
	resp, err := s.client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, errors.New("failed to fetch user")
	}

	var user User
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, err
	}
	return &user, nil
}

func (s *UserService) CreateUser(name string) (*User, error) {
	payload := map[string]string{"name": name}
	data, _ := json.Marshal(payload)

	resp, err := s.client.Post(s.baseURL+"/users", "application/json", bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 201 {
		return nil, errors.New("failed to create user")
	}

	var user User
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, err
	}
	return &user, nil
}

type User struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

func TestUserService_FetchUser_Success(t *testing.T) {
	mock := NewMockHTTPClient()
	mock.GetFunc = func(url string) (*http.Response, error) {
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader(`{"id":1,"name":"Alice"}`)),
		}, nil
	}

	svc := NewUserService(mock, "http://api.example.com")
	user, err := svc.FetchUser(1)

	require.NoError(t, err)
	assert.Equal(t, 1, user.ID)
	assert.Equal(t, "Alice", user.Name)
}

func TestUserService_FetchUser_Error(t *testing.T) {
	mock := NewMockHTTPClient()
	mock.GetFunc = func(url string) (*http.Response, error) {
		return nil, assert.AnError
	}

	svc := NewUserService(mock, "http://api.example.com")
	_, err := svc.FetchUser(999)

	assert.ErrorIs(t, err, assert.AnError)
}

func TestUserService_FetchUser_Non200(t *testing.T) {
	mock := NewMockHTTPClient()
	mock.GetFunc = func(url string) (*http.Response, error) {
		return &http.Response{StatusCode: 404, Body: io.NopCloser(strings.NewReader(""))}, nil
	}

	svc := NewUserService(mock, "http://api.example.com")
	_, err := svc.FetchUser(1)

	assert.EqualError(t, err, "failed to fetch user")
}

func TestUserService_CreateUser_Success(t *testing.T) {
	client := NewInMemoryHTTPClient()
	client.Register("POST", "http://api.example.com/users", 201, []byte(`{"id":42,"name":"Bob"}`), nil)

	svc := NewUserService(client, "http://api.example.com")
	user, err := svc.CreateUser("Bob")

	require.NoError(t, err)
	assert.Equal(t, 42, user.ID)
	assert.Equal(t, "Bob", user.Name)
}

func TestUserService_CreateUser_VerifyRequest(t *testing.T) {
	client := NewInMemoryHTTPClient()
	client.Register("POST", "http://api.example.com/users", 201, []byte(`{"id":42,"name":"Bob"}`), nil)

	svc := NewUserService(client, "http://api.example.com")
	svc.CreateUser("Bob")

	history := client.History()
	require.Len(t, history, 1)
	assert.Equal(t, "POST", history[0].Method)
	assert.Equal(t, "http://api.example.com/users", history[0].URL)
	assert.Contains(t, string(history[0].Body), `"name":"Bob"`)
}

func TestUserService_CreateUser_Conflict(t *testing.T) {
	client := NewInMemoryHTTPClient()
	client.Register("POST", "http://api.example.com/users", 409, []byte(`{"error":"user exists"}`), nil)

	svc := NewUserService(client, "http://api.example.com")
	_, err := svc.CreateUser("Alice")

	assert.EqualError(t, err, "failed to create user")
}
