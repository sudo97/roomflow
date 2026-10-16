package booking

import (
	"fmt"
	"net/mail"
	"time"
)

const dateLayout = "2006-01-02"

// BookingRequest is the input of the CreateBooking use case.
type BookingRequest struct {
	Tenant     TenantID
	RoomID     string
	GuestName  string
	GuestEmail string
	Guests     int
	CheckIn    string
	CheckOut   string
	Promo      string
}

// stay is a validated booking request.
type stay struct {
	BookingRequest
	in, out time.Time
}

func (r BookingRequest) validate() (stay, error) {
	if r.Tenant == "" || r.RoomID == "" || r.GuestName == "" || r.Guests <= 0 {
		return stay{}, fmt.Errorf("%w: tenant, room, guest name and guests are required", ErrInvalidInput)
	}
	if _, err := mail.ParseAddress(r.GuestEmail); err != nil {
		return stay{}, fmt.Errorf("%w: guest email", ErrInvalidInput)
	}
	in, errIn := time.Parse(dateLayout, r.CheckIn)
	out, errOut := time.Parse(dateLayout, r.CheckOut)
	if errIn != nil || errOut != nil || !out.After(in) {
		return stay{}, fmt.Errorf("%w: dates", ErrInvalidInput)
	}
	return stay{BookingRequest: r, in: in, out: out}, nil
}
