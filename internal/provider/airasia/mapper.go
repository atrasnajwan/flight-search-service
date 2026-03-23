package airasia

import (
	"flight-search-service/internal/domain"
	"flight-search-service/internal/helper"
	"flight-search-service/internal/repository/airport"
	"strings"
	"time"
)

type RawAirAsiaStop struct {
	Airport         string `json:"airport"`
	WaitTimeMinutes int    `json:"wait_time_minutes"`
}

type RawAirAsiaFlight struct {
	FlightCode    string           `json:"flight_code"`
	Airline       string           `json:"airline"`
	FromAirport   string           `json:"from_airport"`
	ToAirport     string           `json:"to_airport"`
	DepartTime    string           `json:"depart_time"`
	ArriveTime    string           `json:"arrive_time"`
	DurationHours float64          `json:"duration_hours"`
	DirectFlight  bool             `json:"direct_flight"`
	Stops         []RawAirAsiaStop `json:"stops,omitempty"`
	PriceIdr      float64          `json:"price_idr"`
	Seats         int              `json:"seats"`
	CabinClass    string           `json:"cabin_class"`
	BaggageNote   string           `json:"baggage_note"`
}

type RawAirAsiaResponse struct {
	Status  string             `json:"status"`
	Flights []RawAirAsiaFlight `json:"flights"`
}

func NormalizedResponse(airportInstance *airport.Airport, raw *RawAirAsiaFlight, departureTime, arrivalTime time.Time, duration time.Duration) domain.Flight {
	stops := 0
	if !raw.DirectFlight {
		stops = len(raw.Stops)
		// no need add "wait_time_minutes", assuming duration is already include stop duration
	}

	return domain.Flight{
		ID:           helper.GetFlightID(raw.FlightCode, raw.Airline),
		Provider:     raw.Airline,
		Airline:      domain.Airline{Name: raw.Airline, Code: "QZ"}, // Assuming QZ
		FlightNumber: raw.FlightCode,
		Departure: domain.FlightPoint{
			Airport:   raw.FromAirport,
			City:      airportInstance.GetCity(raw.FromAirport),
			DateTime:  departureTime,
			Timestamp: departureTime.Unix(),
		},
		Arrival: domain.FlightPoint{
			Airport:   raw.ToAirport,
			City:      airportInstance.GetCity(raw.ToAirport),
			DateTime:  arrivalTime,
			Timestamp: arrivalTime.Unix(),
		},
		Duration: domain.Duration{
			TotalMinutes: int(duration.Minutes()),
			Formatted:    helper.GetFormattedDuration(duration),
		},
		Stops: stops,
		Price: domain.Price{
			Amount:    raw.PriceIdr,
			Currency:  "IDR",
			Formatted: helper.FormatIDR(raw.PriceIdr),
		},
		AvailableSeats: raw.Seats,
		CabinClass:     raw.CabinClass,
		Aircraft:       nil,
		Amenities:      helper.MapAmenities([]string{}), // not provided
		Baggage:        parseBaggage(raw.BaggageNote),
	}
}

// format: carry_on, checked
func parseBaggage(note string) domain.Baggage {
	parts := strings.Split(note, ",")
	carryOn := strings.TrimSpace(parts[0])
	checked := "Additional fee"
	if len(parts) > 1 {
		checked = strings.TrimSpace(strings.Replace(parts[1], "checked bags ", "", 1))
		checked = helper.CapitalizeFirst(checked)
	}
	return domain.Baggage{
		CarryOn: carryOn,
		Checked: checked,
	}
}
