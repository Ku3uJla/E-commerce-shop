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

	"github.com/google/uuid"
	"github.com/joho/godotenv"

	"order_service/internal/catalog"
	"order_service/internal/config"
	"order_service/internal/discovery"
	"order_service/internal/domain/order"
	"order_service/internal/messaging"
	repository "order_service/internal/repository/postgres"
	httptransport "order_service/internal/transport/http"
	"order_service/internal/transport/http/handler"
	kafkatransport "order_service/internal/transport/kafka"
)

func main() {
	_ = godotenv.Load()

	cfg := config.Load()
	instanceID := uuid.NewString()[:8]

	admin := cfg.DB
	admin.Name = "postgres"
	if err := repository.EnsureDatabase(admin.DSN(), cfg.DB.Name); err != nil {
		log.Fatalf("failed to ensure database %q: %v", cfg.DB.Name, err)
	}

	db, err := repository.Connect(cfg.DB.DSN())
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	if err := repository.AutoMigrate(db); err != nil {
		log.Fatalf("failed to migrate database: %v", err)
	}

	// Background workers (outbox relay, saga consumer) stop with this context.
	ctx, stopWorkers := context.WithCancel(context.Background())
	if err := messaging.EnsureTopics(ctx, cfg.KafkaBrokers, 3, order.TopicOrderEvents, order.TopicPaymentEvents); err != nil {
		log.Fatalf("failed to prepare kafka topics: %v", err)
	}
	publisher := messaging.NewPublisher(cfg.KafkaBrokers)
	defer publisher.Close()

	repo := repository.NewOrderRepo(db)
	svc := order.NewService(repo, catalog.NewClient(cfg.CatalogURL))

	go repository.NewOutboxRelay(db, publisher).Run(ctx)
	go messaging.Consume(ctx, cfg.KafkaBrokers, "order-service", order.TopicPaymentEvents, kafkatransport.PaymentEventsHandler(svc))

	h := handler.New(svc, instanceID)
	router := httptransport.NewRouter(h, instanceID)

	server := &nethttp.Server{
		Addr:         cfg.HTTPAddr,
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Printf("order service (instance %s) listening on %s", instanceID, cfg.HTTPAddr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, nethttp.ErrServerClosed) {
			log.Fatalf("http server error: %v", err)
		}
	}()

	consul := discovery.NewClient(cfg.ConsulAddr)
	serviceID := cfg.ServiceName + "-" + instanceID
	registerWithConsul(consul, cfg, serviceID, instanceID)

	<-stop
	stopWorkers()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("graceful shutdown failed: %v", err)
	}
	if err := consul.Deregister(serviceID); err != nil {
		log.Printf("failed to deregister from consul: %v", err)
	}
	log.Println("order service stopped")
}

func registerWithConsul(client *discovery.Client, cfg config.Config, serviceID, instanceID string) {
	reg := discovery.Registration{
		ID:              serviceID,
		Name:            cfg.ServiceName,
		Address:         discovery.LocalIP(),
		Port:            cfg.ServicePort,
		Tags:            []string{"orders", "http"},
		HealthCheckPath: "/healthz",
	}

	for attempt := 1; attempt <= 30; attempt++ {
		if err := client.Register(reg); err == nil {
			log.Printf("registered %s at %s:%d with consul", serviceID, reg.Address, reg.Port)
			return
		} else if attempt == 30 {
			log.Printf("could not register %s with consul: %v", serviceID, err)
			return
		}
		time.Sleep(2 * time.Second)
	}
}
