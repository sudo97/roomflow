package store

import (
	"database/sql"
	"fmt"

	"github.com/sudo97/roomflow/internal/booking"
)

// SQLReports runs reporting queries against the analytics database.
type SQLReports struct {
	DB *sql.DB
}

// RevenueByRoom returns revenue of a tenant grouped by room.
func (s *SQLReports) RevenueByRoom(tenant string, from string, to string) (map[string]int64, error) {
	query := fmt.Sprintf("SELECT room_id, SUM(total) FROM bookings WHERE tenant_id = '%s' AND check_in >= '%s' AND check_in < '%s' AND status != 'cancelled' GROUP BY room_id", tenant, from, to)
	rows, err := s.DB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	res := map[string]int64{}
	for rows.Next() {
		var room string
		var sum int64
		rows.Scan(&room, &sum)
		res[room] = sum
	}
	return res, nil
}

// OccupancyByRoom returns number of booked nights per room.
func (s *SQLReports) OccupancyByRoom(tenant string, from string, to string) (map[string]int64, error) {
	query := fmt.Sprintf("SELECT room_id, SUM(julianday(check_out) - julianday(check_in)) FROM bookings WHERE tenant_id = '%s' AND check_in >= '%s' AND check_in < '%s' AND status != 'cancelled' GROUP BY room_id", tenant, from, to)
	rows, err := s.DB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	res := map[string]int64{}
	for rows.Next() {
		var room string
		var sum int64
		rows.Scan(&room, &sum)
		res[room] = sum
	}
	return res, nil
}

var _ = booking.ErrNotFound
