package booking

import "testing"

// «Слабкий» тест для демонстрації мутаційного тестування: виконує весь код ціноутворення
// (покриття 100%), але перевіряє лише, що ціна додатна.
func TestQuote(t *testing.T) {
	room := Room{BasePrice: 1000}
	cases := []struct {
		in, out string
		guests  int
		promo   string
	}{
		{"2026-03-02", "2026-03-04", 1, ""},
		{"2026-03-07", "2026-03-09", 1, ""},
		{"2026-07-01", "2026-07-02", 1, ""},
		{"2026-12-22", "2026-12-23", 1, ""},
		{"2026-03-02", "2026-03-04", 1, "WELCOME10"},
		{"2026-03-02", "2026-03-10", 1, "LONGSTAY"},
		{"2026-03-02", "2026-03-03", 3, "FRIENDS"},
	}
	for _, c := range cases {
		st, err := Request{Tenant: "t", RoomID: "r", GuestName: "g", GuestEmail: "g@x.io", Guests: c.guests, CheckIn: c.in, CheckOut: c.out, Promo: c.promo}.validate()
		if err != nil {
			t.Fatal(err)
		}
		if quote(room, st) <= 0 {
			t.Fatal("price must be positive")
		}
	}
}
