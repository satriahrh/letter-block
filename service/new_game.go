package service

import (
	"context"

	"github.com/satriahrh/letter-block/data"
)

func (a *application) NewGame(ctx context.Context, firstPlayerId data.PlayerId, numberOfPlayer uint8) (game data.Game, err error) {
	if numberOfPlayer < 2 || 5 < numberOfPlayer {
		err = ErrorNumberOfPlayer
		return
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

	// can ignore the error since we know that id is exist in
	letterBank, _ := data.NewLetterBank("id")
	letterBank.Shuffle()

	// can ignore the error since the initial bank would be 98
	boardBase := letterBank.Pop(25)

	game = data.Game{
		CurrentPlayerOrder: 0,
		NumberOfPlayer:     numberOfPlayer,
		LetterBank:         letterBank,
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
