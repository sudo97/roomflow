package booking_test

import (
	"testing"

	"github.com/sudo97/roomflow/internal/booking"
	"github.com/sudo97/roomflow/internal/store"
)

func TestCreateBookingHappyPath(t *testing.T) {
	svc := booking.NewService(store.NewMemory())
	if _, err := svc.AddRoom("hotel-a", "r1", "Double", 2, 100000); err != nil {
		t.Fatal(err)
	}
	b, err := svc.CreateBooking("hotel-a", "r1", "Ivan", "ivan@example.com", 2, "2026-03-02", "2026-03-04", "")
	if err != nil {
		t.Fatal(err)
	}
	if b.Total != 200000 {
		t.Fatalf("total = %d, want 200000", b.Total)
	}
}
