package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestEnvOr(t *testing.T) {
	t.Setenv("ROOMFLOW_X", "")
	if envOr("ROOMFLOW_X", "def") != "def" {
		t.Fatal("default expected")
	}
	t.Setenv("ROOMFLOW_X", "v")
	if envOr("ROOMFLOW_X", "def") != "v" {
		t.Fatal("env value expected")
	}
}

func TestProbe(t *testing.T) {
	ok := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {}))
	defer ok.Close()
	bad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(503) }))
	defer bad.Close()
	if probe(ok.URL) != 0 || probe(bad.URL) != 1 || probe("http://127.0.0.1:1") != 1 {
		t.Fatal("probe returned unexpected exit codes")
	}
}
