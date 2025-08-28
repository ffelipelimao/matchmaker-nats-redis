package handler

import (
	"time"

	"matchmaker-nats/internal/entities"

	"github.com/go-redis/redis/v8"
	"github.com/gofiber/fiber/v2"
	"github.com/nats-io/nats.go"
)

const (
	playerPoolKey = "player_pool"
	natsSubject   = "matchmake.request"
)

type matchmakeHandler struct {
	natsClient  *nats.Conn
	redisClient *redis.Client
}

func NewMatchmakeHandler(natsClient *nats.Conn, redisClient *redis.Client) *matchmakeHandler {
	return &matchmakeHandler{
		natsClient:  natsClient,
		redisClient: redisClient,
	}
}

func (h *matchmakeHandler) Executer(c *fiber.Ctx) error {
	ctx := c.Context()
	var req entities.MatchRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Add player to FIFO pool in Redis (Sorted Set)
	_, err := h.redisClient.ZAdd(ctx, playerPoolKey, &redis.Z{
		Score:  float64(time.Now().Unix()),
		Member: req.Player.ID,
	}).Result()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to add player to pool",
		})
	}

	// Publish request to NATS for worker processing
	reqData, err := req.ToJSON()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to serialize request",
		})
	}

	err = h.natsClient.Publish(natsSubject, reqData)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to publish matchmaking request",
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Matchmaking request sent successfully",
		"player":  req.Player,
	})
}
