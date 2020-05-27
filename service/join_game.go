package service

import (
	"context"

	"github.com/satriahrh/letter-block/data"
)

func (a *application) JoinGame(ctx context.Context, gameId data.GameId, playerId data.PlayerId) (game data.Game, err error) {
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

	game.Players = []data.Player{}
	for _, gamePlayer := range gamePlayers {
		game.Players = append(game.Players, data.Player{Id: gamePlayer.PlayerId})
	}

	game, err = a.transactional.InsertGamePlayer(ctx, tx, game, player)
	if err != nil {
		return
	}

	return
}
