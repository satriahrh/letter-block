package data

import (
	"context"
	"database/sql"
)

type Dictionary interface {
	//generateKey(lang, key string) string
	Get(lang, key string) (resut bool, exist bool)
	Set(lang, key string, value bool)
}

// Transactional should satisfying consistency and availability from CAP
type Transactional interface {
	BeginTransaction(context.Context, *sql.TxOptions) (*sql.Tx, error)
	FinalizeTransaction(*sql.Tx, error) error
	Transaction(context.Context, *sql.TxOptions, func(*sql.Tx) error) error
	InsertGame(context.Context, Game) (Game, error)
	GetPlayerByUsername(context.Context, string) (Player, error)
	GetPlayersByUsernames(context.Context, []string) ([]Player, error)
	GetGameByID(context.Context, *sql.Tx, uint64) (Game, error)
	GetGamePlayerByID(context.Context, uint64) (uint64, uint64, error)
}

type Player struct {
	ID       uint64 `json:"id"`
	Username string `json:"username"`
}

type Game struct {
	ID               uint64   `json:"id"`
	CurrentPlayerID  uint64   `json:"current_player_id"`
	Players          []Player `json:"players"`
	MaxStrength      uint8    `json:"max_strength"`
	BoardBase        []uint8  `json:"board_base"`
	BoardPositioning []uint8  `json:"board_positioning"`
}
