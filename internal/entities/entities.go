package entities

import (
	"encoding/json"
	"time"

	"matchmaker-nats/pkg/protos/gen"
)

type Player struct {
	ID   string `json:"id"`
	Ping int    `json:"ping"`
}

type MatchRequest struct {
	Player Player `json:"player"`
}

type Match struct {
	MatchID   string    `json:"match_id"`
	Players   []Player  `json:"players"`
	CreatedAt time.Time `json:"created_at"`
}

// Player serialization methods
func (p *Player) ToProto() *gen.Player {
	return &gen.Player{
		Id:   p.ID,
		Ping: int32(p.Ping),
	}
}

func (p *Player) FromProto(proto *gen.Player) {
	p.ID = proto.Id
	p.Ping = int(proto.Ping)
}

func (p *Player) ToJSON() ([]byte, error) {
	return json.Marshal(p)
}

func (p *Player) FromJSON(data []byte) error {
	return json.Unmarshal(data, p)
}

// MatchRequest serialization methods
func (mr *MatchRequest) ToProto() *gen.MatchRequest {
	return &gen.MatchRequest{
		Player: mr.Player.ToProto(),
	}
}

func (mr *MatchRequest) FromProto(proto *gen.MatchRequest) {
	mr.Player.FromProto(proto.Player)
}

func (mr *MatchRequest) ToJSON() ([]byte, error) {
	return json.Marshal(mr)
}

func (mr *MatchRequest) FromJSON(data []byte) error {
	return json.Unmarshal(data, mr)
}

// Match serialization methods
func (m *Match) ToProto() *gen.Match {
	players := make([]*gen.Player, len(m.Players))
	for i, player := range m.Players {
		players[i] = player.ToProto()
	}

	return &gen.Match{
		MatchId:   m.MatchID,
		Players:   players,
		CreatedAt: m.CreatedAt.Unix(),
	}
}

func (m *Match) FromProto(proto *gen.Match) {
	m.MatchID = proto.MatchId
	m.Players = make([]Player, len(proto.Players))
	for i, playerProto := range proto.Players {
		m.Players[i].FromProto(playerProto)
	}
	m.CreatedAt = time.Unix(proto.CreatedAt, 0)
}

func (m *Match) ToJSON() ([]byte, error) {
	return json.Marshal(m)
}

func (m *Match) FromJSON(data []byte) error {
	return json.Unmarshal(data, m)
}
