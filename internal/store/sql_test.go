package store

import (
	"database/sql"
	"database/sql/driver"
	"errors"
	"io"
	"strings"
	"testing"
)

// fakeDriver records the query and its arguments and returns canned rows.
type fakeDriver struct {
	lastQuery string
	lastArgs  []driver.Value
	rows      [][]driver.Value
	err       error
}

func (d *fakeDriver) Open(string) (driver.Conn, error) { return &fakeConn{d}, nil }

type fakeConn struct{ d *fakeDriver }

func (c *fakeConn) Prepare(q string) (driver.Stmt, error) { return &fakeStmt{c.d, q}, nil }
func (c *fakeConn) Close() error                          { return nil }
func (c *fakeConn) Begin() (driver.Tx, error)             { return nil, errors.New("no tx") }

type fakeStmt struct {
	d *fakeDriver
	q string
}

func (s *fakeStmt) Close() error                               { return nil }
func (s *fakeStmt) NumInput() int                              { return -1 }
func (s *fakeStmt) Exec([]driver.Value) (driver.Result, error) { return nil, errors.New("no exec") }
func (s *fakeStmt) Query(args []driver.Value) (driver.Rows, error) {
	s.d.lastQuery, s.d.lastArgs = s.q, args
	if s.d.err != nil {
		return nil, s.d.err
	}
	return &fakeRows{rows: s.d.rows}, nil
}

type fakeRows struct {
	rows [][]driver.Value
	i    int
}

func (r *fakeRows) Columns() []string { return []string{"room_id", "total"} }
func (r *fakeRows) Close() error      { return nil }
func (r *fakeRows) Next(dest []driver.Value) error {
	if r.i >= len(r.rows) {
		return io.EOF
	}
	copy(dest, r.rows[r.i])
	r.i++
	return nil
}

var registered = map[string]*fakeDriver{}

func openFake(t *testing.T, d *fakeDriver) *sql.DB {
	t.Helper()
	name := "fake-" + t.Name()
	if _, ok := registered[name]; !ok {
		sql.Register(name, d)
		registered[name] = d
	}
	db, err := sql.Open(name, "")
	if err != nil {
		t.Fatal(err)
	}
	return db
}

func TestRevenueByRoomIsParameterized(t *testing.T) {
	d := &fakeDriver{rows: [][]driver.Value{{"r1", int64(5000)}, {"r2", int64(700)}}}
	r := &SQLReports{DB: openFake(t, d)}
	evil := "a' OR '1'='1"
	got, err := r.RevenueByRoom(evil, "2026-01-01", "2026-02-01")
	if err != nil {
		t.Fatal(err)
	}
	if got["r1"] != 5000 || got["r2"] != 700 {
		t.Fatalf("got %v", got)
	}
	if strings.Contains(d.lastQuery, evil) || d.lastArgs[0] != evil {
		t.Fatalf("tenant must be passed as a parameter, query=%q args=%v", d.lastQuery, d.lastArgs)
	}
}

func TestOccupancyByRoomPropagatesErrors(t *testing.T) {
	d := &fakeDriver{err: errors.New("db down")}
	r := &SQLReports{DB: openFake(t, d)}
	if _, err := r.OccupancyByRoom("a", "2026-01-01", "2026-02-01"); err == nil {
		t.Fatal("expected error")
	}
}

func TestScanErrorIsReported(t *testing.T) {
	d := &fakeDriver{rows: [][]driver.Value{{"r1", "not-a-number"}}}
	r := &SQLReports{DB: openFake(t, d)}
	if _, err := r.OccupancyByRoom("a", "x", "y"); err == nil {
		t.Fatal("expected scan error")
	}
}
