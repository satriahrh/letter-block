package service

import (
	"context"
	"log"

	"github.com/satriahrh/letter-block/data"
)

func (a *application) GetGames(ctx context.Context, playerId data.PlayerId) (games []data.Game, err error) {
	games, err = a.transactional.GetGamesByPlayerId(ctx, playerId)
	if err != nil {
		log.Println(err)
		return
	}

	return
}
