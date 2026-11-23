package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// The read-only listener must not offer anything that changes state.
func TestReadOnlyRouter(t *testing.T) {
	r := readOnlyRouter()
	cases := []struct {
		method, path string
		want         int
	}{
		{"GET", "/api/clients", http.StatusOK},
		{"POST", "/api/clients", http.StatusMethodNotAllowed},
		{"POST", "/api/clients/aa:bb:cc:dd:ee:ff/disconnect", http.StatusNotFound},
		{"GET", "/ws/clients", http.StatusNotFound},
		{"GET", "/", http.StatusNotFound},
	}
	for _, c := range cases {
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, httptest.NewRequest(c.method, c.path, nil))
		if rec.Code != c.want {
			t.Errorf("%s %s: got %d, want %d", c.method, c.path, rec.Code, c.want)
		}
	}
}
