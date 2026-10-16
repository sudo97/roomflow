package booking

import "testing"

func TestQuote(t *testing.T) {
	room := Room{BasePrice: 1000}
	cases := []struct {
		name    string
		in, out string
		guests  int
		promo   string
		want    int64
	}{
		{"two weekdays", "2026-03-02", "2026-03-04", 1, "", 2000},
		{"weekend nights +20%", "2026-03-07", "2026-03-09", 1, "", 2400},
		{"high season +30%", "2026-07-01", "2026-07-02", 1, "", 1300},
		{"august weekend", "2026-08-01", "2026-08-02", 1, "", 1560},
		{"christmas week +50%", "2026-12-22", "2026-12-23", 1, "", 1500},
		{"early december is normal", "2026-12-01", "2026-12-02", 1, "", 1000},
		{"welcome promo -10%", "2026-03-02", "2026-03-04", 1, "WELCOME10", 1800},
		{"longstay needs 7 nights", "2026-03-02", "2026-03-08", 1, "LONGSTAY", 6200},
		{"longstay applies at 7", "2026-03-02", "2026-03-09", 1, "LONGSTAY", 6290},
		{"friends needs 3 guests", "2026-03-02", "2026-03-03", 2, "FRIENDS", 1000},
		{"friends applies at 3", "2026-03-02", "2026-03-03", 3, "FRIENDS", 950},
		{"unknown promo ignored", "2026-03-02", "2026-03-03", 1, "FREE", 1000},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			st, err := BookingRequest{Tenant: "t", RoomID: "r", GuestName: "g", GuestEmail: "g@x.io", Guests: c.guests, CheckIn: c.in, CheckOut: c.out, Promo: c.promo}.validate()
			if err != nil {
				t.Fatal(err)
			}
			if got := quote(room, st); got != c.want {
				t.Fatalf("quote = %d, want %d", got, c.want)
			}
		})
	}
}
