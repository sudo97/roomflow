package main

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHealthAndMetrics(t *testing.T) {
	h := handler()
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/healthz", nil))
	if w.Code != 200 {
		t.Fatal("healthz")
	}
	w = httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/metrics", nil))
	if !strings.Contains(w.Body.String(), "service_requests_total 2") {
		t.Fatalf("metrics: %s", w.Body)
	}
}
