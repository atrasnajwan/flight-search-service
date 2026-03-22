package domain

import "time"

type Flight struct {
	ID             string      `json:"id"`
	Provider       string      `json:"provider"`
	Airline        Airline     `json:"airline"`
	FlightNumber   string      `json:"flight_number"`
	Departure      FlightPoint `json:"departure"`
	Arrival        FlightPoint `json:"arrival"`
	Duration       Duration    `json:"duration"`
	Stops          int         `json:"stops"`
	Price          Price       `json:"price"`
	AvailableSeats int         `json:"available_seats"`
	CabinClass     string      `json:"cabin_class"`
	Aircraft       string      `json:"aircraft"`
	Amenities      []Amenity   `json:"amenities"`
}

type Price struct {
	Amount    float64 `json:"amount"`
	Currency  string  `json:"currency"`
	Formatted string  `json:"formatted"`
}

type FlightPoint struct {
	Airport   string    `json:"airport"`
	City      string    `json:"city"`
	DateTime  time.Time `json:"datetime"`
	Timestamp int64     `json:"timestamp"`
}

type Duration struct {
	TotalMinutes int    `json:"total_minutes"`
	Formatted    string `json:"formatted"`
}

type Amenity string

const (
	AmenityWiFi          Amenity = "wifi"
	AmenityMeal          Amenity = "meal"
	AmenityEntertainment Amenity = "entertainment"
)
