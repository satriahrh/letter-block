package service

import (
	"context"

	"github.com/satriahrh/letter-block/data"
)

func (a *application) GetPlayer(ctx context.Context, playerId data.PlayerId) (player data.Player, err error) {
	return a.transactional.GetPlayerById(ctx, playerId)
}
