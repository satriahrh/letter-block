package service

import (
	"context"
	"errors"

	"github.com/satriahrh/letter-block/data"
	"github.com/satriahrh/letter-block/dictionary"
)

var (
	ErrorDoesntMakeWord   = errors.New("doesn't make word")
	ErrorGameIsUnplayable = errors.New("game is unplayable")
	ErrorPlayerIsEnough   = errors.New("player is enough")
	ErrorNotYourTurn      = errors.New("not your turn")
	ErrorNumberOfPlayer   = errors.New("number of player invalid")
	ErrorUnauthorized     = errors.New("player is not authorized")
	ErrorWordHavePlayed   = errors.New("word have played")
	ErrorWordInvalid      = errors.New("word invalid")
)

const (
	alphabet    = "abcdefghijklmnopqrstuvwxyz"
	maxStrength = 2
)

type Service interface {
	NewGame(ctx context.Context, firstPlayerId data.PlayerId, numberOfPlayer uint8) (data.Game, error)
	TakeTurn(ctx context.Context, gameId data.GameId, playerId data.PlayerId, word []uint8) (data.Game, error)
	JoinGame(ctx context.Context, gameId data.GameId, playerId data.PlayerId) (data.Game, error)
	GetGames(ctx context.Context, playerId data.PlayerId) ([]data.Game, error)
	GetGame(ctx context.Context, gameId data.GameId) (game data.Game, err error)
	GetPlayer(ctx context.Context, playerId data.PlayerId) (player data.Player, err error)
}

type application struct {
	transactional data.Transactional
	dictionaries  map[string]dictionary.Dictionary
}

func NewService(transactional data.Transactional, dictionaries map[string]dictionary.Dictionary) Service {
	return &application{
		transactional: transactional,
		dictionaries:  dictionaries,
	}
}
