// Package store provides repository implementations.
package store

import (
	"sync"

	"github.com/sudo97/roomflow/internal/booking"
)

// Memory is an in-memory, tenant-aware repository.
type Memory struct {
	mu       sync.RWMutex
	rooms    map[string]booking.Room
	bookings map[string]booking.Booking
}

// NewMemory creates an empty in-memory repository.
func NewMemory() *Memory {
	return &Memory{rooms: map[string]booking.Room{}, bookings: map[string]booking.Booking{}}
}

func key(t booking.TenantID, id string) string { return string(t) + "/" + id }

// GetRoom returns a room of the tenant.
func (m *Memory) GetRoom(t booking.TenantID, id string) (booking.Room, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	r, ok := m.rooms[key(t, id)]
	if !ok {
		return booking.Room{}, booking.ErrNotFound
	}
	return r, nil
}

// ListRooms returns all rooms of the tenant.
func (m *Memory) ListRooms(t booking.TenantID) ([]booking.Room, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []booking.Room
	for _, r := range m.rooms {
		if r.TenantID == t {
			out = append(out, r)
		}
	}
	return out, nil
}

// SaveRoom stores a room.
func (m *Memory) SaveRoom(r booking.Room) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.rooms[key(r.TenantID, r.ID)] = r
	return nil
}

// ListBookings returns bookings of a room.
func (m *Memory) ListBookings(t booking.TenantID, roomID string) ([]booking.Booking, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []booking.Booking
	for _, b := range m.bookings {
		if b.TenantID == t && b.RoomID == roomID {
			out = append(out, b)
		}
	}
	return out, nil
}

// SaveBooking stores a booking.
func (m *Memory) SaveBooking(b booking.Booking) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.bookings[key(b.TenantID, b.ID)] = b
	return nil
}

// GetBooking returns a booking of the tenant.
func (m *Memory) GetBooking(t booking.TenantID, id string) (booking.Booking, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	b, ok := m.bookings[key(t, id)]
	if !ok {
		return booking.Booking{}, booking.ErrNotFound
	}
	return b, nil
}
