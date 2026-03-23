package airasia

import (
	"context"
	"encoding/json"
	"errors"
	"flight-search-service/internal/domain"
	"flight-search-service/internal/helper"
	"flight-search-service/internal/repository/airport"
	"fmt"
	"log"
	"math"
	"math/rand"
	"os"
	"strings"
	"time"
)

type AirAsiaProvider struct {
	mockFilePath       string
	airportInstance    *airport.Airport
	minDelay, maxDelay int // ms
	succesRate         int // 0 - 100
}

func NewAirAsiaProvider(path string, airportInstance *airport.Airport, minDelay, maxDelay, succesRate int) domain.Provider {
	rate := max(0, min(100, succesRate)) // min 0, max 100
	return &AirAsiaProvider{
		mockFilePath:    path,
		airportInstance: airportInstance,
		minDelay:        minDelay,
		maxDelay:        maxDelay,
		succesRate:      rate,
	}
}

const (
	airasiaMaxRetries     = 3
	airasiaInitialBackoff = 100 * time.Millisecond
	airasiaBackoffFactor  = 2
	airasiaTimeLayout     = "2006-01-02T15:04:05Z07:00"
)

func (p *AirAsiaProvider) Name() string {
	return "AirAsia"
}

func (p *AirAsiaProvider) Search(ctx context.Context, req domain.SearchRequest) ([]domain.Flight, error) {
	var lastErr error
	for attempt := 1; attempt <= airasiaMaxRetries; attempt++ {
		flights, err := p.searchOnce(ctx, req)
		if err == nil {
			return flights, nil
		}

		lastErr = err
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) || ctx.Err() != nil {
			return nil, ctx.Err()
		}

		if attempt == airasiaMaxRetries {
			break
		}
		// 100 * (2^(attempt-1))
		backoff := airasiaInitialBackoff * time.Duration(math.Pow(airasiaBackoffFactor, float64(attempt)-1))
		log.Printf("retry in %d", backoff.Milliseconds())
		select {
		case <-time.After(backoff):
		case <-ctx.Done():
			return []domain.Flight{}, ctx.Err()
		}
	}

	return []domain.Flight{}, fmt.Errorf("airasia search failed after %d attempts: %w", airasiaMaxRetries, lastErr)
}

func (p *AirAsiaProvider) searchOnce(ctx context.Context, req domain.SearchRequest) ([]domain.Flight, error) {
	err := helper.SimulateDelay(ctx, p.minDelay, p.maxDelay)
	if err != nil {
		return []domain.Flight{}, err
	}

	if rand.Intn(100) > p.succesRate {
		return nil, fmt.Errorf("failed to fetch airasia")
	}
	// read the mock JSON file
	data, err := os.ReadFile(p.mockFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read airasia mock: %w", err)
	}

	var response RawAirAsiaResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal airasia response: %w", err)
	}

	if !strings.EqualFold(response.Status, "ok") {
		return nil, fmt.Errorf("airasia mock returned non-success status: %s", response.Status)
	}

	normalized := make([]domain.Flight, 0, len(response.Flights))
	for _, f := range response.Flights {
		if req.Origin != "" && !strings.EqualFold(f.FromAirport, req.Origin) {
			continue
		}
		if req.Destination != "" && !strings.EqualFold(f.ToAirport, req.Destination) {
			continue
		}
		if req.CabinClass != "" && !strings.EqualFold(req.CabinClass, f.CabinClass) {
			continue
		}

		departureTime, err := time.Parse(airasiaTimeLayout, f.DepartTime)
		if err != nil {
			continue
		}
		arrivalTime, err := time.Parse(airasiaTimeLayout, f.ArriveTime)
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
		if f.Seats < req.Passengers {
			continue
		}

		duration := time.Duration(f.DurationHours * 60 * 60 * 1000 * 1000 * 1000) // hours to nanoseconds

		normalizedResponse := NormalizedResponse(p.airportInstance, &f, departureTime, arrivalTime, duration)
		// filters
		isEligible := helper.IsMatchFilter(&req, &normalizedResponse)
		if isEligible {
			normalized = append(normalized, normalizedResponse)
		}
	}

	return normalized, nil
}
