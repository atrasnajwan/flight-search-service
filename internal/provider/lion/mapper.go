package lion

import (
	"flight-search-service/internal/domain"
	"flight-search-service/internal/helper"
	"strings"
	"time"
)

type RawLionCarrier struct {
	Name string `json:"name"`
	Iata string `json:"iata"`
}

type RawLionAirport struct {
	Code string `json:"code"`
	Name string `json:"name"`
	City string `json:"city"`
}

type RawLionRoute struct {
	From RawLionAirport `json:"from"`
	To   RawLionAirport `json:"to"`
}

type RawLionSchedule struct {
	Departure         string `json:"departure"`
	DepartureTimezone string `json:"departure_timezone"`
	Arrival           string `json:"arrival"`
	ArrivalTimezone   string `json:"arrival_timezone"`
}

type RawLionPricing struct {
	Total    float64 `json:"total"`
	Currency string  `json:"currency"`
	FareType string  `json:"fare_type"`
}

type RawLionLayover struct {
	Airport         string `json:"airport"`
	DurationMinutes int    `json:"duration_minutes"`
}

type RawLionServices struct {
	WifiAvailable    bool              `json:"wifi_available"`
	MealsIncluded    bool              `json:"meals_included"`
	BaggageAllowance map[string]string `json:"baggage_allowance"`
}

type RawLionFlight struct {
	Id         string           `json:"id"`
	Carrier    RawLionCarrier   `json:"carrier"`
	Route      RawLionRoute     `json:"route"`
	Schedule   RawLionSchedule  `json:"schedule"`
	FlightTime int              `json:"flight_time"`
	IsDirect   bool             `json:"is_direct"`
	StopCount  int              `json:"stop_count,omitempty"`
	Layovers   []RawLionLayover `json:"layovers,omitempty"`
	Pricing    RawLionPricing   `json:"pricing"`
	SeatsLeft  int              `json:"seats_left"`
	PlaneType  *string           `json:"plane_type"`
	Services   RawLionServices  `json:"services"`
}

type RawLionData struct {
	AvailableFlights []RawLionFlight `json:"available_flights"`
}

type RawLionResponse struct {
	Success bool        `json:"success"`
	Data    RawLionData `json:"data"`
}

func NormalizedResponse(raw *RawLionFlight, departureTime, arrivalTime time.Time, duration time.Duration, stops int) domain.Flight {
	amenities := []string{}
	if raw.Services.WifiAvailable {
		amenities = append(amenities, "wifi")
	}
	if raw.Services.MealsIncluded {
		amenities = append(amenities, "meal")
	}

	return domain.Flight{
		ID:           helper.GetFlightID(raw.Id, raw.Carrier.Name),
		Provider:     raw.Carrier.Name,
		Airline:      domain.Airline{Name: raw.Carrier.Name, Code: raw.Carrier.Iata},
		FlightNumber: raw.Id,
		Departure: domain.FlightPoint{
			Airport:   raw.Route.From.Code,
			City:      raw.Route.From.City,
			DateTime:  departureTime,
			Timestamp: departureTime.Unix(),
		},
		Arrival: domain.FlightPoint{
			Airport:   raw.Route.To.Code,
			City:      raw.Route.To.City,
			DateTime:  arrivalTime,
			Timestamp: arrivalTime.Unix(),
		},
		Duration: domain.Duration{
			TotalMinutes: int(duration.Minutes()),
			Formatted:    helper.GetFormattedDuration(duration),
		},
		Stops: stops,
		Price: domain.Price{
			Amount:    raw.Pricing.Total,
			Currency:  raw.Pricing.Currency,
			Formatted: helper.FormatIDR(raw.Pricing.Total),
		},
		AvailableSeats: raw.SeatsLeft,
		CabinClass:     strings.ToLower(raw.Pricing.FareType),
		Aircraft:       raw.PlaneType,
		Amenities:      helper.MapAmenities(amenities),
	}
}
