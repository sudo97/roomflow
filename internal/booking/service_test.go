package booking_test

import (
	"testing"

	"github.com/sudo97/roomflow/internal/booking"
	"github.com/sudo97/roomflow/internal/store"
)

// «Слабкі» тести для демонстрації: проходять усі гілки сервісу, але майже нічого не перевіряють.
func TestServiceSmoke(t *testing.T) {
	svc := booking.NewService(store.NewMemory())
	_, _ = svc.AddRoom(booking.Room{ID: "r1", TenantID: "hotel-a", Name: "Double", Capacity: 2, BasePrice: 1000})
	_, _ = svc.AddRoom(booking.Room{ID: "r2", TenantID: "hotel-a", Name: "x", Capacity: 0, BasePrice: 1})
	req := booking.Request{Tenant: "hotel-a", RoomID: "r1", GuestName: "Ivan", GuestEmail: "i@x.io", Guests: 2, CheckIn: "2026-03-02", CheckOut: "2026-03-05"}
	b, err := svc.CreateBooking(req)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range []booking.Request{
		{Tenant: "hotel-a", RoomID: "r1", GuestName: "Olha", GuestEmail: "o@x.io", Guests: 1, CheckIn: "2026-03-03", CheckOut: "2026-03-04"},
		{Tenant: "hotel-a", RoomID: "r1", GuestName: "Olha", GuestEmail: "o@x.io", Guests: 3, CheckIn: "2026-04-03", CheckOut: "2026-04-04"},
		{Tenant: "hotel-a", RoomID: "nope", GuestName: "Olha", GuestEmail: "o@x.io", Guests: 1, CheckIn: "2026-04-03", CheckOut: "2026-04-04"},
		{Tenant: "hotel-a", RoomID: "r1", GuestName: "", GuestEmail: "bad", Guests: 1, CheckIn: "x", CheckOut: "y"},
		{Tenant: "hotel-a", RoomID: "r1", GuestName: "Olha", GuestEmail: "bad", Guests: 1, CheckIn: "2026-04-03", CheckOut: "2026-04-04"},
	} {
		_, _ = svc.CreateBooking(r) // результат не перевіряється
	}
	_ = svc.CancelBooking("hotel-a", b.ID)
	_ = svc.CancelBooking("hotel-a", b.ID)
	_ = svc.CancelBooking("hotel-a", "missing")
	_, _ = svc.Rooms("hotel-a")
}
