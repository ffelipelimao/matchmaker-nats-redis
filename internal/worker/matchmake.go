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
	MinPlayers     = 2
	MaxPlayers     = 16
	BatchSize      = 50 // Process players in batches of 50
)

type MatchmakeWorker struct {
	natsClient  *nats.Conn
	redisClient *redis.Client
}

func NewMatchmakeWorker(natsClient *nats.Conn, redisClient *redis.Client) *MatchmakeWorker {
	return &MatchmakeWorker{
		natsClient:  natsClient,
		redisClient: redisClient,
	}
}

func (mw *MatchmakeWorker) Start() error {
	_, err := mw.natsClient.QueueSubscribe(MatchmakeQueue, MatchmakeQueue, func(msg *nats.Msg) {
		mw.processPlayerBatches()
		msg.Ack()
	})

	return err
}

func (mw *MatchmakeWorker) processPlayerBatches() {
	ctx := context.Background()

	for {
		result, err := mw.redisClient.ZRangeWithScores(ctx, playerPoolKey, 0, BatchSize-1).Result()
		if err != nil {
			log.Printf("Error getting player batch: %v", err)
			return
		}

		if len(result) < MinPlayers {
			log.Printf("Not enough players in batch: %d (minimum: %d)", len(result), MinPlayers)
			return
		}

		matches := mw.processBatch(ctx, result)

		for _, match := range matches {
			log.Printf("Created match %s with %d players", match.MatchID, len(match.Players))
		}

		if len(result) < BatchSize {
			break
		}
	}
}

func (mw *MatchmakeWorker) processBatch(ctx context.Context, players []redis.Z) []entities.Match {
	// Convert Redis results to Player entities
	playerEntities := make([]entities.Player, 0, len(players))
	for _, z := range players {
		playerID := z.Member.(string)
		player := entities.Player{
			ID:   playerID,
			Ping: 0,
		}
		playerEntities = append(playerEntities, player)
	}

	matches := mw.createOptimalMatches(playerEntities)

	playerIDs := make([]interface{}, len(playerEntities))
	for i, player := range playerEntities {
		playerIDs[i] = player.ID
	}
	mw.redisClient.ZRem(ctx, playerPoolKey, playerIDs...)

	return matches
}

func (mw *MatchmakeWorker) createOptimalMatches(players []entities.Player) []entities.Match {
	var matches []entities.Match
	remainingPlayers := players

	for len(remainingPlayers) >= MinPlayers {
		matchSize := mw.calculateOptimalMatchSize(len(remainingPlayers))

		matchPlayers := remainingPlayers[:matchSize]
		remainingPlayers = remainingPlayers[matchSize:]

		match := entities.Match{
			MatchID:   generateMatchID(),
			Players:   matchPlayers,
			CreatedAt: time.Now(),
		}
		matches = append(matches, match)
	}

	return matches
}

func (mw *MatchmakeWorker) calculateOptimalMatchSize(totalPlayers int) int {
	if totalPlayers <= MaxPlayers {
		return totalPlayers
	}

	if totalPlayers >= 24 {
		return 12
	} else if totalPlayers >= 18 {
		return 9
	} else {
		return totalPlayers
	}
}

func generateMatchID() string {
	return "match_" + time.Now().Format("20060102150405")
}
