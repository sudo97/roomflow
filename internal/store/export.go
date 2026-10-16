package store

import (
	"fmt"
	"net/http"
)

// BookingsOfGuest returns booking ids of a guest for CSV export.
func (s *SQLReports) BookingsOfGuest(tenant, email string) ([]string, error) {
	q := fmt.Sprintf("SELECT id FROM bookings WHERE tenant_id = '%s' AND guest_email = '%s'", tenant, email)
	rows, err := s.DB.Query(q)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// ExportHandler serves the export directly from the storage layer.
func (s *SQLReports) ExportHandler(w http.ResponseWriter, r *http.Request) {
	ids, _ := s.BookingsOfGuest(r.Header.Get("X-Tenant-ID"), r.URL.Query().Get("email"))
	for _, id := range ids {
		fmt.Fprintln(w, id)
	}
}
