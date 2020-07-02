package service_test

import (
	"database/sql"
	"testing"

	"github.com/satriahrh/letter-block/data"
	"github.com/satriahrh/letter-block/dictionary"
	"github.com/satriahrh/letter-block/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestApplication_JoinGame(t *testing.T) {
	player := players[1]
	game := data.Game{
		Id: gameId, CurrentPlayerOrder: 1, NumberOfPlayer: 2,
		BoardBase: boardBaseFresh(), State: data.ONGOING,
	}
	t.Run("ErrorBeginTransaction", func(t *testing.T) {
		trans := &Transactional{}
		trans.On("BeginTransaction", ctx).
			Return(&sql.Tx{}, unexpectedError)

		svc := service.NewService(trans, make(map[string]dictionary.Dictionary))
		_, err := svc.JoinGame(ctx, game.Id, player.Id)
		assert.EqualError(t, err, unexpectedError.Error())
	})
	t.Run("ErrorGetGameById", func(t *testing.T) {
		trans := &Transactional{}
		trans.On("BeginTransaction", ctx).
			Return(tx, nil)
		trans.On("GetGameById", ctx, tx, game.Id).
			Return(data.Game{}, unexpectedError)
		trans.On("FinalizeTransaction", tx, unexpectedError).
			Return(nil)

		svc := service.NewService(trans, make(map[string]dictionary.Dictionary))
		_, err := svc.JoinGame(ctx, game.Id, player.Id)
		assert.EqualError(t, err, unexpectedError.Error())
	})
	t.Run("ErrorGetPlayerById", func(t *testing.T) {
		trans := &Transactional{}
		trans.On("BeginTransaction", ctx).
			Return(tx, nil)
		trans.On("GetGameById", ctx, tx, game.Id).
			Return(game, nil)
		trans.On("GetPlayerById", player.Id).
			Return(data.Player{}, unexpectedError)
		trans.On("FinalizeTransaction", tx, unexpectedError).
			Return(nil)

		svc := service.NewService(trans, make(map[string]dictionary.Dictionary))
		_, err := svc.JoinGame(ctx, game.Id, player.Id)
		assert.EqualError(t, err, unexpectedError.Error())
	})
	t.Run("ErrorGetGamePlayersByGameId", func(t *testing.T) {
		trans := &Transactional{}
		trans.On("BeginTransaction", ctx).
			Return(tx, nil)
		trans.On("GetGameById", ctx, tx, game.Id).
			Return(game, nil)
		trans.On("GetPlayerById", player.Id).
			Return(player, nil)
		trans.On("GetGamePlayersByGameId", ctx, tx, game.Id).
			Return([]data.GamePlayer{}, unexpectedError)
		trans.On("FinalizeTransaction", tx, unexpectedError).
			Return(nil)

		svc := service.NewService(trans, make(map[string]dictionary.Dictionary))
		_, err := svc.JoinGame(ctx, game.Id, player.Id)
		assert.EqualError(t, err, unexpectedError.Error())
	})
	t.Run("ErrorPlayerIsEnough", func(t *testing.T) {
		trans := &Transactional{}
		trans.On("BeginTransaction", ctx).
			Return(tx, nil)
		trans.On("GetGameById", ctx, tx, game.Id).
			Return(game, nil)
		trans.On("GetPlayerById", player.Id).
			Return(player, nil)
		trans.On("GetGamePlayersByGameId", ctx, tx, game.Id).
			Return(gamePlayers[:2], nil)
		trans.On("FinalizeTransaction", tx, service.ErrorPlayerIsEnough).
			Return(nil)

		svc := service.NewService(trans, make(map[string]dictionary.Dictionary))
		_, err := svc.JoinGame(ctx, game.Id, player.Id)
		assert.EqualError(t, err, service.ErrorPlayerIsEnough.Error())
	})
	t.Run("ErrorInsertGamePlayer", func(t *testing.T) {
		trans := &Transactional{}
		trans.On("BeginTransaction", ctx).
			Return(tx, nil)
		trans.On("GetGameById", ctx, tx, game.Id).
			Return(game, nil)
		trans.On("GetPlayerById", player.Id).
			Return(player, nil)
		trans.On("GetGamePlayersByGameId", ctx, tx, game.Id).
			Return(gamePlayers[:1], nil)
		trans.On("InsertGamePlayer", ctx, tx, mock.MatchedBy(func(calledGame data.Game) bool {
			return assert.Equal(t, game.Id, calledGame.Id)
		}), player).
			Return(unexpectedError)
		trans.On("FinalizeTransaction", tx, unexpectedError).
			Return(nil)

		svc := service.NewService(trans, make(map[string]dictionary.Dictionary))
		_, err := svc.JoinGame(ctx, game.Id, player.Id)
		assert.EqualError(t, err, unexpectedError.Error())
	})
	t.Run("Success", func(t *testing.T) {
		trans := &Transactional{}
		trans.On("BeginTransaction", ctx).
			Return(tx, nil)
		trans.On("GetGameById", ctx, tx, game.Id).
			Return(game, nil)
		trans.On("GetPlayerById", players[1].Id).
			Return(player, nil)
		trans.On("GetGamePlayersByGameId", ctx, tx, game.Id).
			Return(gamePlayers[:1], nil)
		trans.On("InsertGamePlayer", ctx, tx, mock.MatchedBy(func(calledGame data.Game) bool {
			return assert.Equal(t, game.Id, calledGame.Id)
		}), player).
			Return(nil)
		trans.On("FinalizeTransaction", tx, nil).
			Return(nil)

		svc := service.NewService(trans, make(map[string]dictionary.Dictionary))
		actualGame, err := svc.JoinGame(ctx, game.Id, player.Id)
		if assert.NoError(t, err) {
			assert.Equal(t, players, actualGame.Players)
		}
	})
}
