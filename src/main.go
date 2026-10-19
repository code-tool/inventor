package main

import (
	"context"
	"errors"
	"fmt"
	"inventor/src/config"
	"inventor/src/handler"
	"inventor/src/metrics"
	"log"
	"net/http"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/redis/go-redis/v9"
)

// shutdownTimeout bounds how long in-flight requests may drain after a
// termination signal. Kept below Kubernetes' default 30s grace period so the
// process exits on its own before it gets SIGKILLed.
const shutdownTimeout = 25 * time.Second

var (
	server         *http.Server
	registerTarget *handler.SDTargetsMiddleware
	buildVersion   string
)

func main() {

	ctx := context.Background()

	db, err := strconv.Atoi(config.GetConfig().REDIS_DBNO)
	if err != nil {
		log.Fatalf("Can't configure REDIS_DBNO: %v", err)
	}
	con := redis.NewClient(&redis.Options{
		Addr: fmt.Sprintf("%s:%s", config.GetConfig().REDIS_ADDR, config.GetConfig().REDIS_PORT),
		DB:   db,
	})

	ttlSeconds, err := strconv.Atoi(config.GetConfig().TTL_SECONDS)
	if err != nil {
		log.Fatalf("Can't configure TTL_SECONDS: %v", err)
	}
	registerTarget = &handler.SDTargetsMiddleware{
		SDTargets: &handler.SDTargets{
			Items: make(map[uuid.UUID]handler.StaticConfig),
		},
		Context:  ctx,
		Client:   con,
		ApiToken: config.GetConfig().API_TOKEN,
		SdToken:  config.GetConfig().SD_TOKEN,
		TTL:      ttlSeconds,
	}

	TargetCollector := metrics.SDTargetsCollector{
		TargetInfo: registerTarget,
	}

	prometheus.MustRegister(TargetCollector)

	sigCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGINT)
	defer stop()

	configureServer()
	initRouting()
	go startServer()

	<-sigCtx.Done()
	// Restore default signal handling: a second signal kills the process
	// immediately instead of waiting for the drain to finish.
	stop()
	log.Println("OS signal received, shutting down server.")
	shutdownServer()
	if err := con.Close(); err != nil {
		log.Printf("Redis client Close: %v.", err)
	}
	log.Println("Exiting server")
}

func configureServer() {
	log.Printf("Creating server on port %s.", config.GetConfig().LISTEN_PORT)
	server = &http.Server{
		Addr:         fmt.Sprintf(":%s", config.GetConfig().LISTEN_PORT),
		Handler:      nil,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
}

func initRouting() {
	http.Handle("/metrics", promhttp.Handler())
	http.HandleFunc("/target", registerTarget.HandleSDTarget)
	http.HandleFunc("/group", registerTarget.HandleDiscoverGroup)
	http.HandleFunc("/discover", registerTarget.HandleDiscover)
	http.HandleFunc("/healthcheck", registerTarget.HealthCheck)
}

func startServer() {
	log.Printf("Starting http server. Build [%s]", buildVersion)
	if err := server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("HTTP server ListenAndServe: %v.", err)
	}
}

// shutdownServer stops accepting new connections and blocks until in-flight
// requests finish or shutdownTimeout expires, then force-closes the rest.
func shutdownServer() {
	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Printf("HTTP server Shutdown: %v, forcing close.", err)
		if err := server.Close(); err != nil {
			log.Printf("HTTP server Close: %v.", err)
		}
	}
}
