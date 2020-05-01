package letter_block

import (
	"context"
	"errors"
	"math/rand"
	"regexp"

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

type LogicOfApplication interface {
	NewGame(ctx context.Context, firstPlayerId data.PlayerId, numberOfPlayer uint8) (data.Game, error)
	TakeTurn(ctx context.Context, gameId data.GameId, playerId data.PlayerId, word []uint8) (data.Game, error)
	Join(ctx context.Context, gameId data.GameId, playerId data.PlayerId) (data.Game, error)
}

type Application struct {
	transactional data.Transactional
	dictionaries  map[string]dictionary.Dictionary
}

func NewApplication(transactional data.Transactional, dictionaries map[string]dictionary.Dictionary) *Application {
	return &Application{
		transactional: transactional,
		dictionaries:  dictionaries,
	}
}

func (a *Application) NewGame(ctx context.Context, firstPlayerId data.PlayerId, numberOfPlayer uint8) (game data.Game, err error) {
	if numberOfPlayer < 2 || 5 < numberOfPlayer {
		err = ErrorNumberOfPlayer
		return
	}

	boardBase := make([]uint8, 25)
	for i := range boardBase {
		boardBase[i] = uint8(rand.Uint64() % 26)
	}
	player, err := a.transactional.GetPlayerById(ctx, firstPlayerId)
	if err != nil {
		return
	}

	tx, err := a.transactional.BeginTransaction(ctx)
	if err != nil {
		return
	}
	defer func() {
		err = a.transactional.FinalizeTransaction(tx, err)
		if err != nil {
			game = data.Game{}
		}
	}()

	game = data.Game{
		CurrentPlayerOrder: 0,
		NumberOfPlayer:     numberOfPlayer,
		BoardBase:          boardBase,
		BoardPositioning:   make([]uint8, 25),
		State:              data.ONGOING,
	}

	game, err = a.transactional.InsertGame(ctx, tx, game)
	if err != nil {
		return
	}

	game, err = a.transactional.InsertGamePlayer(ctx, tx, game, player)
	if err != nil {
		return
	}

	return
}

func (a *Application) TakeTurn(ctx context.Context, gameId data.GameId, playerId data.PlayerId, word []uint8) (game data.Game, err error) {
	tx, err := a.transactional.BeginTransaction(ctx)
	if err != nil {
		return
	}
	defer func() {
		err = a.transactional.FinalizeTransaction(tx, err)
	}()

	game, err = a.transactional.GetGameById(ctx, tx, gameId)
	if err != nil {
		return
	}

	if game.State != data.ONGOING {
		err = ErrorGameIsUnplayable
		return
	}

	gamePlayers, err := a.transactional.GetGamePlayersByGameId(ctx, tx, gameId)
	if err != nil {
		return
	}

	if uint8(len(gamePlayers)) < game.NumberOfPlayer { // waiting for other player to join
		err = ErrorNotYourTurn
		return
	} else if gamePlayers[game.CurrentPlayerOrder].PlayerId != playerId { // not your turn
		err = ErrorNotYourTurn
		return
	}

	wordOnce := make(map[uint8]bool)
	wordByte := make([]byte, len(word))
	for i, wordPosition := range word {
		if wordOnce[wordPosition] {
			err = ErrorDoesntMakeWord
			return
		} else {
			wordOnce[wordPosition] = true
		}
		wordByte[i] = alphabet[game.BoardBase[wordPosition]]
	}

	wordString := string(wordByte)
	var valid bool
	valid, err = a.dictionaries["id-id"].LemmaIsValid(wordString)
	if err != nil {
		return
	}
	if !valid {
		err = ErrorWordInvalid
		return
	}

	err = a.transactional.LogPlayedWord(ctx, tx, game.Id, playerId, wordString)
	if err != nil {
		if exist, _ := regexp.MatchString("Error 2601", err.Error()); exist {
			err = ErrorWordHavePlayed
		}
		return
	}

	positioningSpace := uint8(len(gamePlayers)) + 1
	for _, position := range word {
		boardPosition := game.BoardPositioning[position]
		if boardPosition == 0 {
			game.BoardPositioning[position] = game.CurrentPlayerOrder + 1
		} else {
			ownedBy := boardPosition % positioningSpace
			currentStrength := boardPosition/positioningSpace + 1
			if ownedBy == game.CurrentPlayerOrder+1 {
				if currentStrength < maxStrength {
					game.BoardPositioning[position] += positioningSpace
				}
			} else {
				if currentStrength > 1 {
					game.BoardPositioning[position] -= positioningSpace
				} else {
					game.BoardPositioning[position] = game.CurrentPlayerOrder + 1
				}
			}
		}
	}

	game.CurrentPlayerOrder += 1
	if game.CurrentPlayerOrder >= uint8(len(gamePlayers)) {
		game.CurrentPlayerOrder = 0
	}

	if gameIsEnding(game) {
		game.State = data.END
	}

	err = a.transactional.UpdateGame(ctx, tx, game)
	if err != nil {
		return
	}

	return
}

func (a *Application) Join(ctx context.Context, gameId data.GameId, playerId data.PlayerId) (game data.Game, err error) {
	tx, err := a.transactional.BeginTransaction(ctx)
	if err != nil {
		return
	}
	defer func() {
		err = a.transactional.FinalizeTransaction(tx, err)
	}()

	game, err = a.transactional.GetGameById(ctx, tx, gameId)
	if err != nil {
		return
	}

	var player data.Player
	player, err = a.transactional.GetPlayerById(ctx, playerId)
	if err != nil {
		return
	}

	var gamePlayers []data.GamePlayer
	gamePlayers, err = a.transactional.GetGamePlayersByGameId(ctx, tx, gameId)
	if err != nil {
		return
	}

	if !(uint8(len(gamePlayers)) < game.NumberOfPlayer) {
		err = ErrorPlayerIsEnough
		return
	}

	game, err = a.transactional.InsertGamePlayer(ctx, tx, game, player)
	if err != nil {
		return
	}

	return
}

func gameIsEnding(game data.Game) bool {
	for _, positioning := range game.BoardPositioning {
		if positioning == 0 {
			return false
		}
	}
	return true
}
