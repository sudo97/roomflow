package booking

import (
	"fmt"
	"sync/atomic"
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
	seq  atomic.Int64
}

// NewService creates a booking service.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// CreateBooking validates the request, checks availability, calculates price and stores the booking.
func (s *Service) CreateBooking(req Request) (Booking, error) {
	st, err := req.validate()
	if err != nil {
		return Booking{}, err
	}
	room, err := s.repo.GetRoom(req.Tenant, req.RoomID)
	if err != nil {
		return Booking{}, err
	}
	if req.Guests > room.Capacity {
		return Booking{}, fmt.Errorf("%w: room capacity is %d", ErrInvalidInput, room.Capacity)
	}
	if err := s.ensureAvailable(st); err != nil {
		return Booking{}, err
	}
	b := Booking{
		ID:         fmt.Sprintf("bk-%d", s.seq.Add(1)),
		TenantID:   req.Tenant,
		RoomID:     req.RoomID,
		GuestName:  req.GuestName,
		GuestEmail: req.GuestEmail,
		Guests:     req.Guests,
		CheckIn:    st.in,
		CheckOut:   st.out,
		Total:      quote(room, st),
		Status:     StatusConfirmed,
	}
	if err := s.repo.SaveBooking(b); err != nil {
		return Booking{}, err
	}
	return b, nil
}

func (s *Service) ensureAvailable(st stay) error {
	existing, err := s.repo.ListBookings(st.Tenant, st.RoomID)
	if err != nil {
		return err
	}
	for _, e := range existing {
		if e.Active() && e.Overlaps(st.in, st.out) {
			return ErrUnavailable
		}
	}
	return nil
}

// CancelBooking marks a booking as cancelled. Cancelling twice is a no-op.
func (s *Service) CancelBooking(tenant TenantID, id string) error {
	b, err := s.repo.GetBooking(tenant, id)
	if err != nil {
		return err
	}
	if !b.Active() {
		return nil
	}
	b.Status = StatusCancelled
	return s.repo.SaveBooking(b)
}

// AddRoom registers a new room for a tenant.
func (s *Service) AddRoom(r Room) (Room, error) {
	if r.TenantID == "" || r.ID == "" || r.Name == "" || r.Capacity <= 0 || r.BasePrice <= 0 {
		return Room{}, ErrInvalidInput
	}
	return r, s.repo.SaveRoom(r)
}

// Rooms lists rooms of a tenant.
func (s *Service) Rooms(tenant TenantID) ([]Room, error) {
	return s.repo.ListRooms(tenant)
}
