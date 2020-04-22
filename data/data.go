package data

import (
	"context"
	"database/sql"
)

// Dictionary should store value in cache db
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
	GetGameByID(context.Context, *sql.Tx, uint64) (Game, error)
	GetGamePlayerByID(context.Context, uint64) (GamePlayer, error)
	GetGamePlayersByGameID(ctx context.Context, tx *sql.Tx, gameID uint64) ([]GamePlayer, error)
	LogPlayedWord(ctx context.Context, tx *sql.Tx, gameID, playerID uint64, word string) error
	UpdateGame(ctx context.Context, tx *sql.Tx, game Game) error
}

// Player define player
type Player struct {
	ID       uint64 `json:"id"`
	Username string `json:"username"`
}

// Game define game
type Game struct {
	ID               uint64    `json:"id"`
	CurrentOrder     uint8     `json:"current_order"`
	Players          []Player  `json:"players"`
	State            GameState `json:"state"`
	MaxStrength      uint8     `json:"max_strength"`
	BoardBase        []uint8   `json:"board_base"`
	BoardPositioning []uint8   `json:"board_positioning"`
}

// GameState define Game's state
type GameState uint8

const (
	// CREATED game is created
	CREATED GameState = iota

	// ONGOING game is on going
	ONGOING GameState = iota

	// END game is ended
	END GameState = iota
)

// GamePlayer list player participating in a game
type GamePlayer struct {
	ID       uint64 `json:"id"`
	PlayerID uint64 `json:"player_id"`
	Ordering uint8  `json:"ordering"`
	GameID   uint64 `json:"game_id"`
}

// PlayedWord word played in a game
type PlayedWord struct {
	PlayerID uint64 `json:"player_id"`
	Word     string `json:"word"`
}
