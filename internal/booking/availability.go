package booking

import (
	"fmt"
	"time"
)

// maxAvailabilityNights limits the calendar window of one availability request.
const maxAvailabilityNights = 366

// Night is one night of a room calendar.
type Night struct {
	Date   string `json:"date"`
	Price  int64  `json:"price"`
	Booked bool   `json:"booked"`
}

// Availability returns the calendar of a room for [from, to): the price of every night and whether it is taken.
func (s *Service) Availability(tenant TenantID, roomID, from, to string) ([]Night, error) {
	in, errIn := time.Parse(dateLayout, from)
	out, errOut := time.Parse(dateLayout, to)
	if errIn != nil || errOut != nil || !out.After(in) || nightsBetween(in, out) > maxAvailabilityNights {
		return nil, fmt.Errorf("%w: dates", ErrInvalidInput)
	}
	room, err := s.repo.GetRoom(tenant, roomID)
	if err != nil {
		return nil, err
	}
	existing, err := s.repo.ListBookings(tenant, roomID)
	if err != nil {
		return nil, err
	}
	nights := make([]Night, nightsBetween(in, out))
	for i := range nights {
		day := in.AddDate(0, 0, i)
		nights[i] = Night{Date: day.Format(dateLayout), Price: nightlyRate(room.BasePrice, day), Booked: taken(existing, day)}
	}
	return nights, nil
}

// taken reports whether an active booking covers the night starting on day.
func taken(bookings []Booking, day time.Time) bool {
	for _, b := range bookings {
		if b.Active() && b.Overlaps(day, day.AddDate(0, 0, 1)) {
			return true
		}
	}
	return false
}
