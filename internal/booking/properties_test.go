package booking

import (
	"testing"
	"testing/quick"
	"time"
)

var epoch = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

func day(n uint16) time.Time { return epoch.AddDate(0, 0, int(n%730)) }

// Властивість: перетин інтервалів симетричний — якщо A перетинає B, то й B перетинає A.
func TestPropertyOverlapIsSymmetric(t *testing.T) {
	prop := func(a1, a2, b1, b2 uint16) bool {
		a := Booking{CheckIn: day(a1), CheckOut: day(a1).AddDate(0, 0, 1+int(a2%14))}
		b := Booking{CheckIn: day(b1), CheckOut: day(b1).AddDate(0, 0, 1+int(b2%14))}
		return a.Overlaps(b.CheckIn, b.CheckOut) == b.Overlaps(a.CheckIn, a.CheckOut)
	}
	if err := quick.Check(prop, &quick.Config{MaxCount: 2000}); err != nil {
		t.Fatal(err)
	}
}

// Властивість: будь-яке бронювання перетинає саме себе, а «стик у стик» — ні.
func TestPropertyBackToBackNeverOverlaps(t *testing.T) {
	prop := func(start, n1, n2 uint16) bool {
		in := day(start)
		mid := in.AddDate(0, 0, 1+int(n1%14))
		out := mid.AddDate(0, 0, 1+int(n2%14))
		first := Booking{CheckIn: in, CheckOut: mid}
		return first.Overlaps(in, mid) && !first.Overlaps(mid, out)
	}
	if err := quick.Check(prop, nil); err != nil {
		t.Fatal(err)
	}
}

// Властивості ціни: не від'ємна, промокод ніколи не збільшує ціну, довше проживання не дешевше (без промо).
func TestPropertyQuoteInvariants(t *testing.T) {
	room := Room{BasePrice: 1000}
	prop := func(start, nights uint16, guests uint8, promoIdx uint8) bool {
		in := day(start)
		n := 1 + int(nights%21)
		promos := []string{"", "WELCOME10", "LONGSTAY", "FRIENDS"}
		s := stay{Request: Request{Guests: 1 + int(guests%6), Promo: promos[int(promoIdx)%len(promos)]}, in: in, out: in.AddDate(0, 0, n)}
		noPromo := s
		noPromo.Promo = ""
		longer := noPromo
		longer.out = longer.out.AddDate(0, 0, 1)
		price := quote(room, s)
		return price > 0 && price <= quote(room, noPromo) && quote(room, longer) > quote(room, noPromo)
	}
	if err := quick.Check(prop, &quick.Config{MaxCount: 2000}); err != nil {
		t.Fatal(err)
	}
}

// Fuzz-тест валідації запиту: валідатор ніколи не панікує, а прийнятий запит має коректні дати.
func FuzzValidate(f *testing.F) {
	f.Add("2026-03-02", "2026-03-04", "a@b.io", 2)
	f.Add("", "", "", 0)
	f.Add("2026-02-30", "2026-13-01", "@", -1)
	f.Fuzz(func(t *testing.T, in, out, email string, guests int) {
		s, err := Request{Tenant: "t", RoomID: "r", GuestName: "g", GuestEmail: email, Guests: guests, CheckIn: in, CheckOut: out}.validate()
		if err == nil && (!s.out.After(s.in) || s.Guests <= 0) {
			t.Fatalf("accepted invalid request: %+v", s)
		}
	})
}

func BenchmarkQuoteTwoWeeks(b *testing.B) {
	room := Room{BasePrice: 120000}
	s := stay{Request: Request{Guests: 2, Promo: "LONGSTAY"}, in: epoch, out: epoch.AddDate(0, 0, 14)}
	for i := 0; i < b.N; i++ {
		quote(room, s)
	}
}
