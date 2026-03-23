package lion

import (
	"context"
	"encoding/json"
	"errors"
	"flight-search-service/internal/domain"
	"flight-search-service/internal/helper"
	"fmt"
	"log"
	"math"
	"os"
	"strings"
	"time"
)

type LionProvider struct {
	mockFilePath       string
	minDelay, maxDelay int // ms
}

func NewLionProvider(path string, minDelay, maxDelay int) domain.Provider {
	return &LionProvider{mockFilePath: path, minDelay: minDelay, maxDelay: maxDelay}
}

const (
	lionMaxRetries     = 3
	lionInitialBackoff = 100 * time.Millisecond
	lionBackoffFactor  = 2
	lionTimeLayout     = "2006-01-02T15:04:05"
)

func (p *LionProvider) Name() string {
	return "Lion Air"
}

func (p *LionProvider) Search(ctx context.Context, req domain.SearchRequest) ([]domain.Flight, error) {
	var lastErr error
	for attempt := 1; attempt <= lionMaxRetries; attempt++ {
		flights, err := p.searchOnce(ctx, req)
		if err == nil {
			return flights, nil
		}

		lastErr = err
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) || ctx.Err() != nil {
			return nil, ctx.Err()
		}

		if attempt == lionMaxRetries {
			break
		}
		// 100 * (2^(attempt-1))
		backoff := lionInitialBackoff * time.Duration(math.Pow(lionBackoffFactor, float64(attempt)-1))
		log.Printf("retry in %d", backoff.Milliseconds())
		select {
		case <-time.After(backoff):
		case <-ctx.Done():
			return []domain.Flight{}, ctx.Err()
		}
	}

	return []domain.Flight{}, fmt.Errorf("lion search failed after %d attempts: %w", lionMaxRetries, lastErr)
}

func (p *LionProvider) searchOnce(ctx context.Context, req domain.SearchRequest) ([]domain.Flight, error) {
	err := helper.SimulateDelay(ctx, p.minDelay, p.maxDelay)
	if err != nil {
		return []domain.Flight{}, err
	}

	// read the mock JSON file
	data, err := os.ReadFile(p.mockFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read lion mock: %w", err)
	}

	var response RawLionResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal lion response: %w", err)
	}

	if !response.Success {
		return nil, fmt.Errorf("lion mock returned non-success")
	}

	normalized := make([]domain.Flight, 0, len(response.Data.AvailableFlights))
	for _, f := range response.Data.AvailableFlights {
		if req.Origin != "" && !strings.EqualFold(f.Route.From.Code, req.Origin) {
			continue
		}
		if req.Destination != "" && !strings.EqualFold(f.Route.To.Code, req.Destination) {
			continue
		}
		if req.CabinClass != "" && !strings.EqualFold(req.CabinClass, f.Pricing.FareType) {
			continue
		}

		departureLoc, err := time.LoadLocation(f.Schedule.DepartureTimezone)
		if err != nil {
			departureLoc = time.UTC
		}
		departureTime, err := time.ParseInLocation(lionTimeLayout, f.Schedule.Departure, departureLoc)
		if err != nil {
			continue
		}

		arrivalLoc, err := time.LoadLocation(f.Schedule.ArrivalTimezone)
		if err != nil {
			arrivalLoc = time.UTC
		}
		arrivalTime, err := time.ParseInLocation(lionTimeLayout, f.Schedule.Arrival, arrivalLoc)
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
		if f.SeatsLeft < req.Passengers {
			continue
		}

		duration := time.Duration(f.FlightTime * 60 * 1000 * 1000 * 1000) // minutes to nanoseconds

		stops := 0
		if !f.IsDirect {
			stops = f.StopCount
		}

		normalizedResponse := NormalizedResponse(&f, departureTime, arrivalTime, duration, stops)
		// filters
		isEligible := helper.IsMatchFilter(&req, &normalizedResponse)
		if isEligible {
			normalized = append(normalized, normalizedResponse)
		}
	}

	return normalized, nil
}
