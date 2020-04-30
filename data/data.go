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
	InsertGamePlayer(context.Context, *sql.Tx, Game, Player) (Game, error)
	GetPlayerById(context.Context, PlayerId) (Player, error)
	GetGameById(context.Context, *sql.Tx, GameId) (Game, error)
	GetGamePlayersByGameId(context.Context, *sql.Tx, GameId) ([]GamePlayer, error)
	LogPlayedWord(context.Context, *sql.Tx, GameId, PlayerId, string) error
	UpdateGame(context.Context, *sql.Tx, Game) error
}

type PlayerId uint64
type GameId uint64
type GamePlayerId uint64

type Player struct {
	Id       PlayerId `json:"id"`
	Username string   `json:"username"`
}

type Game struct {
	Id                 GameId    `json:"id"`
	CurrentPlayerOrder uint8     `json:"current_player_order"` // zero based
	NumberOfPlayer     uint8     `json:"number_of_player"`
	Players            []Player  `json:"players"`
	State              GameState `json:"state"`
	BoardBase          []uint8   `json:"board_base"`
	BoardPositioning   []uint8   `json:"board_positioning"`
}

type GameState uint8

const (
	CREATED GameState = iota
	ONGOING GameState = iota
	END     GameState = iota
)

type GamePlayer struct {
	Id       GamePlayerId `json:"id"`
	PlayerId PlayerId     `json:"player_id"`
	Ordering uint8        `json:"ordering"`
	GameId   GameId       `json:"game_id"`
}

type PlayedWord struct {
	PlayerId GamePlayerId `json:"player_id"`
	Word     string       `json:"word"`
}
