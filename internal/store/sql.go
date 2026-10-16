package store

import (
	"database/sql"
	"fmt"
)

// SQLReports runs reporting queries against the analytics database.
type SQLReports struct {
	DB *sql.DB
}

const (
	revenueQuery = `SELECT room_id, SUM(total) FROM bookings
WHERE tenant_id = ? AND check_in >= ? AND check_in < ? AND status != 'cancelled'
GROUP BY room_id`
	occupancyQuery = `SELECT room_id, SUM(julianday(check_out) - julianday(check_in)) FROM bookings
WHERE tenant_id = ? AND check_in >= ? AND check_in < ? AND status != 'cancelled'
GROUP BY room_id`
)

// RevenueByRoom returns revenue of a tenant grouped by room.
func (s *SQLReports) RevenueByRoom(tenant, from, to string) (map[string]int64, error) {
	return s.totalsByRoom(revenueQuery, tenant, from, to)
}

// OccupancyByRoom returns number of booked nights per room.
func (s *SQLReports) OccupancyByRoom(tenant, from, to string) (map[string]int64, error) {
	return s.totalsByRoom(occupancyQuery, tenant, from, to)
}

// totalsByRoom executes a parameterized "room_id, total" query; values are never concatenated into SQL.
func (s *SQLReports) totalsByRoom(query string, args ...any) (map[string]int64, error) {
	rows, err := s.DB.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("report query: %w", err)
	}
	defer func() { _ = rows.Close() }()
	res := map[string]int64{}
	for rows.Next() {
		var room string
		var sum int64
		if err := rows.Scan(&room, &sum); err != nil {
			return nil, fmt.Errorf("report scan: %w", err)
		}
		res[room] = sum
	}
	return res, rows.Err()
}
