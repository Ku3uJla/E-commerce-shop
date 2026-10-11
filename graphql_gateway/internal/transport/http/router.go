package http

import (
	nethttp "net/http"
	"strings"

	graphql "github.com/graph-gophers/graphql-go"
	"github.com/graph-gophers/graphql-go/relay"

	"graphql_gateway/internal/graph"
)

// sandboxHTML embeds Apollo Sandbox pointed at this very endpoint.
const sandboxHTML = `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>GraphQL Sandbox</title>
  <style>html, body { margin: 0; height: 100%; } #sandbox { height: 100vh; }</style>
</head>
<body>
  <div id="sandbox"></div>
  <script src="https://embeddable-sandbox.cdn.apollographql.com/_latest/embeddable-sandbox.umd.production.min.js"></script>
  <script>
    new window.EmbeddedSandbox({
      target: "#sandbox",
      initialEndpoint: location.origin + location.pathname,
    });
  </script>
</body>
</html>`

// NewRouter wires GET /healthz and /graphql. In dev, GET /graphql from a
// browser serves Apollo Sandbox; POST /graphql executes queries.
func NewRouter(schema *graphql.Schema, catalog graph.ProductFetcher, dev bool) nethttp.Handler {
	mux := nethttp.NewServeMux()

	mux.HandleFunc("GET /healthz", func(w nethttp.ResponseWriter, _ *nethttp.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	api := withLoaders(catalog, &relay.Handler{Schema: schema})
	mux.Handle("POST /graphql", api)
	mux.HandleFunc("GET /graphql", func(w nethttp.ResponseWriter, r *nethttp.Request) {
		if !dev || !strings.Contains(r.Header.Get("Accept"), "text/html") {
			nethttp.Error(w, "send queries with POST", nethttp.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(sandboxHTML))
	})
	mux.HandleFunc("OPTIONS /graphql", func(w nethttp.ResponseWriter, _ *nethttp.Request) {
		w.WriteHeader(nethttp.StatusNoContent)
	})

	return withCORS(mux)
}

// withLoaders attaches fresh DataLoaders to every request.
func withLoaders(catalog graph.ProductFetcher, next nethttp.Handler) nethttp.Handler {
	return nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
		ctx := graph.WithLoaders(r.Context(), graph.NewLoaders(catalog))
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func withCORS(next nethttp.Handler) nethttp.Handler {
	return nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		next.ServeHTTP(w, r)
	})
}
