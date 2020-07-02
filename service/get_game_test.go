package service_test

import (
	"database/sql"
	"testing"

	"github.com/satriahrh/letter-block/data"
	"github.com/satriahrh/letter-block/dictionary"
	"github.com/satriahrh/letter-block/service"
	"github.com/stretchr/testify/assert"
)

func TestApplication_GetGame(t *testing.T) {
	t.Run("ErrorGetGameById", func(t *testing.T) {
		trans := &Transactional{}
		trans.On("GetGameById", ctx, (*sql.Tx)(nil), gameId).
			Return(data.Game{}, unexpectedError)

		svc := service.NewService(trans, make(map[string]dictionary.Dictionary))
		_, err := svc.GetGame(ctx, gameId)
		assert.EqualError(t, err, unexpectedError.Error())
	})
	t.Run("Success", func(t *testing.T) {
		trans := &Transactional{}

		game := data.Game{
			Id:                 gameId,
			CurrentPlayerOrder: 1,
			NumberOfPlayer:     2,
			State:              data.ONGOING,
			BoardBase:          boardBaseFresh(),
			BoardPositioning:   make([]uint8, 25),
		}
		trans.On("GetGameById", ctx, (*sql.Tx)(nil), gameId).
			Return(game, nil)

		playedWords := []data.PlayedWord{
			{players[0].Id, "KATA"},
			{players[1].Id, "KITA"},
		}
		trans.On("GetPlayedWordsByGameId", ctx, gameId).
			Return(playedWords, nil)

		trans.On("GetPlayersByGameId", ctx, gameId).
			Return(players, nil)

		svc := service.NewService(trans, make(map[string]dictionary.Dictionary))
		actual, err := svc.GetGame(ctx, gameId)
		if assert.NoError(t, err) {
			game.PlayedWords = playedWords
			game.Players = players
			assert.Equal(t, game, actual)
		}
	})
}
