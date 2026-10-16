package main

import (
	"path/filepath"
	"testing"
)

func TestBadCodeIsReported(t *testing.T) {
	f, err := CheckDir(filepath.Join("testdata", "src", "bad"))
	if err != nil {
		t.Fatal(err)
	}
	if len(f) != 2 {
		t.Fatalf("want 2 findings, got %d: %v", len(f), f)
	}
}

func TestGoodCodeIsClean(t *testing.T) {
	f, err := CheckDir(filepath.Join("testdata", "src", "good"))
	if err != nil {
		t.Fatal(err)
	}
	if len(f) != 0 {
		t.Fatalf("false positives: %v", f)
	}
}

func TestRepositoryIsClean(t *testing.T) {
	f, err := CheckDir(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	if len(f) != 0 {
		t.Fatalf("roomflow has dynamic SQL: %v", f)
	}
}
