package main

import (
	"context"
	"errors"
	"log"
	nethttp "net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"

	"graphql_gateway/internal/clients"
	"graphql_gateway/internal/config"
	"graphql_gateway/internal/graph"
	httptransport "graphql_gateway/internal/transport/http"
)

func main() {
	_ = godotenv.Load()

	cfg := config.Load()

	orders := clients.NewOrderClient(cfg.OrderServiceURL)
	catalog := clients.NewCatalogClient(cfg.CatalogServiceURL)
	schema := graph.NewSchema(orders, cfg.Dev)

	server := &nethttp.Server{
		Addr:         cfg.HTTPAddr,
		Handler:      httptransport.NewRouter(schema, catalog, cfg.Dev),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	go func() {
		log.Printf("graphql gateway listening on %s/graphql (dev=%t)", cfg.HTTPAddr, cfg.Dev)
		log.Printf("  orders  -> %s", cfg.OrderServiceURL)
		log.Printf("  catalog -> %s", cfg.CatalogServiceURL)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, nethttp.ErrServerClosed) {
			log.Fatalf("http server error: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Printf("graceful shutdown failed: %v", err)
	}
	log.Println("graphql gateway stopped")
}
