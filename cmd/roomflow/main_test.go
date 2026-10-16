package main

import "testing"

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
