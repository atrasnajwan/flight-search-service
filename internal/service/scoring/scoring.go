package scoring

import "flight-search-service/internal/domain"

func CalculateBestValueScore(f domain.Flight, minPrice, maxPrice float64, minDuration, maxDuration int) float64 {
	priceScore := getScore(f.Price.Amount, minPrice, maxPrice)

	durationScore := getScore(float64(f.Duration.TotalMinutes), float64(minDuration), float64(maxDuration))

	stopsScore := float64(f.Stops) * 0.5

	// Lower total = Higher Rank
	// 50% Price, 30% Duration, 20% Stops
	return (priceScore * 0.5) + (durationScore * 0.3) + (stopsScore * 0.2)
}

func CalculateRoundTripBestValueScore(r domain.RoundTrip, minPrice, maxPrice float64, minDuration, maxDuration int) float64 {
	priceScore := getScore(r.TotalPrice, minPrice, maxPrice)

	durationScore := getScore(float64(r.TotalDurationMinutes), float64(minDuration), float64(maxDuration))

	totalStops := r.Outbound.Stops + r.Inbound.Stops
	stopsScore := float64(totalStops) * 0.5

	// Lower total = Higher Rank
	// 50% Price, 30% Duration, 20% Stops
	return (priceScore * 0.5) + (durationScore * 0.3) + (stopsScore * 0.2)
}

// get score range 0 - 1.0
func getScore(val, min, max float64) float64 {
	if max-min == 0 {
		return 0
	}
	return (val - min) / (max - min)
}
