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

// Тести додано за результатами мутаційного тестування (ЛР4): попередній тест не помічав
// інверсії фільтрів за тенантом і номером, бо перевіряв лише кількість результатів.
func TestMemoryListFiltersByTenantAndRoom(t *testing.T) {
	m := NewMemory()
	_ = m.SaveRoom(booking.Room{ID: "r1", TenantID: "a"})
	_ = m.SaveRoom(booking.Room{ID: "r2", TenantID: "a"})
	_ = m.SaveRoom(booking.Room{ID: "r1", TenantID: "b"})
	_ = m.SaveBooking(booking.Booking{ID: "b1", TenantID: "a", RoomID: "r1"})
	_ = m.SaveBooking(booking.Booking{ID: "b2", TenantID: "a", RoomID: "r2"})
	_ = m.SaveBooking(booking.Booking{ID: "b3", TenantID: "b", RoomID: "r1"})

	rooms, _ := m.ListRooms("a")
	if len(rooms) != 2 {
		t.Fatalf("tenant a must see exactly its 2 rooms, got %v", rooms)
	}
	for _, r := range rooms {
		if r.TenantID != "a" {
			t.Fatalf("room of another tenant leaked: %+v", r)
		}
	}
	bs, _ := m.ListBookings("a", "r1")
	if len(bs) != 1 || bs[0].ID != "b1" {
		t.Fatalf("want only booking b1, got %v", bs)
	}
}
