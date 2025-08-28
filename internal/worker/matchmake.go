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
		// Get the 100 oldest players in the pool
		ctx := context.Background()
		result, err := mw.redisClient.ZRangeWithScores(ctx, playerPoolKey, 0, 99).Result()
		if err != nil {
			log.Printf("Error getting players from pool: %v", err)
			return
		}

		if len(result) < 2 {
			// Not enough players for a match
			return
		}

		// Create a match with available players
		players := make([]entities.Player, 0, len(result))
		for _, z := range result {
			playerID := z.Member.(string)
			player := entities.Player{
				ID:   playerID,
				Ping: 0,
			}
			players = append(players, player)
		}

		match := entities.Match{
			MatchID:   generateMatchID(),
			Players:   players,
			CreatedAt: time.Now(),
		}

		playerIDs := make([]interface{}, len(players))
		for i, player := range players {
			playerIDs[i] = player.ID
		}

		// Remove players from the pool
		mw.redisClient.ZRem(ctx, playerPoolKey, playerIDs...)

		log.Printf("Created match %s with %d players", match.MatchID, len(match.Players))

		msg.Ack()
	})

	return err
}

func generateMatchID() string {
	return "match_" + time.Now().Format("20060102150405")
}
