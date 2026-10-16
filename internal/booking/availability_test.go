package booking_test

import (
	"errors"
	"testing"

	"github.com/sudo97/roomflow/internal/booking"
	"github.com/sudo97/roomflow/internal/store"
)

// failingList is a repository whose ListBookings always fails.
type failingList struct{ *store.Memory }

func (failingList) ListBookings(booking.TenantID, string) ([]booking.Booking, error) {
	return nil, errors.New("storage is down")
}

func TestRoomCalendar(t *testing.T) {
	svc := newService(t)
	if _, err := svc.CreateBooking(req("2026-03-03", "2026-03-05")); err != nil {
		t.Fatal(err)
	}
	nights, err := svc.Availability(tenant, "r1", "2026-03-02", "2026-03-08")
	if err != nil {
		t.Fatal(err)
	}
	want := []booking.Night{
		{Date: "2026-03-02", Price: 100000}, {Date: "2026-03-03", Price: 100000, Booked: true},
		{Date: "2026-03-04", Price: 100000, Booked: true}, {Date: "2026-03-05", Price: 100000},
		{Date: "2026-03-06", Price: 100000}, {Date: "2026-03-07", Price: 120000},
	}
	if len(nights) != len(want) {
		t.Fatalf("got %d nights", len(nights))
	}
	for i := range want {
		if nights[i] != want[i] {
			t.Fatalf("night %d: got %+v, want %+v", i, nights[i], want[i])
		}
	}
	if err := svc.CancelBooking(tenant, "bk-1"); err != nil {
		t.Fatal(err)
	}
	if nights, _ := svc.Availability(tenant, "r1", "2026-03-03", "2026-03-04"); nights[0].Booked {
		t.Fatal("cancelled booking must free the night")
	}
}

func TestRoomCalendarErrors(t *testing.T) {
	svc := newService(t)
	for _, c := range [][2]string{{"bad", "2026-03-04"}, {"2026-03-04", "bad"}, {"2026-03-04", "2026-03-04"}, {"2026-01-01", "2027-01-03"}} {
		if _, err := svc.Availability(tenant, "r1", c[0], c[1]); !errors.Is(err, booking.ErrInvalidInput) {
			t.Fatalf("%v: got %v", c, err)
		}
	}
	if _, err := svc.Availability(tenant, "nope", "2026-03-01", "2026-03-02"); !errors.Is(err, booking.ErrNotFound) {
		t.Fatalf("missing room: %v", err)
	}
	if n, err := svc.Availability(tenant, "r1", "2026-01-01", "2027-01-02"); err != nil || len(n) != 366 {
		t.Fatalf("366 nights must be allowed: %v", err)
	}
	repo := failingList{store.NewMemory()}
	_ = repo.SaveRoom(booking.Room{ID: "r1", TenantID: tenant, Name: "D", Capacity: 1, BasePrice: 1})
	if _, err := booking.NewService(repo).Availability(tenant, "r1", "2026-03-01", "2026-03-02"); err == nil {
		t.Fatal("repository error must propagate")
	}
}
