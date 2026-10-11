package graph

import (
	"context"
	_ "embed"
	"time"

	graphql "github.com/graph-gophers/graphql-go"

	"graphql_gateway/internal/clients"
)

//go:embed schema.graphql
var schemaSDL string

// OrderLister loads the orders of a user.
type OrderLister interface {
	ListByUser(ctx context.Context, userID string) ([]clients.Order, error)
}

// NewSchema parses the SDL and binds it to the resolvers. Introspection (and
// with it the Docs explorer of the Sandbox) is enabled only in dev.
func NewSchema(orders OrderLister, introspection bool) *graphql.Schema {
	opts := []graphql.SchemaOpt{graphql.UseStringDescriptions()}
	if !introspection {
		opts = append(opts, graphql.DisableIntrospection())
	}
	return graphql.MustParseSchema(schemaSDL, &RootResolver{orders: orders}, opts...)
}

// RootResolver resolves the Query type.
type RootResolver struct {
	orders OrderLister
}

// OrdersByUserId resolves Query.ordersByUserId.
func (r *RootResolver) OrdersByUserId(ctx context.Context, args struct{ UserID graphql.ID }) ([]*orderResolver, error) {
	// Step 1: fetch the user's orders from the Order service.
	orders, err := r.orders.ListByUser(ctx, string(args.UserID))
	if err != nil {
		return nil, toGraphQLError(err)
	}

	// Step 2: queue every product id at once. The loader de-duplicates them
	// and issues a single Catalog request; OrderItem.product then reads the
	// cached results, so the number of upstream calls does not grow with N.
	var ids []string
	for _, o := range orders {
		for _, it := range o.Items {
			ids = append(ids, it.ProductID)
		}
	}
	loadersFrom(ctx).Product.LoadMany(ctx, ids)

	out := make([]*orderResolver, len(orders))
	for i := range orders {
		out[i] = &orderResolver{o: orders[i]}
	}
	return out, nil
}

type orderResolver struct {
	o clients.Order
}

func (r *orderResolver) ID() graphql.ID     { return graphql.ID(r.o.ID) }
func (r *orderResolver) UserID() graphql.ID { return graphql.ID(r.o.UserID) }
func (r *orderResolver) CreatedAt() string  { return r.o.CreatedAt.UTC().Format(time.RFC3339) }

func (r *orderResolver) Items() []*orderItemResolver {
	out := make([]*orderItemResolver, len(r.o.Items))
	for i := range r.o.Items {
		out[i] = &orderItemResolver{item: r.o.Items[i]}
	}
	return out
}

type orderItemResolver struct {
	item clients.OrderItem
}

func (r *orderItemResolver) Quantity() int32 { return int32(r.item.Quantity) }

func (r *orderItemResolver) Product(ctx context.Context) (*productResolver, error) {
	p, err := loadersFrom(ctx).Product.Load(ctx, r.item.ProductID)()
	if err != nil {
		return nil, err
	}
	return &productResolver{p: p}, nil
}

type productResolver struct {
	p clients.Product
}

func (r *productResolver) ID() graphql.ID { return graphql.ID(r.p.ID) }
func (r *productResolver) Name() string   { return r.p.Name }

// Price converts the Catalog's cents into the main currency unit.
func (r *productResolver) Price() float64 { return float64(r.p.Price) / 100 }
