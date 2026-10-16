package httpapi

import (
	"net/http"

	"github.com/sudo97/roomflow/internal/booking"
	"github.com/sudo97/roomflow/internal/flags"
	"github.com/sudo97/roomflow/internal/obs"
)

// FlagPriceBreakdown adds the per-night price breakdown to booking responses.
const FlagPriceBreakdown = "price_breakdown"

// WithFlags enables feature flags (GET /api/flags and flag-dependent behaviour).
func (a *API) WithFlags(f *flags.Store) *API { a.flags = f; return a }

// WithMetrics enables business metrics.
func (a *API) WithMetrics(m *obs.Registry) *API { a.metrics = m; return a }

func (a *API) flagOn(name string, tenant booking.TenantID) bool {
	return a.flags != nil && a.flags.Enabled(name, string(tenant))
}

func (a *API) count(name, help string) {
	if a.metrics != nil {
		a.metrics.Add(name, help, "", 1)
	}
}

func (a *API) listFlags(w http.ResponseWriter, _ *http.Request, tenant booking.TenantID) {
	if a.flags == nil {
		writeJSON(w, http.StatusOK, map[string]bool{})
		return
	}
	writeJSON(w, http.StatusOK, a.flags.All(string(tenant)))
}
