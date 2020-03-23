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

type LogicOfApplication interface {
	NewGame(context.Context, []string, uint8, uint8) (data.Game, error)
	TakeTurn(string, []uint, string) (data.Game, error)
}

type Application struct {
	Data *data.Data
}

func NewApplication(d *data.Data) (*Application, error) {
	return &Application{
		Data: d,
	}, nil
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

	players, err := a.Data.Mysql.GetPlayersByUsernames(ctx, usernames)
	if err != nil {
		return data.Game{}, err
	}
	if len(players) != len(usernames) {
		return data.Game{}, ErrorPlayerNotFound
	}

	game := data.Game{
		CurrentTurn:      0,
		Players:          players,
		MaxStrength:      maxStrength,
		BoardBase:        boardBase,
		BoardPositioning: make([]uint8, boardSize*boardSize),
	}

	return game, nil
}

func (a *Application) TakeTurn(username string, words []uint, gameID string) (data.Game, error) {
	return data.Game{}, nil
}
