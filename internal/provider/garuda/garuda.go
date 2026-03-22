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
	err := helper.SimulateDelay(ctx, p.minDelay, p.maxDelay)
	if err != nil {
		return []domain.Flight{}, err
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

		normalizedResponse := NormalizedResponse(&f, departureTime, arrivalTime, duration)
		// filters
		isEligible := helper.IsMatchFilter(&req, &normalizedResponse)
		if isEligible {
			normalized = append(normalized, normalizedResponse)
		}
	}

	return normalized, nil
}


