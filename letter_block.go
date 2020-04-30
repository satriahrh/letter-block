package letter_block

import (
	"github.com/satriahrh/letter-block/data"
	"github.com/satriahrh/letter-block/dictionary"
	"regexp"

	"context"
	"errors"
	"math/rand"
)

var (
	ErrorDoesntMakeWord   = errors.New("doesn't make word")
	ErrorGameIsUnplayable = errors.New("game is unplayable")
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
	TakeTurn(ctx context.Context, gamePlayerId data.GamePlayerId, playerId data.PlayerId, word []uint8) (data.Game, error)
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
		CurrentPlayerOrder: 1,
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

func (a *Application) TakeTurn(ctx context.Context, gamePlayerId data.GamePlayerId, playerId data.PlayerId, word []uint8) (game data.Game, err error) {
	var gamePlayer data.GamePlayer

	gamePlayer, err = a.transactional.GetGamePlayerById(ctx, gamePlayerId)
	if err != nil {
		return data.Game{}, err
	}

	if gamePlayer.PlayerId != playerId {
		return data.Game{}, ErrorUnauthorized
	}

	tx, err := a.transactional.BeginTransaction(ctx)
	if err != nil {
		return
	}
	defer func() {
		err = a.transactional.FinalizeTransaction(tx, err)
	}()

	game, err = a.transactional.GetGameById(ctx, tx, gamePlayer.GameId)
	if err != nil {
		return
	}

	if game.State != data.ONGOING {
		err = ErrorGameIsUnplayable
		return
	}

	if game.CurrentPlayerOrder != gamePlayer.Ordering {
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

	err = a.transactional.LogPlayedWord(ctx, tx, game.Id, gamePlayer.PlayerId, wordString)
	if err != nil {
		if exist, _ := regexp.MatchString("Error 2601", err.Error()); exist {
			err = ErrorWordHavePlayed
		}
		return
	}

	gamePlayers, err := a.transactional.GetGamePlayersByGameId(ctx, tx, game.Id)
	if err != nil {
		return
	}

	positioningSpace := uint8(len(gamePlayers)) + 1
	for _, position := range word {
		boardPosition := game.BoardPositioning[position]
		if boardPosition == 0 {
			game.BoardPositioning[position] = gamePlayer.Ordering
		} else {
			ownedBy := boardPosition % positioningSpace
			currentStrength := boardPosition/positioningSpace + 1
			if ownedBy == gamePlayer.Ordering {
				if currentStrength < maxStrength {
					game.BoardPositioning[position] += positioningSpace
				}
			} else {
				if currentStrength > 1 {
					game.BoardPositioning[position] -= positioningSpace
				} else {
					game.BoardPositioning[position] = gamePlayer.Ordering
				}
			}
		}
	}

	game.CurrentPlayerOrder += 1
	if game.CurrentPlayerOrder > uint8(len(gamePlayers)) {
		game.CurrentPlayerOrder = 1
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

func gameIsEnding(game data.Game) bool {
	for _, positioning := range game.BoardPositioning {
		if positioning == 0 {
			return false
		}
	}
	return true
}
