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

	"auth_service/internal/config"
	"auth_service/internal/discovery"
	"auth_service/internal/domain/user"
	"auth_service/internal/features"
	repository "auth_service/internal/repository/postgres"
	httptransport "auth_service/internal/transport/http"
	"auth_service/internal/transport/http/handler"
)

func main() {
	_ = godotenv.Load()

	cfg := config.Load()
	instanceID := uuid.NewString()[:8]

	db, err := repository.Connect(cfg.DB.DSN())
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	if err := repository.AutoMigrate(db); err != nil {
		log.Fatalf("failed to migrate database: %v", err)
	}

	repo := repository.NewUserRepo(db)
	hasher := features.NewBcryptHasher()
	tokens := features.NewJWTService(cfg.JWTKey, cfg.JWTExpiry)

	userService := user.NewUserSrv(repo, hasher, tokens)
	h := handler.New(userService)
	router := httptransport.NewRouter(h, tokens)

	server := &nethttp.Server{
		Addr:         cfg.HTTPAddr,
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	go func() {
		log.Printf("auth service (instance %s) listening on %s", instanceID, cfg.HTTPAddr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, nethttp.ErrServerClosed) {
			log.Fatalf("http server error: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	consul := discovery.NewClient(cfg.ConsulAddr)
	serviceID := cfg.ServiceName + "-" + instanceID
	registerWithConsul(consul, cfg, serviceID)

	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Printf("graceful shutdown failed: %v", err)
	}
	if err := consul.Deregister(serviceID); err != nil {
		log.Printf("failed to deregister from consul: %v", err)
	}
	log.Println("auth service stopped")
}

func registerWithConsul(client *discovery.Client, cfg config.Config, serviceID string) {
	reg := discovery.Registration{
		ID:              serviceID,
		Name:            cfg.ServiceName,
		Address:         discovery.LocalIP(),
		Port:            cfg.ServicePort,
		Tags:            []string{"auth", "http"},
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
