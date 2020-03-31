package letter_block

import (
	"context"
	"database/sql"
	"errors"
	"github.com/satriahrh/letter-block/data"
	"math/rand"
)

var (
	ErrorBoardSizeInsufficient       = errors.New("minimum board size is 5")
	ErrorDoesntMakeWord              = errors.New("doesn't make word")
	ErrorMaximumStrengthInsufficient = errors.New("minimum strengh is 2")
	ErrorNotYourTurn                 = errors.New("not your turn")
	ErrorPlayerInsufficient          = errors.New("minimum number of player is 2")
	ErrorPlayerNotFound              = errors.New("player not found")
	ErrorUnauthorized                = errors.New("player is not authorized")
	ErrorWordInvalid                 = errors.New("word invalid")
)

var (
	alphabet = "abcdefghijklmnopqrstuvwxyz"
)

type LogicOfApplication interface {
	NewGame(context.Context, []string, uint8, uint8) (data.Game, error)
	TakeTurn(context.Context, uint64, uint64, []uint16) (data.Game, error)
}

type Application struct {
	transactional data.Transactional
}

func NewApplication(transactional data.Transactional) *Application {
	return &Application{
		transactional: transactional,
	}
}

func (a *Application) NewGame(ctx context.Context, usernames []string, boardSize, maxStrength uint8) (data.Game, error) {
	if len(usernames) < 2 {
		return data.Game{}, ErrorPlayerInsufficient
	}
	if boardSize < 5 {
		return data.Game{}, ErrorBoardSizeInsufficient
	}
	if maxStrength < 2 {
		return data.Game{}, ErrorMaximumStrengthInsufficient
	}
	boardBase := make([]uint8, boardSize*boardSize)
	for i := range boardBase {
		boardBase[i] = uint8(rand.Uint64() % 26)
	}

	players, err := a.transactional.GetPlayersByUsernames(ctx, usernames)
	if err != nil {
		return data.Game{}, err
	}
	if len(players) != len(usernames) {
		return data.Game{}, ErrorPlayerNotFound
	}

	game := data.Game{
		CurrentPlayerID:  players[0].ID,
		Players:          players,
		MaxStrength:      maxStrength,
		BoardBase:        boardBase,
		BoardPositioning: make([]uint8, boardSize*boardSize),
	}

	return a.transactional.InsertGame(ctx, game)
}

func (a *Application) TakeTurn(ctx context.Context, gamePlayerID uint64, playerID uint64, word []uint16) (game data.Game, err error) {
	var player data.Player

	game.ID, player.ID, err = a.transactional.GetGamePlayerByID(ctx, gamePlayerID)
	if err != nil {
		return data.Game{}, err
	}

	if player.ID != playerID {
		return data.Game{}, ErrorUnauthorized
	}

	tx, err := a.transactional.BeginTransaction(ctx, &sql.TxOptions{
		Isolation: sql.LevelWriteCommitted,
		ReadOnly:  false,
	})
	if err != nil {
		return
	}
	defer func() {
		err = a.transactional.FinalizeTransaction(tx, err)
	}()

	game, err = a.transactional.GetGameByID(ctx, tx, game.ID)
	if err != nil {
		return
	}

	if game.CurrentPlayerID != player.ID {
		err = ErrorNotYourTurn
		return
	}

	wordOnce := make(map[uint16]bool)
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

	// TODO validate wordByte is in dictionary
	// TODO validate word haven't played? -> query game_words
	// TODO update positioning on Game
	// TODO update next player on Game
	// TODO check victory condition

	return
}
