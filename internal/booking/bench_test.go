package booking_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/sudo97/roomflow/internal/booking"
	"github.com/sudo97/roomflow/internal/store"
)

// BenchmarkCreateBooking вимірює вартість бронювання, коли в номері вже є N бронювань
// (перевірка доступності лінійна за кількістю бронювань номера).
func BenchmarkCreateBooking(b *testing.B) {
	const layout = "2006-01-02"
	start := time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)
	for _, existing := range []int{0, 100, 1000} {
		b.Run(fmt.Sprintf("existing=%d", existing), func(b *testing.B) {
			svc := booking.NewService(store.NewMemory())
			_, _ = svc.AddRoom(booking.Room{ID: "r1", TenantID: "t", Name: "x", Capacity: 2, BasePrice: 1000})
			for i := 0; i < existing; i++ {
				in := start.AddDate(0, 0, i)
				r := booking.BookingRequest{Tenant: "t", RoomID: "r1", GuestName: "g", GuestEmail: "g@x.io", Guests: 1,
					CheckIn: in.Format(layout), CheckOut: in.AddDate(0, 0, 1).Format(layout)}
				if _, err := svc.CreateBooking(r); err != nil {
					b.Fatal(err)
				}
			}
			req := booking.BookingRequest{Tenant: "t", RoomID: "r1", GuestName: "g", GuestEmail: "g@x.io", Guests: 1, CheckIn: "2035-01-01", CheckOut: "2035-01-02"}
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, _ = svc.CreateBooking(req)
			}
		})
	}
}
