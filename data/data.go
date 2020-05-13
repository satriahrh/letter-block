package data

import (
	"context"
	"database/sql"
)

type Dictionary interface {
	// generateKey(lang, key string) string
	Get(lang, key string) (result bool, exist bool)
	Set(lang, key string, value bool)
}

// Transactional should satisfying consistency and availability from CAP
type Transactional interface {
	BeginTransaction(context.Context) (*sql.Tx, error)
	FinalizeTransaction(*sql.Tx, error) error
	InsertGame(context.Context, *sql.Tx, Game) (Game, error)
	InsertGamePlayer(context.Context, *sql.Tx, Game, Player) (Game, error)
	GetPlayerById(context.Context, PlayerId) (Player, error)
	GetPlayersByGameId(context.Context, GameId) ([]Player, error)
	GetGameById(context.Context, *sql.Tx, GameId) (Game, error)
	GetGamePlayersByGameId(context.Context, *sql.Tx, GameId) ([]GamePlayer, error)
	GetGamesByPlayerId(context.Context, PlayerId) ([]Game, error)
	LogPlayedWord(context.Context, *sql.Tx, GameId, PlayerId, string) error
	GetPlayedWordsByGameId(context.Context, GameId) ([]PlayedWord, error)
	UpdateGame(context.Context, *sql.Tx, Game) error
	UpdatePlayer(context.Context, *sql.Tx, Player) error
	GetSetPlayerByDeviceFingerprint(context.Context, *sql.Tx, DeviceFingerprint) (Player, error)
}

type PlayerId uint64
type GameId uint64
type GamePlayerId uint64
type DeviceFingerprint string

type Player struct {
	Id                PlayerId          `json:"id"`
	DeviceFingerprint DeviceFingerprint `json:"device_fingerprint"`
	SessionExpiredAt  int64             `json:"session_expired_at"`
}

type Game struct {
	Id                 GameId       `json:"id"`
	CurrentPlayerOrder uint8        `json:"current_player_order"` // zero based
	NumberOfPlayer     uint8        `json:"number_of_player"`
	Players            []Player     `json:"players"`
	PlayedWords        []PlayedWord `json:"played_words"`
	State              GameState    `json:"state"`
	BoardBase          []uint8      `json:"board_base"`
	BoardPositioning   []uint8      `json:"board_positioning"`
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
	GameId   GameId       `json:"game_id"`
}

type PlayedWord struct {
	PlayerId PlayerId `json:"player_id"`
	Word     string   `json:"word"`
}
