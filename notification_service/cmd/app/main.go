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

	"notification_service/internal/config"
	"notification_service/internal/discovery"
	"notification_service/internal/messaging"
	"notification_service/internal/notification"
	httptransport "notification_service/internal/transport/http"
)

func main() {
	_ = godotenv.Load()

	cfg := config.Load()
	instanceID := uuid.NewString()[:8]

	ctx, stopWorkers := context.WithCancel(context.Background())
	if err := messaging.EnsureTopics(ctx, cfg.KafkaBrokers, 3, notification.TopicOrderEvents, notification.TopicPaymentEvents); err != nil {
		log.Fatalf("failed to prepare kafka topics: %v", err)
	}

	svc := notification.NewService(notification.LogSender{})
	// One consumer group per topic; both feed the same service.
	go messaging.Consume(ctx, cfg.KafkaBrokers, "notification-service.orders", notification.TopicOrderEvents, svc.Handle)
	go messaging.Consume(ctx, cfg.KafkaBrokers, "notification-service.payments", notification.TopicPaymentEvents, svc.Handle)

	server := &nethttp.Server{
		Addr:         cfg.HTTPAddr,
		Handler:      httptransport.NewRouter(svc, instanceID),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Printf("notification service (instance %s) listening on %s", instanceID, cfg.HTTPAddr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, nethttp.ErrServerClosed) {
			log.Fatalf("http server error: %v", err)
		}
	}()

	consul := discovery.NewClient(cfg.ConsulAddr)
	serviceID := cfg.ServiceName + "-" + instanceID
	registerWithConsul(consul, cfg, serviceID)

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
	log.Println("notification service stopped")
}

func registerWithConsul(client *discovery.Client, cfg config.Config, serviceID string) {
	reg := discovery.Registration{
		ID:              serviceID,
		Name:            cfg.ServiceName,
		Address:         discovery.LocalIP(),
		Port:            cfg.ServicePort,
		Tags:            []string{"notifications", "http"},
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
