package flight

import (
	"context"
	"flight-search-service/internal/domain"
	"flight-search-service/internal/service"
	"log"
	"math"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type FlightService struct {
	providers []domain.Provider
}

func NewService(providers []domain.Provider) *FlightService {
	return &FlightService{providers: providers}
}

type SearchResponse struct {
	Flights    []domain.Flight    `json:"flights,omitempty"`
	RoundTrips []domain.RoundTrip `json:"round_trips,omitempty"`
	Meta       Metadata           `json:"meta"`
}

type Metadata struct {
	TotalResults      int  `json:"total_results"`
	ProviderQueried   int  `json:"providers_queried"`
	ProviderSucceeded int  `json:"providers_succeeded"`
	ProviderFailed    int  `json:"providers_failed"`
	SearchTime        int  `json:"search_time_ms"`
	CacheHit          bool `json:"cache_hit"`
}

type providerStats struct {
	succeeded int32
	failed    int32
}

func (s *FlightService) AggregateSearch(ctx context.Context, req domain.SearchRequest) (SearchResponse, error) {
	start := time.Now()
	// TODO: implement cache
	cacheHit := false

	flightResults, stats := s.fetchAll(ctx, req)

	return s.processResults(req, flightResults, stats, start, cacheHit), nil
}

func (s *FlightService) fetchAll(ctx context.Context, req domain.SearchRequest) ([]domain.Flight, *providerStats) {
	stats := &providerStats{}
	resultChan := make(chan []domain.Flight, len(s.providers))

	// 1s timeout
	ctx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()

	var wg sync.WaitGroup
	for _, p := range s.providers {
		wg.Add(1)
		go func(p domain.Provider) {
			defer wg.Done()

			flights, err := p.Search(ctx, req)
			if err != nil {
				log.Printf("Provider %s failed: %v", p.Name(), err)
				atomic.AddInt32(&stats.failed, 1)
				return
			}
			atomic.AddInt32(&stats.succeeded, 1)
			resultChan <- flights
		}(p)
	}

	// wait until all finished and close channel
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	var all []domain.Flight
	for flights := range resultChan {
		all = append(all, flights...)
	}
	return all, stats
}

func (s *FlightService) buildMeta(count int, stats *providerStats, start time.Time, cacheHit bool) Metadata {
	return Metadata{
		TotalResults:      count,
		ProviderQueried:   len(s.providers),
		ProviderSucceeded: int(stats.succeeded),
		ProviderFailed:    int(stats.failed),
		SearchTime:        getSearchDuration(start),
		CacheHit:          cacheHit,
	}
}

func (s *FlightService) processResults(
	req domain.SearchRequest,
	allFlights []domain.Flight,
	stats *providerStats,
	startTime time.Time,
	cacheHit bool,
) SearchResponse {
	s.applyScoring(allFlights)
	flights := s.sortResults(allFlights, req)

	return SearchResponse{
		Flights: flights,
		Meta:    s.buildMeta(len(flights), stats, startTime, cacheHit),
	}
}

func (s *FlightService) sortResults(flights []domain.Flight, req domain.SearchRequest) []domain.Flight {
	if len(flights) <= 0 {
		return []domain.Flight{}
	}

	order := strings.ToLower(req.SortOrder)
	if order != "asc" && order != "desc" {
		order = "asc"
	}

	switch strings.ToLower(req.SortBy) {
	case "price":
		sort.Slice(flights, func(a, b int) bool {
			if order == "desc" {
				return flights[a].Price.Amount > flights[b].Price.Amount
			}
			return flights[a].Price.Amount < flights[b].Price.Amount
		})
	case "duration":
		sort.Slice(flights, func(a, b int) bool {
			if order == "desc" {
				return flights[a].Duration.TotalMinutes > flights[b].Duration.TotalMinutes
			}
			return flights[a].Duration.TotalMinutes < flights[b].Duration.TotalMinutes
		})
	case "departure":
		sort.Slice(flights, func(a, b int) bool {
			if order == "desc" {
				return flights[a].Departure.DateTime.After(flights[b].Departure.DateTime)
			}
			return flights[a].Departure.DateTime.Before(flights[b].Departure.DateTime)
		})
	case "arrival":
		sort.Slice(flights, func(a, b int) bool {
			if order == "desc" {
				return flights[a].Arrival.DateTime.After(flights[b].Arrival.DateTime)
			}
			return flights[a].Arrival.DateTime.Before(flights[b].Arrival.DateTime)
		})
	default:
		// sort by scoring
		sort.Slice(flights, func(a, b int) bool {
			if order == "desc" {
				return flights[a].Score > flights[b].Score
			}
			return flights[a].Score < flights[b].Score
		})
	}

	return flights
}

func (s *FlightService) getGlobalMaxMin(flights []domain.Flight) (minP, maxP float64, minD, maxD int) {
	minP, minD = math.MaxFloat64, math.MaxInt

	for _, f := range flights {
		if f.Price.Amount < minP {
			minP = f.Price.Amount
		}
		if f.Price.Amount > maxP {
			maxP = f.Price.Amount
		}
		if f.Duration.TotalMinutes < minD {
			minD = f.Duration.TotalMinutes
		}
		if f.Duration.TotalMinutes > maxD {
			maxD = f.Duration.TotalMinutes
		}
	}
	return minP, maxP, minD, maxD
}

func (s *FlightService) applyScoring(flights []domain.Flight) {
	if len(flights) == 0 {
		return
	}

	minP, maxP, minD, maxD := s.getGlobalMaxMin(flights)

	for i := range flights {
		flights[i].Score = service.CalculateBestValueScore(flights[i], minP, maxP, minD, maxD)
	}
}

func getSearchDuration(start time.Time) int {
	elapsed := time.Since(start)
	return int(elapsed.Milliseconds())
}
