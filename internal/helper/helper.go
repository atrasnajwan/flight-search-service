package helper

import (
	"context"
	"flight-search-service/internal/domain"
	"fmt"
	"math/rand"
	"strings"
	"time"
	"unicode"
)

// get flight_id {flightNumber}_{airline}
func GetFlightID(flightNumber, airlineName string) string {
	return fmt.Sprintf("%s_%s", flightNumber, strings.ReplaceAll(airlineName, " ", ""))
}

// get formatted duration like 1h 30m
func GetFormattedDuration(d time.Duration) string {
	totalMinutes := int(d.Minutes())
	if totalMinutes <= 0 {
		return "0m"
	}

	days := totalMinutes / (24 * 60)
	hours := (totalMinutes % (24 * 60)) / 60
	minutes := totalMinutes % 60

	var parts []string

	if days > 0 {
		parts = append(parts, fmt.Sprintf("%dd", days))
	}
	if hours > 0 {
		parts = append(parts, fmt.Sprintf("%dh", hours))
	}
	if minutes > 0 {
		parts = append(parts, fmt.Sprintf("%dm", minutes))
	}

	return strings.Join(parts, " ")
}

// map array string to match domain.Amenity
func MapAmenities(raw []string) []domain.Amenity {
	normalized := make([]domain.Amenity, 0, len(raw))
	for _, a := range raw {
		switch strings.ToLower(a) {
		case "wifi":
			normalized = append(normalized, domain.AmenityWiFi)
		case "meal", "snack":
			normalized = append(normalized, domain.AmenityMeal)
		case "entertainment":
			normalized = append(normalized, domain.AmenityEntertainment)
		default:
			normalized = append(normalized, domain.Amenity(strings.ToLower(a)))
		}
	}
	return normalized
}

// converts float64 to a string like "Rp 1.250.000"
func FormatIDR(amount float64) string {
	// drop decimals
	intAmount := int64(amount)

	s := fmt.Sprintf("%d", intAmount)

	var result []string
	length := len(s)

	// Iterate backwards
	for i := length; i > 0; i -= 3 {
		start := i - 3
		if start < 0 {
			start = 0
		}
		result = append([]string{s[start:i]}, result...)
	}

	// add dot(.) between price
	return fmt.Sprintf("Rp %s", strings.Join(result, "."))
}

// check if 2 dates is same date
func IsSameDate(a, b time.Time) bool {
	y1, m1, d1 := a.Date()
	y2, m2, d2 := b.Date()
	return y1 == y2 && m1 == m2 && d1 == d2
}

// parse string hh:mm to hours and minutes
func parseTime(timeStr string) (int, int, error) {
	var h, m int
	_, err := fmt.Sscanf(timeStr, "%d:%d", &h, &m)
	return h, m, err
}

// capitalize first letter on string
func CapitalizeFirst(s string) string {
	if s == "" {
		return s
	}
	// Convert string to a slice of runes
	runes := []rune(s)
	// Capitalize the first rune
	runes[0] = unicode.ToUpper(runes[0])

	return string(runes)
}

// simulate delay
func SimulateDelay(ctx context.Context, minDelay, maxDelay int) error {
	delay := time.Duration(minDelay+rand.Intn(maxDelay-minDelay)) * time.Millisecond

	select {
	case <-time.After(delay):
		// continue processing
		return nil
	case <-ctx.Done(): // when context is cancelled or timeout
		return ctx.Err()
	}
}

func IsMatchFilter(req *domain.SearchRequest, flight *domain.Flight) bool {
	if req.PriceMin > 0 && flight.Price.Amount < req.PriceMin {
		return false
	}
	if req.PriceMax > 0 && flight.Price.Amount > req.PriceMax {
		return false
	}
	if req.MaxStops >= 0 && flight.Stops > req.MaxStops {
		return false
	}
	if req.DurationMin > 0 && flight.Duration.TotalMinutes < req.DurationMin {
		return false
	}
	if req.DurationMax > 0 && flight.Duration.TotalMinutes > req.DurationMax {
		return false
	}
	if len(req.Airlines) > 0 {
		found := false
		for _, airline := range req.Airlines {
			if strings.EqualFold(flight.Airline.Name, airline) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	// Time filters
	if req.DepartureTimeMin != "" {
		if h, m, err := parseTime(req.DepartureTimeMin); err == nil {
			if flight.Departure.DateTime.Hour() < h || (flight.Departure.DateTime.Hour() == h && flight.Departure.DateTime.Minute() < m) {
				return false
			}
		}
	}
	if req.DepartureTimeMax != "" {
		if h, m, err := parseTime(req.DepartureTimeMax); err == nil {
			if flight.Departure.DateTime.Hour() > h || (flight.Departure.DateTime.Hour() == h && flight.Departure.DateTime.Minute() > m) {
				return false
			}
		}
	}
	if req.ArrivalTimeMin != "" {
		if h, m, err := parseTime(req.ArrivalTimeMin); err == nil {
			if flight.Arrival.DateTime.Hour() < h || (flight.Arrival.DateTime.Hour() == h && flight.Arrival.DateTime.Minute() < m) {
				return false
			}
		}
	}
	if req.ArrivalTimeMax != "" {
		if h, m, err := parseTime(req.ArrivalTimeMax); err == nil {
			if flight.Arrival.DateTime.Hour() > h || (flight.Arrival.DateTime.Hour() == h && flight.Arrival.DateTime.Minute() > m) {
				return false
			}
		}
	}
	return true
}

func IsValidTrip(req domain.SearchRequest, trip domain.TripResult) bool {
	totalPrice := trip.GetTotalPrice()
	totalDuration := trip.GetTotalDuration()

	if req.PriceMin > 0 && totalPrice < req.PriceMin {
		return false
	}
	if req.PriceMax > 0 && totalPrice > req.PriceMax {
		return false
	}
	if req.DurationMin > 0 && totalDuration < req.DurationMin {
		return false
	}
	if req.DurationMax > 0 && totalDuration > req.DurationMax {
		return false
	}

	return true
}
