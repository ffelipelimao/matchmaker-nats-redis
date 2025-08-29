package worker

import (
	"context"
	"log"
	"time"

	"matchmaker-nats/internal/entities"

	"github.com/go-redis/redis/v8"
	"github.com/nats-io/nats.go"
)

const (
	MatchmakeQueue = "matchmake"
	playerPoolKey  = "player_pool"
	natsSubject    = "matchmake.request"
	MinPlayers     = 2
	MaxPlayers     = 16
	BatchSize      = 50
)

type MatchmakeWorker struct {
	natsClient  *nats.Conn
	redisClient *redis.Client
}

func NewMatchmakeWorker(natsClient *nats.Conn, redisClient *redis.Client) *MatchmakeWorker {
	log.Printf("[WORKER] Initializing MatchmakeWorker")
	return &MatchmakeWorker{
		natsClient:  natsClient,
		redisClient: redisClient,
	}
}

func (mw *MatchmakeWorker) Start() error {
	log.Printf("[WORKER] Starting worker with queue: %s", MatchmakeQueue)

	_, err := mw.natsClient.QueueSubscribe(natsSubject, MatchmakeQueue, func(msg *nats.Msg) {
		log.Printf("[WORKER] Received NATS message, starting matchmaking process")
		mw.processPlayerBatches()
		msg.Ack()
		log.Printf("[WORKER] NATS message acknowledged")
	})
	if err != nil {
		log.Printf("[WORKER] Failed to subscribe to NATS queue: %v", err)
		return err
	}

	log.Printf("[WORKER] Successfully subscribed to NATS subject: %s", natsSubject)
	log.Printf("[WORKER] Worker is now listening for messages...")

	return err
}

func (mw *MatchmakeWorker) processPlayerBatches() {
	log.Printf("[WORKER] Starting player batch processing")
	ctx := context.Background()

	batchCount := 0
	totalPlayersProcessed := 0

	for {
		batchCount++
		log.Printf("[WORKER] Processing batch #%d", batchCount)

		result, err := mw.redisClient.ZRangeWithScores(ctx, playerPoolKey, 0, BatchSize-1).Result()
		if err != nil {
			log.Printf("[WORKER] Error getting player batch #%d: %v", batchCount, err)
			return
		}

		if len(result) < MinPlayers {
			log.Printf("[WORKER] Batch #%d has insufficient players: %d (minimum: %d)", batchCount, len(result), MinPlayers)
			break
		}

		log.Printf("[WORKER] Batch #%d contains %d players", batchCount, len(result))

		matches := mw.processBatch(ctx, result)
		totalPlayersProcessed += len(result)

		log.Printf("[WORKER] Batch #%d created %d matches", batchCount, len(matches))

		for _, match := range matches {
			log.Printf("[WORKER] Created match %s with %d players", match.MatchID, len(match.Players))
		}

		if len(result) < BatchSize {
			log.Printf("[WORKER] Reached end of player pool, stopping batch processing")
			break
		}
	}

	log.Printf("[WORKER] Batch processing completed - Total batches: %d, Total players processed: %d", batchCount, totalPlayersProcessed)
}

func (mw *MatchmakeWorker) processBatch(ctx context.Context, players []redis.Z) []entities.Match {
	log.Printf("[WORKER] Processing batch of %d players", len(players))

	playerEntities := make([]entities.Player, 0, len(players))
	for _, z := range players {
		playerID := z.Member.(string)
		score := z.Score
		player := entities.Player{
			ID:   playerID,
			Ping: 0,
		}
		playerEntities = append(playerEntities, player)
		log.Printf("[WORKER] Player %s (score: %.0f) added to batch", playerID, score)
	}

	log.Printf("[WORKER] Creating optimal matches from %d players", len(playerEntities))
	matches := mw.createOptimalMatches(playerEntities)

	playerIDs := make([]interface{}, len(playerEntities))
	for i, player := range playerEntities {
		playerIDs[i] = player.ID
	}

	log.Printf("[WORKER] Removing %d matched players from Redis pool", len(playerIDs))
	mw.redisClient.ZRem(ctx, playerPoolKey, playerIDs...)

	return matches
}

func (mw *MatchmakeWorker) createOptimalMatches(players []entities.Player) []entities.Match {
	log.Printf("[WORKER] Creating optimal matches for %d players", len(players))

	var matches []entities.Match
	remainingPlayers := players

	for len(remainingPlayers) >= MinPlayers {
		matchSize := mw.calculateOptimalMatchSize(len(remainingPlayers))
		log.Printf("[WORKER] Optimal match size for %d remaining players: %d", len(remainingPlayers), matchSize)

		matchPlayers := remainingPlayers[:matchSize]
		remainingPlayers = remainingPlayers[matchSize:]

		match := entities.Match{
			MatchID:   generateMatchID(),
			Players:   matchPlayers,
			CreatedAt: time.Now(),
		}
		matches = append(matches, match)

		log.Printf("[WORKER] Match %s created with %d players", match.MatchID, len(matchPlayers))
	}

	log.Printf("[WORKER] Created %d total matches", len(matches))
	return matches
}

func (mw *MatchmakeWorker) calculateOptimalMatchSize(totalPlayers int) int {
	if totalPlayers <= MaxPlayers {
		log.Printf("[WORKER] Using all %d players (within max limit)", totalPlayers)
		return totalPlayers
	}

	if totalPlayers >= 24 {
		log.Printf("[WORKER] Large group (%d players) - creating match of 12", totalPlayers)
		return 12
	} else if totalPlayers >= 18 {
		log.Printf("[WORKER] Medium group (%d players) - creating match of 9", totalPlayers)
		return 9
	} else {
		log.Printf("[WORKER] Small group (%d players) - using all players", totalPlayers)
		return totalPlayers
	}
}

func generateMatchID() string {
	return "match_" + time.Now().Format("20060102150405")
}
