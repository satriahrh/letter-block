package service_test

import (
	"testing"

	"github.com/satriahrh/letter-block/data"
	"github.com/satriahrh/letter-block/dictionary"
	"github.com/satriahrh/letter-block/service"
	"github.com/stretchr/testify/assert"
)

func TestApplication_GetGames(t *testing.T) {
	t.Run("ErrorGetGamesByPlayerId", func(t *testing.T) {
		trans := &Transactional{}
		trans.On("GetGamesByPlayerId", playerId).
			Return([]data.Game{}, unexpectedError)

		svc := service.NewService(trans, make(map[string]dictionary.Dictionary))
		_, err := svc.GetGames(ctx, playerId)
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
		trans.On("GetGamesByPlayerId", playerId).
			Return([]data.Game{game}, nil)

		svc := service.NewService(trans, make(map[string]dictionary.Dictionary))
		games, err := svc.GetGames(ctx, playerId)
		if assert.NoError(t, err) {
			assert.Equal(t, []data.Game{game}, games)
		}
	})
}
