package garuda

import (
	"context"
	"encoding/json"
	"errors"
	"flight-search-service/internal/domain"
	"flight-search-service/internal/helper"
	"fmt"
	"log"
	"math"
	"math/rand"
	"os"
	"strings"
	"time"
)

type GarudaProvider struct {
	mockFilePath       string
	minDelay, maxDelay int // ms
}

func NewGarudaProvider(path string, minDelay, maxDelay int) domain.Provider {
	return &GarudaProvider{mockFilePath: path, minDelay: minDelay, maxDelay: maxDelay}
}

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
	Aircraft        string             `json:"aircraft"`
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

const (
	garudaMaxRetries     = 3
	garudaInitialBackoff = 100 * time.Millisecond
	garudaBackoffFactor  = 2
	garudaTimeLayout     = "2006-01-02T15:04:05Z07:00"
)

func (p *GarudaProvider) Name() string {
	return "Garuda Indonesia"
}

func (p *GarudaProvider) Search(ctx context.Context, req domain.SearchRequest) ([]domain.Flight, error) {
	var lastErr error
	for attempt := 1; attempt <= garudaMaxRetries; attempt++ {
		flights, err := p.searchOnce(ctx, req)
		if err == nil {
			return flights, nil
		}

		lastErr = err
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) || ctx.Err() != nil {
			return nil, ctx.Err()
		}

		if attempt == garudaMaxRetries {
			break
		}
		// 100 * (2^(attempt-1))
		backoff := garudaInitialBackoff * time.Duration(math.Pow(garudaBackoffFactor, float64(attempt)-1))
		log.Printf("retry in %d", backoff.Milliseconds())
		select {
		case <-time.After(backoff):
		case <-ctx.Done():
			return []domain.Flight{}, ctx.Err()
		}
	}

	return []domain.Flight{}, fmt.Errorf("garuda search failed after %d attempts: %w", garudaMaxRetries, lastErr)
}

func (p *GarudaProvider) searchOnce(ctx context.Context, req domain.SearchRequest) ([]domain.Flight, error) {
	// simulate delay
	delay := time.Duration(p.minDelay+rand.Intn(p.maxDelay-p.minDelay)) * time.Millisecond

	select {
	case <-time.After(delay):
		// continue processing after delay
	case <-ctx.Done(): // when context is cancelled or timeout
		return nil, ctx.Err()
	}

	// read the mock JSON file
	data, err := os.ReadFile(p.mockFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read garuda mock: %w", err)
	}

	var response RawGarudaResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal garuda response: %w", err)
	}

	if !strings.EqualFold(response.Status, "success") {
		return nil, fmt.Errorf("garuda mock returned non-success status: %s", response.Status)
	}

	normalized := make([]domain.Flight, 0, len(response.Flights))
	for _, f := range response.Flights {
		if req.Origin != "" && !strings.EqualFold(f.Departure.Airport, req.Origin) {
			continue
		}
		if req.Destination != "" && !strings.EqualFold(f.Arrival.Airport, req.Destination) {
			continue
		}
		if req.CabinClass != "" && !strings.EqualFold(req.CabinClass, f.FareClass) {
			continue
		}

		departureTime, err := time.Parse(garudaTimeLayout, f.Departure.Time)
		if err != nil {
			continue
		}
		arrivalTime, err := time.Parse(garudaTimeLayout, f.Arrival.Time)
		if err != nil {
			continue
		}

		if !req.DepartureDate.IsZero() && !helper.IsSameDate(departureTime, req.DepartureDate) {
			continue
		}

		if arrivalTime.Before(departureTime) {
			continue
		}

		// check available seats
		if f.AvailableSeats < req.Passengers {
			continue
		}

		duration := arrivalTime.Sub(departureTime)

		normalized = append(normalized, domain.Flight{
			ID:           helper.GetFlightID(f.ID, f.Airline),
			Provider:     f.Airline,
			Airline:      domain.Airline{Name: f.Airline, Code: f.AirlineCode},
			FlightNumber: f.ID,
			Departure: domain.FlightPoint{
				Airport:   f.Departure.Airport,
				City:      f.Departure.City,
				DateTime:  departureTime,
				Timestamp: departureTime.Unix(),
			},
			Arrival: domain.FlightPoint{
				Airport:   f.Arrival.Airport,
				City:      f.Arrival.City,
				DateTime:  arrivalTime,
				Timestamp: arrivalTime.Unix(),
			},
			Duration: domain.Duration{
				TotalMinutes: int(duration.Minutes()),
				Formatted:    helper.GetFormattedDuration(duration),
			},
			Stops: f.Stops,
			Price: domain.Price{
				Amount:    f.Price.Amount,
				Currency:  f.Price.Currency,
				Formatted: helper.FormatIDR(f.Price.Amount),
			},
			AvailableSeats: f.AvailableSeats,
			CabinClass:     f.FareClass,
			Aircraft:       f.Aircraft,
			Amenities:      helper.MapAmenities(f.Amenities),
		})
	}

	return normalized, nil
}