package service_test

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/satriahrh/letter-block/data"
	"github.com/stretchr/testify/mock"
)

var (
	gameId = data.GameId(time.Now().UnixNano())

	players = []data.Player{
		{Id: data.PlayerId(time.Now().UnixNano())},
		{Id: data.PlayerId(time.Now().UnixNano())},
	}

	gamePlayers = []data.GamePlayer{
		{Id: data.GamePlayerId(time.Now().UnixNano()), PlayerId: players[0].Id, GameId: gameId},
		{Id: data.GamePlayerId(time.Now().UnixNano()), PlayerId: players[1].Id, GameId: gameId},
	}

	numberOfPlayer = uint8(5)

	playerId = players[0].Id

	gamePlayerId = gamePlayers[0].Id

	word      = []uint8{0, 1, 2, 3}
	boardBase = []uint8{22, 14, 17, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24}

	unexpectedError = errors.New("unexpected error")

	ctx = context.TODO()

	tx = &sql.Tx{}
)

func boardBaseFresh() []uint8 {
	fresh := make([]uint8, len(boardBase))
	copy(fresh, boardBase)
	return fresh
}

type Dictionary struct {
	mock.Mock
}

func (d *Dictionary) LemmaIsValid(lemma string) (result bool, err error) {
	args := d.Called(lemma)
	return args.Bool(0), args.Error(1)
}

type Transactional struct {
	mock.Mock
}

func (t *Transactional) BeginTransaction(ctx context.Context) (tx *sql.Tx, err error) {
	args := t.Called(ctx)
	tx = args.Get(0).(*sql.Tx)
	err = args.Error(1)
	return
}

func (t *Transactional) FinalizeTransaction(tx *sql.Tx, err error) error {
	expectedError := t.Called(tx, err).Error(0)
	if expectedError != nil {
		return expectedError
	}
	return err
}

func (t *Transactional) InsertGame(ctx context.Context, tx *sql.Tx, game data.Game) (data.Game, error) {
	args := t.Called(ctx, tx, game)
	err := args.Error(0)
	if err != nil {
		game = data.Game{}
	}
	game.Id = gameId
	return game, err
}

func (t *Transactional) InsertGamePlayer(ctx context.Context, tx *sql.Tx, game data.Game, player data.Player) (data.Game, error) {
	args := t.Called(ctx, tx, game, player)
	err := args.Error(0)
	if err != nil {
		game = data.Game{}
	}
	game.Players = append(game.Players, player)
	return game, err
}

func (t *Transactional) GetPlayerById(ctx context.Context, playerId data.PlayerId) (player data.Player, err error) {
	args := t.Called(playerId)
	player = args.Get(0).(data.Player)
	err = args.Error(1)
	player.Id = playerId
	return
}

func (t *Transactional) GetPlayersByGameId(ctx context.Context, gameId data.GameId) (players []data.Player, err error) {
	args := t.Called(ctx, gameId)
	players = args.Get(0).([]data.Player)
	err = args.Error(1)
	return
}

func (t *Transactional) GetGameById(ctx context.Context, tx *sql.Tx, gameId data.GameId) (game data.Game, err error) {
	args := t.Called(ctx, tx, gameId)
	game = args.Get(0).(data.Game)
	err = args.Error(1)
	game.Id = gameId
	return
}

func (t *Transactional) GetGamePlayersByGameId(ctx context.Context, tx *sql.Tx, gameId data.GameId) (gamePlayers []data.GamePlayer, err error) {
	args := t.Called(ctx, tx, gameId)
	gamePlayers = args.Get(0).([]data.GamePlayer)
	err = args.Error(1)
	return
}

func (t *Transactional) GetGamesByPlayerId(ctx context.Context, playerId data.PlayerId) (games []data.Game, err error) {
	args := t.Called(playerId)
	games = args.Get(0).([]data.Game)
	err = args.Error(1)
	return
}

func (t *Transactional) LogPlayedWord(ctx context.Context, tx *sql.Tx, gameId data.GameId, playerId data.PlayerId, word string) error {
	return t.Called(ctx, tx, gameId, playerId).Error(0)
}

func (t *Transactional) GetPlayedWordsByGameId(ctx context.Context, gameId data.GameId) (playedWords []data.PlayedWord, err error) {
	args := t.Called(ctx, gameId)
	playedWords = args.Get(0).([]data.PlayedWord)
	err = args.Error(1)
	return
}

func (t *Transactional) UpdateGame(ctx context.Context, tx *sql.Tx, game data.Game) error {
	return t.Called().Error(0)
}

func (t *Transactional) UpdatePlayer(ctx context.Context, tx *sql.Tx, player data.Player) error {
	return t.Called().Error(0)
}

func (t *Transactional) UpsertPlayer(ctx context.Context, tx *sql.Tx, player data.Player) error {
	return t.Called().Error(0)
}

func (t *Transactional) GetPlayerByDeviceFingerprint(ctx context.Context, tx *sql.Tx, fingerprint data.DeviceFingerprint) (player data.Player, err error) {
	return
}