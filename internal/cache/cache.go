// Package cache implements the cache-aside pattern for RoomFlow read models.
// Invalidation is version-based: every write bumps a per-scope version that is part of the key,
// so stale entries are never read and simply expire by TTL.
package cache

import (
	"context"
	"time"
)

// Cache is a key/value cache with versioned scopes.
type Cache interface {
	Get(ctx context.Context, key string) ([]byte, bool)
	Set(ctx context.Context, key string, val []byte, ttl time.Duration)
	Version(ctx context.Context, scope string) int64
	Bump(ctx context.Context, scope string)
}

// Nop is a disabled cache: every read is a miss.
type Nop struct{}

// Get always misses.
func (Nop) Get(context.Context, string) ([]byte, bool) { return nil, false }

// Set does nothing.
func (Nop) Set(context.Context, string, []byte, time.Duration) {}

// Version is always 0.
func (Nop) Version(context.Context, string) int64 { return 0 }

// Bump does nothing.
func (Nop) Bump(context.Context, string) {}
