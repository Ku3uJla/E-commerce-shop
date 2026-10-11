package graph

import (
	"context"
	"time"

	"github.com/graph-gophers/dataloader/v7"

	"graphql_gateway/internal/clients"
)

// ProductFetcher loads many products at once.
type ProductFetcher interface {
	GetMany(ctx context.Context, ids []string) ([]clients.Product, error)
}

// Loaders are created once per GraphQL request, so the cache never leaks
// data between requests or users.
type Loaders struct {
	Product *dataloader.Loader[string, clients.Product]
}

func NewLoaders(catalog ProductFetcher) *Loaders {
	return &Loaders{
		// All product loads collected within the wait window are de-duplicated
		// and resolved with a single Catalog request (?ids=a,b,c).
		Product: dataloader.NewBatchedLoader(
			productBatchFn(catalog),
			dataloader.WithWait[string, clients.Product](2*time.Millisecond),
			dataloader.WithBatchCapacity[string, clients.Product](clients.CatalogBatchSize),
		),
	}
}

func productBatchFn(catalog ProductFetcher) dataloader.BatchFunc[string, clients.Product] {
	return func(ctx context.Context, ids []string) []*dataloader.Result[clients.Product] {
		results := make([]*dataloader.Result[clients.Product], len(ids))

		products, err := catalog.GetMany(ctx, ids)
		if err != nil {
			for i := range results {
				results[i] = &dataloader.Result[clients.Product]{Error: toGraphQLError(err)}
			}
			return results
		}

		byID := make(map[string]clients.Product, len(products))
		for _, p := range products {
			byID[p.ID] = p
		}
		// DataLoader requires results in the same order as the keys.
		for i, id := range ids {
			if p, ok := byID[id]; ok {
				results[i] = &dataloader.Result[clients.Product]{Data: p}
			} else {
				results[i] = &dataloader.Result[clients.Product]{Error: newError("product "+id+" not found in catalog", "NOT_FOUND")}
			}
		}
		return results
	}
}

type loadersKey struct{}

// WithLoaders stores fresh loaders in the request context.
func WithLoaders(ctx context.Context, l *Loaders) context.Context {
	return context.WithValue(ctx, loadersKey{}, l)
}

func loadersFrom(ctx context.Context) *Loaders {
	return ctx.Value(loadersKey{}).(*Loaders)
}
