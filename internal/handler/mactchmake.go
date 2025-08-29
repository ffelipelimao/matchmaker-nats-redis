package handler

import (
	"log"
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

	log.Printf("[HANDLER] Received matchmaking request from IP: %s", c.IP())

	var req entities.MatchRequest
	if err := c.BodyParser(&req); err != nil {
		log.Printf("[HANDLER] Failed to parse request body: %v", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	log.Printf("[HANDLER] Request parsed successfully - Player ID: %s, Ping: %dms", req.Player.ID, req.Player.Ping)

	// Add player to FIFO pool in Redis (Sorted Set by timestamp)
	log.Printf("[HANDLER] Adding player %s to Redis pool with timestamp %d", req.Player.ID, time.Now().Unix())

	_, err := h.redisClient.ZAdd(ctx, playerPoolKey, &redis.Z{
		Score:  float64(time.Now().Unix()),
		Member: req.Player.ID,
	}).Result()
	if err != nil {
		log.Printf("[HANDLER] Failed to add player %s to Redis pool: %v", req.Player.ID, err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to add player to pool",
		})
	}

	log.Printf("[HANDLER] Player %s added to Redis pool successfully", req.Player.ID)

	// Get current pool size
	poolSize, err := h.redisClient.ZCard(ctx, playerPoolKey).Result()
	if err != nil {
		log.Printf("[HANDLER] Could not get pool size: %v", err)
	} else {
		log.Printf("[HANDLER] Current pool size: %d players", poolSize)
	}

	// Publish request to NATS for worker processing
	log.Printf("[HANDLER] Publishing matchmaking request to NATS subject: %s", natsSubject)

	reqData, err := req.ToJSON()
	if err != nil {
		log.Printf("[HANDLER] Failed to serialize request for NATS: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to serialize request",
		})
	}

	err = h.natsClient.Publish(natsSubject, reqData)
	if err != nil {
		log.Printf("[HANDLER] Failed to publish to NATS: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to publish matchmaking request",
		})
	}

	log.Printf("[HANDLER] Request published to NATS successfully")

	log.Printf("[HANDLER] Matchmaking request completed for player %s", req.Player.ID)

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message":   "Matchmaking request sent successfully",
		"player":    req.Player,
		"pool_size": poolSize,
	})
}
