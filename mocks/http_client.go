package mocks

import (
	"net/http"
	"net/http/httptest"
)

// MockHttpClient is used in unit tests to avoid needing to send real HTTP requests.
type MockHttpClient struct {
	Handler func(w http.ResponseWriter, r *http.Request)
}

func (c *MockHttpClient) Do(request *http.Request) (*http.Response, error) {
	w := httptest.NewRecorder()
	c.Handler(w, request)
	return w.Result(), nil
}
