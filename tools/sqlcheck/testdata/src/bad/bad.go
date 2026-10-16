package bad

import (
	"database/sql"
	"fmt"
)

func sprintf(db *sql.DB, tenant string) {
	q := fmt.Sprintf("SELECT * FROM bookings WHERE tenant_id = '%s'", tenant)
	db.Query(q)
}

func concat(db *sql.DB, tenant string) {
	db.Exec("DELETE FROM bookings WHERE tenant_id = '" + tenant + "'")
}
