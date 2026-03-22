package domain

import "context"

type Provider interface {
    Name() string
    Search(ctx context.Context, req SearchRequest) ([]Flight, error)
}