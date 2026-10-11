// Package docs serves the OpenAPI 3.1 specification of the service and a
// Swagger UI page that renders it.
package docs

import (
	_ "embed"
	nethttp "net/http"
)

//go:embed openapi.yaml
var spec []byte

const swaggerUI = `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>API docs</title>
  <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/swagger-ui-dist@5.17.14/swagger-ui.css">
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="https://cdn.jsdelivr.net/npm/swagger-ui-dist@5.17.14/swagger-ui-bundle.js"></script>
  <script>
    window.ui = SwaggerUIBundle({ url: "openapi.yaml", dom_id: "#swagger-ui" });
  </script>
</body>
</html>`

// Register mounts GET /openapi.yaml and GET /api-docs (Swagger UI).
func Register(mux *nethttp.ServeMux) {
	mux.HandleFunc("GET /openapi.yaml", func(w nethttp.ResponseWriter, _ *nethttp.Request) {
		w.Header().Set("Content-Type", "application/yaml")
		_, _ = w.Write(spec)
	})
	mux.HandleFunc("GET /api-docs", func(w nethttp.ResponseWriter, _ *nethttp.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(swaggerUI))
	})
}
