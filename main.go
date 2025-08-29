package main

import (
	"context"
	"log"
	"os"

	"matchmaker-nats/internal/handler"
	"matchmaker-nats/internal/worker"

	"github.com/go-redis/redis/v8"
	"github.com/gofiber/fiber/v2"
	"github.com/nats-io/nats.go"
)

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func main() {
	log.Printf("[MAIN] Starting Matchmaker application...")

	// Get configuration from environment variables
	redisHost := getEnv("REDIS_HOST", "localhost")
	redisPort := getEnv("REDIS_PORT", "6379")
	redisPassword := getEnv("REDIS_PASSWORD", "")
	natsURL := getEnv("NATS_URL", nats.DefaultURL)
	appPort := getEnv("APP_PORT", "8080")

	log.Printf("[MAIN] Configuration loaded - Redis: %s:%s, NATS: %s, Port: %s", redisHost, redisPort, natsURL, appPort)

	redisAddr := redisHost + ":" + redisPort

	log.Printf("[MAIN] Connecting to Redis at %s", redisAddr)

	// Create Redis client
	rdb := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: redisPassword,
		DB:       0,
	})
	ctx := context.Background()

	defer func() {
		log.Printf("[MAIN] Closing Redis connection")
		rdb.Close()
	}()

	// Test Redis connection
	log.Printf("[MAIN] Testing Redis connection...")
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatalf("[MAIN] Failed to connect to Redis: %v", err)
	}
	log.Printf("[MAIN] Redis connection successful")

	// Connect to NATS
	log.Printf("[MAIN] Connecting to NATS at %s", natsURL)
	nc, err := nats.Connect(natsURL)
	if err != nil {
		log.Fatalf("[MAIN] Failed to connect to NATS: %v", err)
	}
	defer func() {
		log.Printf("[MAIN] Closing NATS connection")
		nc.Close()
	}()

	// Check NATS connection
	if nc.IsConnected() {
		log.Printf("[MAIN] NATS connection successful")
	} else {
		log.Fatalf("[MAIN] NATS connection failed")
	}

	// Check if we should run as worker
	if environment := os.Getenv("WORKER"); environment == "true" {
		log.Printf("[MAIN] Starting as Worker...")
		worker := worker.NewMatchmakeWorker(nc, rdb)
		if err := worker.Start(); err != nil {
			log.Fatalf("[MAIN] Failed to start worker: %v", err)
		}
		log.Printf("[MAIN] Worker started successfully, waiting for messages...")
		<-ctx.Done()
		return
	}

	// Run as API
	log.Printf("[MAIN] Starting as API...")
	app := fiber.New()

	log.Printf("[MAIN] Initializing Fiber app...")
	matchmakerHandler := handler.NewMatchmakeHandler(nc, rdb)

	log.Printf("[MAIN] Setting up API routes...")
	app.Post("/matchmake", matchmakerHandler.Executer)
	app.Get("/healthz", func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	app.Get("/readyz", func(c *fiber.Ctx) error {
		if err := rdb.Ping(ctx).Err(); err != nil {
			log.Printf("[MAIN] Health check failed - Redis: %v", err)
			return c.SendStatus(fiber.StatusServiceUnavailable)
		}

		if err := nc.IsConnected(); !err {
			log.Printf("[MAIN] Health check failed - NATS: %v", err)
			return c.SendStatus(fiber.StatusServiceUnavailable)
		}

		return c.SendStatus(fiber.StatusOK)
	})

	log.Printf("[MAIN] API routes configured successfully")
	log.Printf("[MAIN] Starting HTTP server on port %s", appPort)
	log.Printf("[MAIN] Health check available at /healthz")
	log.Printf("[MAIN] Readiness check available at /readyz")
	log.Printf("[MAIN] Matchmaking endpoint available at POST /matchmake")

	log.Fatal(app.Listen(":" + appPort))
}
