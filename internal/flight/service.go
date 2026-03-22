package flight

import (
	"context"
	"flight-search-service/internal/domain"
	"log"
	"sort"
	"time"
)

type FlightService struct {
	providers []domain.Provider
}

func NewService(providers []domain.Provider) *FlightService {
	return &FlightService{providers: providers}
}

func (s *FlightService) AggregateSearch(ctx context.Context, req domain.SearchRequest) ([]domain.Flight, Metadata) {
	start := time.Now()
    // TODO: implement cache
	cacheHit := false
	var providerSucceeded int
	var providerFailed int

	// channel to receive results
	resultChan := make(chan []domain.Flight, len(s.providers))

	// timeout for the entire search (1s)
	ctx, cancel := context.WithTimeout(ctx, 1000*time.Millisecond)
	defer cancel()

	for _, p := range s.providers {
		go func(provider domain.Provider) {
			flights, err := provider.Search(ctx, req)
			if err != nil {
				log.Printf("Provider %s failed: %v", provider.Name(), err)
				resultChan <- nil
				providerFailed += 1
				return
			}
			resultChan <- flights
			providerSucceeded += 1
		}(p)
	}

	var allFlights []domain.Flight
	for i := 0; i < len(s.providers); i++ {
		select {
		case flights := <-resultChan:
			if len(flights) > 0 {
				allFlights = append(allFlights, flights...)
			}
		case <-ctx.Done():
			flights := s.sortResults(allFlights)
            meta := Metadata{
                TotalResults:      len(flights),
                ProviderQueried:   len(s.providers),
                ProviderSucceeded: providerSucceeded,
                ProviderFailed:    providerFailed,
                SearchTime:        getSearchDuration(start),
                CacheHit:          cacheHit,
            }

            return flights, meta
		}
	}

	flights := s.sortResults(allFlights)
	
	meta := Metadata{
		TotalResults:      len(flights),
		ProviderQueried:   len(s.providers),
		ProviderSucceeded: providerSucceeded,
		ProviderFailed:    providerFailed,
		SearchTime:        getSearchDuration(start),
		CacheHit:          cacheHit,
	}

	return flights, meta
}

func (s *FlightService) sortResults(flights []domain.Flight) []domain.Flight {
	if len(flights) <= 0 {
		return []domain.Flight{}
	}
	// sort by price ASC
	sort.Slice(flights, func(a, b int) bool {
		return flights[a].Price.Amount < flights[b].Price.Amount
	})

	return flights
}

func getSearchDuration(start time.Time) int {
    elapsed := time.Since(start)
    return int(elapsed.Milliseconds())
}