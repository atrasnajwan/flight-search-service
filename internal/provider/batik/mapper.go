package batik

import (
	"flight-search-service/internal/domain"
	"flight-search-service/internal/helper"
	"flight-search-service/internal/repository/airport"
	"time"
)

type RawBatikConnection struct {
	StopAirport  string `json:"stopAirport"`
	StopDuration string `json:"stopDuration"`
}

type RawBatikFare struct {
	BasePrice    float64 `json:"basePrice"`
	Taxes        float64 `json:"taxes"`
	TotalPrice   float64 `json:"totalPrice"`
	CurrencyCode string  `json:"currencyCode"`
	Class        string  `json:"class"`
}

type RawBatikFlight struct {
	FlightNumber      string               `json:"flightNumber"`
	AirlineName       string               `json:"airlineName"`
	AirlineIATA       string               `json:"airlineIATA"`
	Origin            string               `json:"origin"`
	Destination       string               `json:"destination"`
	DepartureDateTime string               `json:"departureDateTime"`
	ArrivalDateTime   string               `json:"arrivalDateTime"`
	TravelTime        string               `json:"travelTime"`
	NumberOfStops     int                  `json:"numberOfStops"`
	Connections       []RawBatikConnection `json:"connections,omitempty"`
	Fare              RawBatikFare         `json:"fare"`
	SeatsAvailable    int                  `json:"seatsAvailable"`
	AircraftModel     string               `json:"aircraftModel"`
	BaggageInfo       string               `json:"baggageInfo"`
	OnboardServices   []string             `json:"onboardServices"`
}

type RawBatikResponse struct {
	Code    int              `json:"code"`
	Message string           `json:"message"`
	Results []RawBatikFlight `json:"results"`
}

func NormalizedResponse(airportInstance *airport.Airport, raw *RawBatikFlight, departureTime, arrivalTime time.Time, duration time.Duration) domain.Flight {
	return domain.Flight{
		ID:           helper.GetFlightID(raw.FlightNumber, raw.AirlineName),
		Provider:     raw.AirlineName,
		Airline:      domain.Airline{Name: raw.AirlineName, Code: raw.AirlineIATA},
		FlightNumber: raw.FlightNumber,
		Departure: domain.FlightPoint{
			Airport:   raw.Origin,
			City:      airportInstance.GetCity(raw.Origin),
			DateTime:  departureTime,
			Timestamp: departureTime.Unix(),
		},
		Arrival: domain.FlightPoint{
			Airport:   raw.Destination,
			City:      airportInstance.GetCity(raw.Destination),
			DateTime:  arrivalTime,
			Timestamp: arrivalTime.Unix(),
		},
		Duration: domain.Duration{
			TotalMinutes: int(duration.Minutes()),
			Formatted:    helper.GetFormattedDuration(duration),
		},
		Stops: raw.NumberOfStops,
		Price: domain.Price{
			Amount:    raw.Fare.TotalPrice,
			Currency:  raw.Fare.CurrencyCode,
			Formatted: helper.FormatIDR(raw.Fare.TotalPrice),
		},
		AvailableSeats: raw.SeatsAvailable,
		CabinClass:     GetCabinClass(raw.Fare.Class),
		Aircraft:       raw.AircraftModel,
		Amenities:      helper.MapAmenities(raw.OnboardServices),
	}
}

func GetCabinClass(class string) string {
	switch(class) {
	case "Y":
		return "economy"
	default:
		return "economy"
	}
}