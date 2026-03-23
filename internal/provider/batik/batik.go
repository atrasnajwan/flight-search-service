package batik

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
	"os"
	"strings"
	"time"
)

type BatikProvider struct {
	mockFilePath       string
	airportInstance    *airport.Airport
	minDelay, maxDelay int // ms
}

func NewBatikProvider(path string, airportInstance    *airport.Airport, minDelay, maxDelay int) domain.Provider {
	return &BatikProvider{mockFilePath: path, airportInstance: airportInstance, minDelay: minDelay, maxDelay: maxDelay}
}

const (
	batikMaxRetries     = 3
	batikInitialBackoff = 100 * time.Millisecond
	batikBackoffFactor  = 2
	batikTimeLayout     = "2006-01-02T15:04:05Z0700"
)

func (p *BatikProvider) Name() string {
	return "Batik Air"
}

func (p *BatikProvider) Search(ctx context.Context, req domain.SearchRequest) ([]domain.Flight, error) {
	var lastErr error
	for attempt := 1; attempt <= batikMaxRetries; attempt++ {
		flights, err := p.searchOnce(ctx, req)
		if err == nil {
			return flights, nil
		}

		lastErr = err
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) || ctx.Err() != nil {
			return nil, ctx.Err()
		}

		if attempt == batikMaxRetries {
			break
		}
		// 100 * (2^(attempt-1))
		backoff := batikInitialBackoff * time.Duration(math.Pow(batikBackoffFactor, float64(attempt)-1))
		log.Printf("retry in %d", backoff.Milliseconds())
		select {
		case <-time.After(backoff):
		case <-ctx.Done():
			return []domain.Flight{}, ctx.Err()
		}
	}

	return []domain.Flight{}, fmt.Errorf("batik search failed after %d attempts: %w", batikMaxRetries, lastErr)
}

func (p *BatikProvider) searchOnce(ctx context.Context, req domain.SearchRequest) ([]domain.Flight, error) {
	err := helper.SimulateDelay(ctx, p.minDelay, p.maxDelay)
	if err != nil {
		return []domain.Flight{}, err
	}

	// read the mock JSON file
	data, err := os.ReadFile(p.mockFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read batik mock: %w", err)
	}

	var response RawBatikResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal batik response: %w", err)
	}

	if response.Code != 200 {
		return nil, fmt.Errorf("batik mock returned non-success code: %d", response.Code)
	}

	normalized := make([]domain.Flight, 0, len(response.Results))
	for _, f := range response.Results {
		if req.Origin != "" && !strings.EqualFold(f.Origin, req.Origin) {
			continue
		}
		if req.Destination != "" && !strings.EqualFold(f.Destination, req.Destination) {
			continue
		}

		if req.CabinClass != "" && !strings.EqualFold(req.CabinClass, GetCabinClass(f.Fare.Class)) {
			continue
		}

		departureTime, err := time.Parse(batikTimeLayout, f.DepartureDateTime)
		if err != nil {
			continue
		}
		arrivalTime, err := time.Parse(batikTimeLayout, f.ArrivalDateTime)
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
		if f.SeatsAvailable < req.Passengers {
			continue
		}

		duration := arrivalTime.Sub(departureTime)

		normalizedResponse := NormalizedResponse(p.airportInstance, &f, departureTime, arrivalTime, duration)
		// filters
		isEligible := helper.IsMatchFilter(&req, &normalizedResponse)
		if isEligible {
			normalized = append(normalized, normalizedResponse)
		}
	}

	return normalized, nil
}
