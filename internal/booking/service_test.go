package booking_test

import (
	"errors"
	"testing"

	"github.com/sudo97/roomflow/internal/booking"
	"github.com/sudo97/roomflow/internal/store"
)

const tenant = booking.TenantID("hotel-a")

func newService(t *testing.T) *booking.Service {
	t.Helper()
	svc := booking.NewService(store.NewMemory())
	if _, err := svc.AddRoom(booking.Room{ID: "r1", TenantID: tenant, Name: "Double", Capacity: 2, BasePrice: 100000}); err != nil {
		t.Fatal(err)
	}
	return svc
}

func req(in, out string) booking.BookingRequest {
	return booking.BookingRequest{Tenant: tenant, RoomID: "r1", GuestName: "Ivan", GuestEmail: "ivan@example.com", Guests: 2, CheckIn: in, CheckOut: out}
}

func TestCreateBookingHappyPath(t *testing.T) {
	b, err := newService(t).CreateBooking(req("2026-03-02", "2026-03-04"))
	if err != nil {
		t.Fatal(err)
	}
	if b.Total != 200000 || b.Status != booking.StatusConfirmed || b.Nights() != 2 {
		t.Fatalf("unexpected booking %+v", b)
	}
}

func TestCreateBookingValidation(t *testing.T) {
	cases := map[string]func(*booking.BookingRequest){
		"no tenant":     func(r *booking.BookingRequest) { r.Tenant = "" },
		"no guest name": func(r *booking.BookingRequest) { r.GuestName = "" },
		"bad email":     func(r *booking.BookingRequest) { r.GuestEmail = "not-an-email" },
		"zero guests":   func(r *booking.BookingRequest) { r.Guests = 0 },
		"bad date":      func(r *booking.BookingRequest) { r.CheckIn = "02.03.2026" },
		"out before in": func(r *booking.BookingRequest) { r.CheckOut = "2026-03-01" },
		"same day":      func(r *booking.BookingRequest) { r.CheckOut = r.CheckIn },
		"over capacity": func(r *booking.BookingRequest) { r.Guests = 3 },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			r := req("2026-03-02", "2026-03-04")
			mutate(&r)
			if _, err := newService(t).CreateBooking(r); !errors.Is(err, booking.ErrInvalidInput) {
				t.Fatalf("err = %v, want ErrInvalidInput", err)
			}
		})
	}
}

func TestCreateBookingUnknownRoom(t *testing.T) {
	r := req("2026-03-02", "2026-03-04")
	r.RoomID = "nope"
	if _, err := newService(t).CreateBooking(r); !errors.Is(err, booking.ErrNotFound) {
		t.Fatalf("err = %v", err)
	}
}

func TestRoomsAreIsolatedBetweenTenants(t *testing.T) {
	svc := newService(t)
	r := req("2026-03-02", "2026-03-04")
	r.Tenant = "hotel-b"
	if _, err := svc.CreateBooking(r); !errors.Is(err, booking.ErrNotFound) {
		t.Fatalf("tenant b must not see tenant a rooms, err = %v", err)
	}
	rooms, _ := svc.Rooms("hotel-b")
	if len(rooms) != 0 {
		t.Fatalf("rooms leaked: %v", rooms)
	}
}

func TestAvailability(t *testing.T) {
	cases := []struct {
		name    string
		in, out string
		wantErr error
	}{
		{"overlap inside", "2026-03-03", "2026-03-04", booking.ErrUnavailable},
		{"overlap start", "2026-03-01", "2026-03-03", booking.ErrUnavailable},
		{"overlap end", "2026-03-04", "2026-03-06", booking.ErrUnavailable},
		{"back to back before", "2026-03-01", "2026-03-02", nil},
		{"back to back after", "2026-03-05", "2026-03-06", nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			svc := newService(t)
			if _, err := svc.CreateBooking(req("2026-03-02", "2026-03-05")); err != nil {
				t.Fatal(err)
			}
			if _, err := svc.CreateBooking(req(c.in, c.out)); !errors.Is(err, c.wantErr) {
				t.Fatalf("err = %v, want %v", err, c.wantErr)
			}
		})
	}
}

func TestCancelFreesTheRoom(t *testing.T) {
	svc := newService(t)
	b, err := svc.CreateBooking(req("2026-03-02", "2026-03-05"))
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.CancelBooking(tenant, b.ID); err != nil {
		t.Fatal(err)
	}
	if err := svc.CancelBooking(tenant, b.ID); err != nil {
		t.Fatalf("second cancel must be a no-op: %v", err)
	}
	if _, err := svc.CreateBooking(req("2026-03-02", "2026-03-05")); err != nil {
		t.Fatalf("room must be free after cancel: %v", err)
	}
	if err := svc.CancelBooking(tenant, "missing"); !errors.Is(err, booking.ErrNotFound) {
		t.Fatalf("err = %v", err)
	}
}

func TestAddRoomValidation(t *testing.T) {
	svc := booking.NewService(store.NewMemory())
	if _, err := svc.AddRoom(booking.Room{ID: "r1", TenantID: tenant, Name: "x", Capacity: 0, BasePrice: 1}); !errors.Is(err, booking.ErrInvalidInput) {
		t.Fatalf("err = %v", err)
	}
}
