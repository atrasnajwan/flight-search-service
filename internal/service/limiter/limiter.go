package limiter

import (
	"context"
	"flight-search-service/internal/domain"
	"log"

	"golang.org/x/time/rate"
)

type RatedProvider struct {
    target  domain.Provider
    limiter *rate.Limiter
}

func NewRatedProvider(p domain.Provider, r rate.Limit, b int) *RatedProvider {
    return &RatedProvider{
        target:  p,
        // r = rps, b = burst
        limiter: rate.NewLimiter(r, b), 
    }
}

func (rp *RatedProvider) Search(ctx context.Context, req domain.SearchRequest) ([]domain.Flight, error) {
    err := rp.limiter.Wait(ctx)
    if err != nil {
		log.Printf("failed when process provider %v", err)
        return nil, err // Context timeout or rate limit exceeded
    }

    return rp.target.Search(ctx, req)
}

func (rp *RatedProvider) Name() string { return rp.target.Name() }