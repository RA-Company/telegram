package telegram

import (
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

const mockToken = "123:SECRET"

// mockRequest is a request received by mockAPI.
type mockRequest struct {
	Method string
	Path   string
	Query  string
	Body   string
}

// mockAPI is a fake Bot API server that records requests and replies with a fixed response.
type mockAPI struct {
	*httptest.Server

	mu       sync.Mutex
	requests []mockRequest
	status   int
	response string
}

// newMockAPI starts a fake Bot API server replying with {"ok":true,"result":true}.
// It returns the server and a Telegram client pointed at it.
func newMockAPI(t *testing.T) (*mockAPI, *Telegram) {
	t.Helper()
	m := &mockAPI{status: http.StatusOK, response: `{"ok":true,"result":true}`}
	m.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		m.mu.Lock()
		m.requests = append(m.requests, mockRequest{Method: r.Method, Path: r.URL.Path, Query: r.URL.RawQuery, Body: string(body)})
		status, response := m.status, m.response
		m.mu.Unlock()
		w.WriteHeader(status)
		_, _ = w.Write([]byte(response))
	}))
	t.Cleanup(m.Close)
	return m, &Telegram{URL: m.URL, Token: mockToken}
}

// reply sets the response for subsequent requests.
func (m *mockAPI) reply(status int, response string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.status, m.response = status, response
}

// last returns the last received request.
func (m *mockAPI) last(t *testing.T) mockRequest {
	t.Helper()
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.requests) == 0 {
		t.Fatal("no requests received")
	}
	return m.requests[len(m.requests)-1]
}

// count returns the number of received requests.
func (m *mockAPI) count() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.requests)
}
