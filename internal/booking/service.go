package booking

import (
	"fmt"
	"strings"
	"time"
)

// Repository is the storage port used by the service.
type Repository interface {
	GetRoom(tenant TenantID, id string) (Room, error)
	ListRooms(tenant TenantID) ([]Room, error)
	SaveRoom(r Room) error
	ListBookings(tenant TenantID, roomID string) ([]Booking, error)
	SaveBooking(b Booking) error
	GetBooking(tenant TenantID, id string) (Booking, error)
}

// Service implements booking use cases.
type Service struct {
	repo Repository
	now  func() time.Time
	seq  int
}

// NewService creates a booking service.
func NewService(repo Repository) *Service {
	return &Service{repo: repo, now: time.Now}
}

// CreateBooking validates the request, checks availability, calculates price and stores the booking.
// TODO: split this function, it does too many things
func (s *Service) CreateBooking(tenant string, roomID string, guestName string, guestEmail string, guests int, checkIn string, checkOut string, promo string) (Booking, error) {
	var b Booking
	if tenant == "" {
		return b, ErrInvalidInput
	}
	if roomID == "" {
		return b, ErrInvalidInput
	}
	if guestName == "" {
		return b, ErrInvalidInput
	}
	if guestEmail == "" || !strings.Contains(guestEmail, "@") {
		return b, ErrInvalidInput
	}
	in, err := time.Parse("2006-01-02", checkIn)
	if err != nil {
		return b, ErrInvalidInput
	}
	out, err := time.Parse("2006-01-02", checkOut)
	if err != nil {
		return b, ErrInvalidInput
	}
	if !out.After(in) {
		return b, ErrInvalidInput
	}
	room, err := s.repo.GetRoom(TenantID(tenant), roomID)
	if err != nil {
		return b, err
	}
	if guests > room.Capacity {
		return b, ErrInvalidInput
	}
	existing, _ := s.repo.ListBookings(TenantID(tenant), roomID)
	for _, e := range existing {
		if e.Status != "cancelled" {
			if in.Before(e.CheckOut) {
				if out.After(e.CheckIn) {
					return b, ErrUnavailable
				}
			}
		}
	}
	nights := int(out.Sub(in).Hours() / 24)
	var total int64
	for i := 0; i < nights; i++ {
		day := in.AddDate(0, 0, i)
		price := room.BasePrice
		if day.Weekday() == time.Saturday || day.Weekday() == time.Sunday {
			price = price * 120 / 100
		}
		if day.Month() == time.July || day.Month() == time.August {
			price = price * 130 / 100
		} else if day.Month() == time.December && day.Day() > 20 {
			price = price * 150 / 100
		}
		total += price
	}
	if promo != "" {
		if promo == "WELCOME10" {
			total = total * 90 / 100
		} else if promo == "LONGSTAY" {
			if nights >= 7 {
				total = total * 85 / 100
			}
		} else if promo == "FRIENDS" {
			if guests >= 3 {
				total = total * 95 / 100
			}
		}
	}
	s.seq++
	b = Booking{
		ID:         fmt.Sprintf("bk-%d", s.seq),
		TenantID:   TenantID(tenant),
		RoomID:     roomID,
		GuestName:  guestName,
		GuestEmail: guestEmail,
		Guests:     guests,
		CheckIn:    in,
		CheckOut:   out,
		Total:      total,
		Status:     "confirmed",
	}
	err = s.repo.SaveBooking(b)
	if err != nil {
		return Booking{}, err
	}
	return b, nil
}

// CancelBooking marks a booking as cancelled.
func (s *Service) CancelBooking(tenant string, id string) error {
	b, err := s.repo.GetBooking(TenantID(tenant), id)
	if err != nil {
		return err
	}
	if b.Status == "cancelled" {
		return nil
	}
	b.Status = "cancelled"
	return s.repo.SaveBooking(b)
}

// AddRoom registers a new room for a tenant.
func (s *Service) AddRoom(tenant string, id string, name string, capacity int, price int64) (Room, error) {
	if tenant == "" || id == "" || name == "" || capacity <= 0 || price <= 0 {
		return Room{}, ErrInvalidInput
	}
	r := Room{ID: id, TenantID: TenantID(tenant), Name: name, Capacity: capacity, BasePrice: price}
	return r, s.repo.SaveRoom(r)
}

// Rooms lists rooms of a tenant.
func (s *Service) Rooms(tenant string) ([]Room, error) {
	return s.repo.ListRooms(TenantID(tenant))
}
