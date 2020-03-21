package letter_block

import (
	"context"
	"errors"
	"github.com/satriahrh/letter-block/data"
	"math/rand"
)

var (
	ErrorPlayerInsufficient          = errors.New("minimum number of player is 2")
	ErrorBoardSizeInsufficient       = errors.New("minimum board size is 5")
	ErrorMaximumStrengthInsufficient = errors.New("minimum strengh is 2")
	ErrorPlayerNotFound              = errors.New("player not found")
)

type Slot struct {
	Owner    data.Player
	Strength uint
	Letter   uint8
}

type Game struct {
	Players         []data.Player
	Board           [][]Slot
	MaximumStrength uint
	CurrentTurn     uint
}

type ApplicationLogic interface {
	NewGame(context.Context, []string, uint, uint) (*Game, error)
	TakeTurn(string, []uint, string) (*Game, error)
}

type Application struct {
	Data *data.Data
}

func NewApplication(d *data.Data) (*Application, error) {
	return &Application{
		Data: d,
	}, nil
}

func (a *Application) NewGame(ctx context.Context, usernames []string, boardSize, maximumStrength uint) (*Game, error) {
	if len(usernames) < 2 {
		return nil, ErrorPlayerInsufficient
	}
	if boardSize < 5 {
		return nil, ErrorBoardSizeInsufficient
	}
	if maximumStrength < 2 {
		return nil, ErrorMaximumStrengthInsufficient
	}
	board := make([][]Slot, boardSize)
	for i := uint(0); i < boardSize; i++ {
		board[i] = make([]Slot, boardSize)
		for j := uint(0); j < boardSize; j++ {
			board[i][j] = Slot{
				Letter:   uint8(rand.Uint64() % 26),
				Strength: 0,
			}
		}
	}

	players, err := a.Data.Mysql.GetPlayersByUsernames(ctx, usernames)
	if err != nil {
		return &Game{}, err
	}
	if len(players) != len(usernames) {
		return &Game{}, ErrorPlayerNotFound
	}

	game := Game{
		Players:         players,
		Board:           board,
		MaximumStrength: maximumStrength,
		CurrentTurn:     0,
	}

	return &game, nil
}

func (a *Application) TakeTurn(username string, words []uint, gameID string) (*Game, error) {
	return &Game{}, nil
}
