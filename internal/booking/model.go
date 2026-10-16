// Package booking contains the core domain of RoomFlow: tenants (small hotels),
// rooms and bookings.
package booking

import (
	"errors"
	"time"
)

// TenantID identifies a hotel (tenant) in the multi-tenant platform.
type TenantID string

// Room is a bookable room that belongs to a single tenant.
type Room struct {
	ID        string
	TenantID  TenantID
	Name      string
	Capacity  int
	BasePrice int64 // price per night in minor units (kopecks/cents)
}

// Booking is a reservation of a room for a date range [CheckIn, CheckOut).
type Booking struct {
	ID         string
	TenantID   TenantID
	RoomID     string
	GuestName  string
	GuestEmail string
	Guests     int
	CheckIn    time.Time
	CheckOut   time.Time
	Total      int64
	Status     string
}

// Domain errors.
var (
	ErrNotFound     = errors.New("not found")
	ErrUnavailable  = errors.New("room is not available for the requested dates")
	ErrInvalidInput = errors.New("invalid input")
	ErrWrongTenant  = errors.New("resource belongs to another tenant")
)
