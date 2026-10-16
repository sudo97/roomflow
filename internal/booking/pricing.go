package booking

import "time"

// percent applies a percentage multiplier to an amount in minor units.
func percent(amount int64, pct int64) int64 { return amount * pct / 100 }

// nightlyRate returns the price of a single night starting on day.
// Weekend nights cost +20%; high season (July, August) +30%; Christmas week (Dec 21–31) +50%.
func nightlyRate(base int64, day time.Time) int64 {
	price := base
	if isWeekend(day) {
		price = percent(price, 120)
	}
	return percent(price, seasonPct(day))
}

func isWeekend(day time.Time) bool {
	return day.Weekday() == time.Saturday || day.Weekday() == time.Sunday
}

func seasonPct(day time.Time) int64 {
	switch {
	case day.Month() == time.July || day.Month() == time.August:
		return 130
	case day.Month() == time.December && day.Day() > 20:
		return 150
	default:
		return 100
	}
}

// promotion describes a promo code: discount percentage and eligibility rule.
type promotion struct {
	pct      int64
	eligible func(s stay, nights int) bool
}

var promotions = map[string]promotion{
	"WELCOME10": {pct: 90, eligible: func(stay, int) bool { return true }},
	"LONGSTAY":  {pct: 85, eligible: func(_ stay, nights int) bool { return nights >= 7 }},
	"FRIENDS":   {pct: 95, eligible: func(s stay, _ int) bool { return s.Guests >= 3 }},
}

// quote calculates the total price of a stay for a room.
func quote(room Room, s stay) int64 {
	nights := nightsBetween(s.in, s.out)
	var total int64
	for i := 0; i < nights; i++ {
		total += nightlyRate(room.BasePrice, s.in.AddDate(0, 0, i))
	}
	if p, ok := promotions[s.Promo]; ok && p.eligible(s, nights) {
		total = percent(total, p.pct)
	}
	return total
}
