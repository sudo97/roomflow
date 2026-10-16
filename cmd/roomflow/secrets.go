package main

import (
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"
)

// adminTokenFromEnvOrVault returns the admin API token. In production it is read from
// HashiCorp Vault (KV v2, path secret/data/roomflow, key admin_token) using VAULT_ADDR and
// VAULT_TOKEN; ADMIN_TOKEN is a fallback for local development. Empty token disables admin endpoints.
func adminTokenFromEnvOrVault() (string, error) {
	addr := os.Getenv("VAULT_ADDR")
	if addr == "" {
		return os.Getenv("ADMIN_TOKEN"), nil
	}
	req, err := http.NewRequest(http.MethodGet, addr+"/v1/secret/data/roomflow", nil) //nolint:gosec // VAULT_ADDR is trusted operator configuration
	if err != nil {
		return "", err
	}
	req.Header.Set("X-Vault-Token", os.Getenv("VAULT_TOKEN"))
	resp, err := (&http.Client{Timeout: 5 * time.Second}).Do(req)
	if err != nil {
		return "", fmt.Errorf("vault: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("vault: unexpected status %d", resp.StatusCode)
	}
	var body struct {
		Data struct {
			Data map[string]string `json:"data"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return "", fmt.Errorf("vault: %w", err)
	}
	token := body.Data.Data["admin_token"]
	if token == "" {
		return "", errors.New("vault: admin_token is missing")
	}
	return token, nil
}

// requireToken protects a handler with a static bearer token compared in constant time.
func requireToken(token string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got := r.Header.Get("Authorization")
		if token == "" || subtle.ConstantTimeCompare([]byte(got), []byte("Bearer "+token)) != 1 {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}
