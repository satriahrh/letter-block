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
	BeginTransaction(context.Context) (*sql.Tx, error)
	FinalizeTransaction(*sql.Tx, error) error
	InsertGame(context.Context, *sql.Tx, Game) (Game, error)
	InsertGamePlayerBulk(context.Context, *sql.Tx, Game, []Player) (Game, error)
	GetPlayersByUsernames(context.Context, []string) ([]Player, error)
	GetGameById(context.Context, *sql.Tx, uint64) (Game, error)
	GetGamePlayerById(context.Context, uint64) (GamePlayer, error)
	GetGamePlayersByGameId(ctx context.Context, tx *sql.Tx, gameId uint64) ([]GamePlayer, error)
	LogPlayedWord(ctx context.Context, tx *sql.Tx, gameId, playerId uint64, word string) error
	UpdateGame(ctx context.Context, tx *sql.Tx, game Game) error
}

type Player struct {
	Id       uint64 `json:"id"`
	Username string `json:"username"`
}

type Game struct {
	Id               uint64   `json:"id"`
	CurrentOrder     uint8    `json:"current_order"`
	Players          []Player `json:"players"`
	MaxStrength      uint8    `json:"max_strength"`
	BoardBase        []uint8  `json:"board_base"`
	BoardPositioning []uint8  `json:"board_positioning"`
}

type GamePlayer struct {
	Id       uint64 `json:"id"`
	PlayerId uint64 `json:"player_id"`
	Ordering uint8  `json:"ordering"`
	GameId   uint64 `json:"game_id"`
}

type PlayedWord struct {
	PlayerId uint64 `json:"player_id"`
	Word     string `json:"word"`
}
