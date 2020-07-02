package service_test

import (
	"testing"

	"github.com/satriahrh/letter-block/dictionary"
	"github.com/satriahrh/letter-block/service"
	"github.com/stretchr/testify/assert"
)

func TestApplication_GetPlayer(t *testing.T) {
	trans := &Transactional{}

	trans.On("GetPlayerById", playerId).
		Return(players[0], nil)

	svc := service.NewService(trans, make(map[string]dictionary.Dictionary))
	player, err := svc.GetPlayer(ctx, playerId)
	if assert.NoError(t, err) {
		assert.Equal(t, players[0], player)
	}
}
