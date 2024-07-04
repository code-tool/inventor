package main

import (
	"context"
	"fmt"
	"inventor/src/config"
	"inventor/src/handler"
	"inventor/src/metrics"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"

	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/go-redis/redis/v9"
)

var (
	server         *http.Server
	shutdownWaiter sync.WaitGroup
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

	shutdownWaiter.Add(1)
	configureServer()
	initSignalHandler()
	initRouting()
	startServer()
	shutdownWaiter.Wait()
	log.Println("Exiting server")
}

func configureServer() {
	log.Printf("Creating server on port %s.", config.GetConfig().LISTEN_PORT)
	server = &http.Server{
		Addr:    fmt.Sprintf(":%s", config.GetConfig().LISTEN_PORT),
		Handler: nil,
	}
	server.RegisterOnShutdown(func() {
		log.Println("Shutting down server.")
		handler.FlushBufferOnShutdown(&shutdownWaiter)
	})
}

func initSignalHandler() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGINT, syscall.SIGKILL)
	go handleSignal(server, sigChan)
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
	if err := server.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatalf("HTTP server ListenAndServe: %v.", err)
		shutdownWaiter.Done()
	}
}

func handleSignal(server *http.Server, signals <-chan os.Signal) {
	log.Println("Listening OS signals.")
	for sig := range signals {
		log.Printf("OS signal received [%s].", sig.String())
		if err := server.Shutdown(context.Background()); err != nil {
			log.Fatalf("HTTP server Shutdown: %v.", err)
		}
	}
}
