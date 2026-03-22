package helper

import (
	"flight-search-service/internal/domain"
	"fmt"
	"strings"
	"time"
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
