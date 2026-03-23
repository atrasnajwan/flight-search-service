package main

import (
	"context"
	"flight-search-service/internal/config"
	"flight-search-service/internal/domain"
	"flight-search-service/internal/flight"
	"flight-search-service/internal/provider/airasia"
	"flight-search-service/internal/provider/batik"
	"flight-search-service/internal/provider/garuda"
	"flight-search-service/internal/provider/lion"
	"flight-search-service/internal/redis"
	"flight-search-service/internal/repository/airport"
	"flight-search-service/internal/service/limiter"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	// Load configuration
	config.LoadConfig()

	// init gin router
	router := gin.New()
	router.Use(gin.Recovery()) // can recover from panics

	router.Use(gin.LoggerWithConfig(gin.LoggerConfig{
		SkipPaths: []string{"/healthz"},
	}))

	corsConfig := cors.Config{
		AllowMethods:    []string{"GET", "POST", "OPTIONS"},
		AllowHeaders:    []string{"Origin", "Content-Type"},
		ExposeHeaders:   []string{"Content-Length"},
		AllowAllOrigins: true, // change on production
	}
	router.Use(cors.New(corsConfig))

	// /healthz
	router.GET("/healthz", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	// redis
	redisClient, _ := redis.NewRedisClient()
	redisCache := redis.NewCache(redisClient)
	
	airportInstance := airport.NewInstance()
	err := airportInstance.LoadFromJSON("internal/repository/airport/airports.json")
	if err != nil {
		log.Fatalf("Failed to load airport mapping: %v", err)
	}

	garudaProvider := garuda.NewGarudaProvider("internal/provider/garuda/mock-response.json", 50, 100)                          // delay 50-100ms
	airasiaProvider := airasia.NewAirAsiaProvider("internal/provider/airasia/mock-response.json", airportInstance, 50, 150, 90) // delay 50-150ms, 90% success rate
	batikProvider := batik.NewBatikProvider("internal/provider/batik/mock-response.json", airportInstance, 200, 400)            // delay 200-400ms
	lionProvider := lion.NewLionProvider("internal/provider/lion/mock-response.json", 100, 200)                                 // delay 100-200ms

	garudaRated := limiter.NewRatedProvider(garudaProvider, 5, 10)
	airasiaRated := limiter.NewRatedProvider(airasiaProvider, 5, 10)
	batikRated := limiter.NewRatedProvider(batikProvider, 5, 10)
	lionRated := limiter.NewRatedProvider(lionProvider, 5, 10)

	providers := []domain.Provider{
		garudaRated,
		airasiaRated,
		batikRated,
		lionRated,
	}

	flightService := flight.NewService(providers, redisCache)
	flightHandler := flight.NewHandler(flightService)

	// /search
	router.POST("/search", flightHandler.Search)
	// /search/multi-city
	router.POST("/search/multi-city", flightHandler.SearchMultiCity)

	server := &http.Server{
		Addr:    fmt.Sprintf(":%s", config.AppConfig.ServerPort),
		Handler: router.Handler(),
	}

	// Start HTTP server
	go func() {
		log.Printf("HTTP server listening on port %s\n", config.AppConfig.ServerPort)
		err := server.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server failed to start: %v\n", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit // block until signal received
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("HTTP server shutdown error: %v\n", err)
	}
	log.Println("Server shutdown complete")
}
