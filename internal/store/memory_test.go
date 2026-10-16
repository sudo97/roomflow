package store

import (
	"errors"
	"testing"

	"github.com/sudo97/roomflow/internal/booking"
)

func TestMemoryTenantIsolation(t *testing.T) {
	m := NewMemory()
	_ = m.SaveRoom(booking.Room{ID: "r1", TenantID: "a"})
	_ = m.SaveRoom(booking.Room{ID: "r1", TenantID: "b"})
	_ = m.SaveBooking(booking.Booking{ID: "b1", TenantID: "a", RoomID: "r1"})

	if rooms, _ := m.ListRooms("a"); len(rooms) != 1 {
		t.Fatalf("rooms of a = %d", len(rooms))
	}
	if _, err := m.GetRoom("c", "r1"); !errors.Is(err, booking.ErrNotFound) {
		t.Fatalf("err = %v", err)
	}
	if bs, _ := m.ListBookings("b", "r1"); len(bs) != 0 {
		t.Fatalf("bookings leaked to tenant b: %v", bs)
	}
	if _, err := m.GetBooking("b", "b1"); !errors.Is(err, booking.ErrNotFound) {
		t.Fatalf("err = %v", err)
	}
	if b, err := m.GetBooking("a", "b1"); err != nil || b.ID != "b1" {
		t.Fatalf("b=%v err=%v", b, err)
	}
}
