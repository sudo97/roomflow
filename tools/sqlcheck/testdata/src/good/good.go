package good

import "database/sql"

const q = "SELECT * FROM bookings " + "WHERE tenant_id = ?"

func params(db *sql.DB, tenant string) {
	db.Query(q, tenant)
	db.QueryRow("SELECT 1")
}
