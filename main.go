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
	redisHost := getEnv("REDIS_HOST", "localhost")
	redisPort := getEnv("REDIS_PORT", "6379")
	redisPassword := getEnv("REDIS_PASSWORD", "")
	natsURL := getEnv("NATS_URL", nats.DefaultURL)
	appPort := getEnv("APP_PORT", "8080")

	redisAddr := redisHost + ":" + redisPort

	rdb := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: redisPassword,
		DB:       0,
	})
	ctx := context.Background()

	defer rdb.Close()

	nc, err := nats.Connect(natsURL)
	if err != nil {
		log.Fatalf("Failed to connect to NATS: %v", err)
	}
	defer nc.Close()

	if environment := os.Getenv("WORKER"); environment == "true" {
		log.Println("Starting as Worker...")
		worker := worker.NewMatchmakeWorker(nc, rdb)
		if err := worker.Start(); err != nil {
			log.Fatalf("Failed to start worker: %v", err)
		}
		<-ctx.Done()
		return
	}

	log.Println("Starting as API...")
	app := fiber.New()

	matchmakerHandler := handler.NewMatchmakeHandler(nc, rdb)

	app.Post("/matchmake", matchmakerHandler.Executer)
	app.Get("/healthz", func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	app.Get("/readyz", func(c *fiber.Ctx) error {
		if err := rdb.Ping(ctx).Err(); err != nil {
			log.Printf("Failed to connect to Redis: %v", err)
			return c.SendStatus(fiber.StatusServiceUnavailable)
		}

		if err := nc.IsConnected(); !err {
			log.Printf("Failed to connect to NATS: %v", err)
			return c.SendStatus(fiber.StatusServiceUnavailable)
		}

		return c.SendStatus(fiber.StatusOK)
	})

	log.Printf("API server starting on port %s", appPort)
	log.Fatal(app.Listen(":" + appPort))
}
