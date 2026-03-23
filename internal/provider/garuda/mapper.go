package garuda

import (
	"flight-search-service/internal/domain"
	"flight-search-service/internal/helper"
	"time"
)

type RawFlightPoint struct {
	Airport  string `json:"airport"`
	City     string `json:"city"`
	Time     string `json:"time"`
	Terminal string `json:"terminal"`
}

type RawSegmentPoint struct {
	Airport string `json:"airport"`
	Time    string `json:"time"`
}

type RawGarudaSegment struct {
	FlightNumber    string          `json:"flight_number"`
	Departure       RawSegmentPoint `json:"departure"`
	Arrival         RawSegmentPoint `json:"arrival"`
	DurationMinutes int             `json:"duration_minutes"`
	LayoverMinutes  int             `json:"layover_minutes"`
}

type RawPrice struct {
	Amount   float64 `json:"amount"`
	Currency string  `json:"currency"`
}
type RawBaggage struct {
	CarryOn int `json:"carry_on"`
	Checked int `json:"checked"`
}
type RawGarudaFlight struct {
	ID              string             `json:"flight_id"`
	Airline         string             `json:"airline"`
	AirlineCode     string             `json:"airline_code"`
	Departure       RawFlightPoint     `json:"departure"`
	Arrival         RawFlightPoint     `json:"arrival"`
	DurationMinutes int                `json:"duration_minutes"`
	Stops           int                `json:"stops"`
	Aircraft        *string             `json:"aircraft"`
	Price           RawPrice           `json:"price"`
	AvailableSeats  int                `json:"available_seats"`
	FareClass       string             `json:"fare_class"`
	Baggage         RawBaggage         `json:"baggage"`
	Amenities       []string           `json:"amenities"`
	Segments        []RawGarudaSegment `json:"segments,omitempty"`
}

type RawGarudaResponse struct {
	Status  string `json:"status"`
	Flights []RawGarudaFlight
}

func NormalizedResponse(raw *RawGarudaFlight, departureTime, arrivalTime time.Time, duration time.Duration) domain.Flight {
	return domain.Flight{
			ID:           helper.GetFlightID(raw.ID, raw.Airline),
			Provider:     raw.Airline,
			Airline:      domain.Airline{Name: raw.Airline, Code: raw.AirlineCode},
			FlightNumber: raw.ID,
			Departure: domain.FlightPoint{
				Airport:   raw.Departure.Airport,
				City:      raw.Departure.City,
				DateTime:  departureTime,
				Timestamp: departureTime.Unix(),
			},
			Arrival: domain.FlightPoint{
				Airport:   raw.Arrival.Airport,
				City:      raw.Arrival.City,
				DateTime:  arrivalTime,
				Timestamp: arrivalTime.Unix(),
			},
			Duration: domain.Duration{
				TotalMinutes: int(duration.Minutes()),
				Formatted:    helper.GetFormattedDuration(duration),
			},
			Stops: raw.Stops,
			Price: domain.Price{
				Amount:    raw.Price.Amount,
				Currency:  raw.Price.Currency,
				Formatted: helper.FormatIDR(raw.Price.Amount),
			},
			AvailableSeats: raw.AvailableSeats,
			CabinClass:     raw.FareClass,
			Aircraft:       raw.Aircraft,
			Amenities:      helper.MapAmenities(raw.Amenities),
		}
}