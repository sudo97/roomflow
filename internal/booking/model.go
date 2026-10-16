// Package booking contains the core domain of RoomFlow: tenants (small hotels),
// rooms and bookings.
package booking

import (
	"errors"
	"time"
)

// TenantID identifies a hotel (tenant) in the multi-tenant platform.
type TenantID string

// Booking statuses.
const (
	StatusConfirmed = "confirmed"
	StatusCancelled = "cancelled"
)

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

// Active reports whether the booking still occupies the room.
func (b Booking) Active() bool { return b.Status != StatusCancelled }

// Overlaps reports whether the booking intersects the half-open range [in, out).
func (b Booking) Overlaps(in, out time.Time) bool {
	return in.Before(b.CheckOut) && out.After(b.CheckIn)
}

// Nights returns the number of nights in the booking.
func (b Booking) Nights() int { return nightsBetween(b.CheckIn, b.CheckOut) }

func nightsBetween(in, out time.Time) int { return int(out.Sub(in).Hours() / 24) }

// Domain errors.
var (
	ErrNotFound     = errors.New("not found")
	ErrUnavailable  = errors.New("room is not available for the requested dates")
	ErrInvalidInput = errors.New("invalid input")
)
