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

const (
	redisAddress = "localhost:6379"
	natsAddress  = nats.DefaultURL
)

func main() {
	rdb := redis.NewClient(&redis.Options{
		Addr: redisAddress,
	})
	ctx := context.Background()

	defer rdb.Close()

	nc, err := nats.Connect(natsAddress)
	if err != nil {
		log.Fatalf("Falha ao conectar ao NATS: %v", err)
	}
	defer nc.Close()

	if environment := os.Getenv("WORKER"); environment == "true" {
		worker := worker.NewMatchmakeWorker(nc, rdb)
		if err := worker.Start(); err != nil {
			log.Fatalf("Falha ao iniciar o worker: %v", err)
		}
		<-ctx.Done()
		return
	}
	app := fiber.New()

	matchmakerHandler := handler.NewMatchmakeHandler(nc, rdb)

	app.Post("/matchmake", matchmakerHandler.Executer)

	log.Fatal(app.Listen(":3000"))
}
