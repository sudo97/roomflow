// Package httpapi exposes the booking service over HTTP/JSON.
package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/sudo97/roomflow/internal/booking"
)

const tenantHeader = "X-Tenant-ID"

// API wires HTTP handlers to the booking service.
type API struct {
	svc *booking.Service
}

// New creates the HTTP API.
func New(svc *booking.Service) *API { return &API{svc: svc} }

// tenantHandler is a handler that receives the tenant resolved from the request.
type tenantHandler func(w http.ResponseWriter, r *http.Request, tenant booking.TenantID)

// Routes returns the HTTP router.
func (a *API) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.Handle("GET /api/rooms", withTenant(a.listRooms))
	mux.Handle("POST /api/rooms", withTenant(a.addRoom))
	mux.Handle("POST /api/bookings", withTenant(a.createBooking))
	mux.Handle("POST /api/bookings/{id}/cancel", withTenant(a.cancelBooking))
	return mux
}

// withTenant rejects requests without a tenant header (tenant isolation at the edge).
func withTenant(h tenantHandler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tenant := booking.TenantID(r.Header.Get(tenantHeader))
		if tenant == "" {
			writeError(w, http.StatusUnauthorized, errors.New("missing tenant"))
			return
		}
		h(w, r, tenant)
	})
}

func (a *API) listRooms(w http.ResponseWriter, _ *http.Request, tenant booking.TenantID) {
	rooms, err := a.svc.Rooms(tenant)
	if err != nil {
		writeError(w, statusFor(err), err)
		return
	}
	writeJSON(w, http.StatusOK, rooms)
}

type roomRequest struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Capacity int    `json:"capacity"`
	Price    int64  `json:"price"`
}

func (a *API) addRoom(w http.ResponseWriter, r *http.Request, tenant booking.TenantID) {
	var req roomRequest
	if !decode(w, r, &req) {
		return
	}
	room, err := a.svc.AddRoom(booking.Room{ID: req.ID, TenantID: tenant, Name: req.Name, Capacity: req.Capacity, BasePrice: req.Price})
	if err != nil {
		writeError(w, statusFor(err), err)
		return
	}
	writeJSON(w, http.StatusCreated, room)
}

type bookingRequest struct {
	RoomID   string `json:"room_id"`
	Name     string `json:"guest_name"`
	Email    string `json:"guest_email"`
	Guests   int    `json:"guests"`
	CheckIn  string `json:"check_in"`
	CheckOut string `json:"check_out"`
	Promo    string `json:"promo"`
}

func (a *API) createBooking(w http.ResponseWriter, r *http.Request, tenant booking.TenantID) {
	var req bookingRequest
	if !decode(w, r, &req) {
		return
	}
	b, err := a.svc.CreateBooking(booking.BookingRequest{
		Tenant: tenant, RoomID: req.RoomID, GuestName: req.Name, GuestEmail: req.Email,
		Guests: req.Guests, CheckIn: req.CheckIn, CheckOut: req.CheckOut, Promo: req.Promo,
	})
	if err != nil {
		writeError(w, statusFor(err), err)
		return
	}
	writeJSON(w, http.StatusCreated, b)
}

func (a *API) cancelBooking(w http.ResponseWriter, r *http.Request, tenant booking.TenantID) {
	if err := a.svc.CancelBooking(tenant, r.PathValue("id")); err != nil {
		writeError(w, statusFor(err), err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// statusFor maps domain errors to HTTP status codes.
func statusFor(err error) int {
	switch {
	case errors.Is(err, booking.ErrInvalidInput):
		return http.StatusBadRequest
	case errors.Is(err, booking.ErrNotFound):
		return http.StatusNotFound
	case errors.Is(err, booking.ErrUnavailable):
		return http.StatusConflict
	default:
		return http.StatusInternalServerError
	}
}

func decode(w http.ResponseWriter, r *http.Request, dst any) bool {
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		writeError(w, http.StatusBadRequest, errors.New("malformed JSON body"))
		return false
	}
	return true
}

func writeError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, map[string]string{"error": err.Error()})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v) // the client has gone away if this fails; nothing to recover
}
