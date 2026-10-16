// Package httpapi exposes the booking service over HTTP/JSON.
package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/sudo97/roomflow/internal/booking"
)

// API wires HTTP handlers to the booking service.
type API struct {
	svc *booking.Service
}

// New creates the HTTP API.
func New(svc *booking.Service) *API { return &API{svc: svc} }

// Routes returns the HTTP router.
func (a *API) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("ok")) })
	mux.HandleFunc("GET /api/rooms", a.listRooms)
	mux.HandleFunc("POST /api/rooms", a.addRoom)
	mux.HandleFunc("POST /api/bookings", a.createBooking)
	mux.HandleFunc("POST /api/bookings/{id}/cancel", a.cancelBooking)
	return mux
}

func (a *API) listRooms(w http.ResponseWriter, r *http.Request) {
	tenant := r.Header.Get("X-Tenant-ID")
	if tenant == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "missing tenant"})
		return
	}
	rooms, err := a.svc.Rooms(tenant)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(rooms)
}

func (a *API) addRoom(w http.ResponseWriter, r *http.Request) {
	tenant := r.Header.Get("X-Tenant-ID")
	if tenant == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "missing tenant"})
		return
	}
	var req struct {
		ID       string `json:"id"`
		Name     string `json:"name"`
		Capacity int    `json:"capacity"`
		Price    int64  `json:"price"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	room, err := a.svc.AddRoom(tenant, req.ID, req.Name, req.Capacity, req.Price)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(room)
}

func (a *API) createBooking(w http.ResponseWriter, r *http.Request) {
	tenant := r.Header.Get("X-Tenant-ID")
	if tenant == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "missing tenant"})
		return
	}
	var req struct {
		RoomID   string `json:"room_id"`
		Name     string `json:"guest_name"`
		Email    string `json:"guest_email"`
		Guests   int    `json:"guests"`
		CheckIn  string `json:"check_in"`
		CheckOut string `json:"check_out"`
		Promo    string `json:"promo"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	b, err := a.svc.CreateBooking(tenant, req.RoomID, req.Name, req.Email, req.Guests, req.CheckIn, req.CheckOut, req.Promo)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, booking.ErrInvalidInput) {
			status = http.StatusBadRequest
		} else if errors.Is(err, booking.ErrNotFound) {
			status = http.StatusNotFound
		} else if errors.Is(err, booking.ErrUnavailable) {
			status = http.StatusConflict
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(b)
}

func (a *API) cancelBooking(w http.ResponseWriter, r *http.Request) {
	tenant := r.Header.Get("X-Tenant-ID")
	if tenant == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "missing tenant"})
		return
	}
	err := a.svc.CancelBooking(tenant, r.PathValue("id"))
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
