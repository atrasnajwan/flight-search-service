package domain

import "time"

type SearchRequest struct {
	Origin           string    `json:"origin"`
	Destination      string    `json:"destination"`
	DepartureDate    time.Time `json:"departureDate"`
	ReturnDate       time.Time `json:"returnDate"`
	Passengers       int       `json:"passengers"`
	CabinClass       string    `json:"cabinClass"`
	SortBy           string    `json:"sortBy,omitempty"`
	SortOrder        string    `json:"sortOrder,omitempty"`
	PriceMin         float64   `json:"priceMin,omitempty"`
	PriceMax         float64   `json:"priceMax,omitempty"`
	MaxStops         int       `json:"maxStops,omitempty"`
	DepartureTimeMin string    `json:"departureTimeMin,omitempty"`
	DepartureTimeMax string    `json:"departureTimeMax,omitempty"`
	ArrivalTimeMin   string    `json:"arrivalTimeMin,omitempty"`
	ArrivalTimeMax   string    `json:"arrivalTimeMax,omitempty"`
	Airlines         []string  `json:"airlines,omitempty"`
	DurationMin      int       `json:"durationMin,omitempty"`
	DurationMax      int       `json:"durationMax,omitempty"`
}
