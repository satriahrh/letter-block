package letterblock

import (
	"github.com/satriahrh/letter-block/data"
	"github.com/satriahrh/letter-block/dictionary"
	"regexp"

	"context"
	"errors"
	"math/rand"
)

var (
	// ErrorBoardSizeInsufficient given board size is more than 5
	ErrorBoardSizeInsufficient = errors.New("minimum board size is 5")

	// ErrorDoesntMakeWord given word is not a valid word
	ErrorDoesntMakeWord = errors.New("doesn't make word")

	// ErrorGameIsUnplayable game state is not playable
	ErrorGameIsUnplayable = errors.New("game is unplayable")

	// ErrorMaximumStrengthInsufficient given strength is more than 2
	ErrorMaximumStrengthInsufficient = errors.New("maximum strength is 2")

	// ErrorNotYourTurn not this player turn
	ErrorNotYourTurn = errors.New("this player is not eligible to take curent turn")

	// ErrorPlayerInsufficient number of player is less than 2
	ErrorPlayerInsufficient = errors.New("minimum number of player is 2")

	// ErrorPlayerNotFound no player found in db
	ErrorPlayerNotFound = errors.New("player not found")

	// ErrorUnauthorized player is not authorized to access this resource
	ErrorUnauthorized = errors.New("player is not authorized")

	// ErrorWordHavePlayed word is played
	ErrorWordHavePlayed = errors.New("word have played")

	// ErrorWordInvalid word is invalid by dictionary
	ErrorWordInvalid = errors.New("word invalid")
)

var (
	alphabet = "abcdefghijklmnopqrstuvwxyz"
)

// LogicOfApplication is the main logic of the application
type LogicOfApplication interface {
	NewGame(context.Context, []string, uint8, uint8) (data.Game, error)
	TakeTurn(context.Context, uint64, uint64, []uint16) (data.Game, error)
}

// Application is implementation of LogicOfApplication
type Application struct {
	transactional data.Transactional
	dictionaries  map[string]dictionary.Dictionary
}

// NewApplication is contructor of Application
func NewApplication(transactional data.Transactional, dictionaries map[string]dictionary.Dictionary) *Application {
	return &Application{
		transactional: transactional,
		dictionaries:  dictionaries,
	}
}

// NewGame create a new game
func (a *Application) NewGame(ctx context.Context, usernames []string, boardSize, maxStrength uint8) (game data.Game, err error) {
	if len(usernames) < 2 {
		err = ErrorPlayerInsufficient
		return
	}
	if boardSize < 5 {
		err = ErrorBoardSizeInsufficient
		return
	}
	if maxStrength < 2 {
		err = ErrorMaximumStrengthInsufficient
		return
	}
	boardBase := make([]uint8, boardSize*boardSize)
	for i := range boardBase {
		boardBase[i] = uint8(rand.Uint64() % 26)
	}

	// Retrieve Players
	players, err := a.transactional.GetPlayersByUsernames(ctx, usernames)
	if err != nil {
		return data.Game{}, err
	}
	if len(players) != len(usernames) {
		return data.Game{}, ErrorPlayerNotFound
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
		CurrentOrder:     1,
		MaxStrength:      maxStrength,
		BoardBase:        boardBase,
		BoardPositioning: make([]uint8, boardSize*boardSize),
		State:            data.ONGOING,
	}

	game, err = a.transactional.InsertGame(ctx, tx, game)
	if err != nil {
		return
	}

	game, err = a.transactional.InsertGamePlayerBulk(ctx, tx, game, players)
	if err != nil {
		return
	}

	return
}

// TakeTurn for player to take his turn
func (a *Application) TakeTurn(ctx context.Context, gamePlayerID uint64, playerID uint64, word []uint8) (game data.Game, err error) {
	var gamePlayer data.GamePlayer

	gamePlayer, err = a.transactional.GetGamePlayerByID(ctx, gamePlayerID)
	if err != nil {
		return data.Game{}, err
	}

	if gamePlayer.PlayerID != playerID {
		return data.Game{}, ErrorUnauthorized
	}

	tx, err := a.transactional.BeginTransaction(ctx)
	if err != nil {
		return
	}
	defer func() {
		err = a.transactional.FinalizeTransaction(tx, err)
	}()

	game, err = a.transactional.GetGameByID(ctx, tx, gamePlayer.GameID)
	if err != nil {
		return
	}

	if game.State != data.ONGOING {
		err = ErrorGameIsUnplayable
		return
	}

	if game.CurrentOrder != gamePlayer.Ordering {
		err = ErrorNotYourTurn
		return
	}

	wordOnce := make(map[uint8]bool)
	wordByte := make([]byte, len(word))
	for i, wordPosition := range word {
		if wordOnce[wordPosition] {
			err = ErrorDoesntMakeWord
			return
		}
		wordOnce[wordPosition] = true
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

	err = a.transactional.LogPlayedWord(ctx, tx, game.ID, gamePlayer.PlayerID, wordString)
	if err != nil {
		if exist, _ := regexp.MatchString("Error 2601", err.Error()); exist {
			err = ErrorWordHavePlayed
		}
		return
	}

	gamePlayers, err := a.transactional.GetGamePlayersByGameID(ctx, tx, game.ID)
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
				if currentStrength < game.MaxStrength {
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

	game.CurrentOrder++
	if game.CurrentOrder > uint8(len(gamePlayers)) {
		game.CurrentOrder = 1
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
