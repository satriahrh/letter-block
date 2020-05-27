package service_test

import (
	"database/sql"
	"errors"
	"testing"

	"github.com/satriahrh/letter-block/data"
	"github.com/satriahrh/letter-block/dictionary"
	"github.com/satriahrh/letter-block/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestApplicationNewGame(t *testing.T) {
	t.Run("ErrorNumberOfPlayer", func(t *testing.T) {
		testSuite := func(t *testing.T, sample uint8) {
			svc := service.NewService(&Transactional{}, make(map[string]dictionary.Dictionary))
			_, err := svc.NewGame(ctx, playerId, sample)
			assert.EqualError(t, err, service.ErrorNumberOfPlayer.Error())
		}
		t.Run("BelowTwo", func(t *testing.T) {
			testSuite(t, 1)
		})
		t.Run("AboveFive", func(t *testing.T) {
			testSuite(t, 6)
		})
	})
	t.Run("ErrorGetPlayerById", func(t *testing.T) {
		trans := &Transactional{}
		trans.On("GetPlayerById", playerId).
			Return(data.Player{}, sql.ErrNoRows)

		svc := service.NewService(trans, make(map[string]dictionary.Dictionary))
		_, err := svc.NewGame(ctx, playerId, numberOfPlayer)
		assert.EqualError(t, err, sql.ErrNoRows.Error())
	})
	t.Run("ErrorBeginTransaction", func(t *testing.T) {
		trans := &Transactional{}
		trans.On("GetPlayerById", playerId).
			Return(players[0], nil)
		trans.On("BeginTransaction", ctx).
			Return(&sql.Tx{}, sql.ErrConnDone)

		svc := service.NewService(trans, make(map[string]dictionary.Dictionary))
		_, err := svc.NewGame(ctx, playerId, numberOfPlayer)
		assert.EqualError(t, err, sql.ErrConnDone.Error())
	})
	t.Run("ErrorInsertGame", func(t *testing.T) {
		trans := &Transactional{}
		trans.On("GetPlayerById", playerId).
			Return(players[0], nil)
		tx := &sql.Tx{}
		trans.On("BeginTransaction", ctx).
			Return(tx, nil)
		unexpectedError := errors.New("unexpected error")
		trans.On("InsertGame", ctx, tx,
			mock.MatchedBy(func(game data.Game) bool {
				return assert.Equal(t, uint8(0), game.CurrentPlayerOrder) &&
					assert.Len(t, game.BoardBase, 25) &&
					assert.Equal(t, make([]uint8, 25), game.BoardPositioning) &&
					assert.Empty(t, game.Players) &&
					assert.Empty(t, game.Id)
			}),
		).
			Return(unexpectedError)
		trans.On("FinalizeTransaction", tx, unexpectedError).
			Return(nil)

		svc := service.NewService(trans, make(map[string]dictionary.Dictionary))
		_, err := svc.NewGame(ctx, playerId, numberOfPlayer)
		assert.EqualError(t, err, unexpectedError.Error())
	})
	t.Run("ErrorInsertGamePlayer", func(t *testing.T) {
		trans := &Transactional{}
		trans.On("GetPlayerById", playerId).
			Return(players[0], nil)
		tx := &sql.Tx{}
		trans.On("BeginTransaction", ctx).
			Return(tx, nil)
		trans.On("InsertGame", ctx, tx,
			mock.MatchedBy(func(game data.Game) bool {
				return assert.Equal(t, uint8(0), game.CurrentPlayerOrder) &&
					assert.Len(t, game.BoardBase, 25) &&
					assert.Equal(t, make([]uint8, 25), game.BoardPositioning) &&
					assert.Empty(t, game.Players) &&
					assert.Empty(t, game.Id)
			}),
		).
			Return(nil)
		unexpectedError := errors.New("unexpected error")
		trans.On("InsertGamePlayer", ctx, tx,
			mock.MatchedBy(func(game data.Game) bool {
				return assert.Equal(t, uint8(0), game.CurrentPlayerOrder) &&
					assert.Len(t, game.BoardBase, 25) &&
					assert.Equal(t, make([]uint8, 25), game.BoardPositioning) &&
					assert.Empty(t, game.Players) &&
					assert.Equal(t, gameId, game.Id)
			}),
			players[0],
		).
			Return(unexpectedError)
		trans.On("FinalizeTransaction", tx, unexpectedError).
			Return(nil)

		svc := service.NewService(trans, make(map[string]dictionary.Dictionary))
		_, err := svc.NewGame(ctx, playerId, numberOfPlayer)
		assert.EqualError(t, err, unexpectedError.Error())
	})
	t.Run("Success", func(t *testing.T) {
		testSuite := func(t *testing.T, finalizeError error) (data.Game, error) {
			trans := &Transactional{}
			trans.On("GetPlayerById", playerId).
				Return(players[0], nil)
			tx := &sql.Tx{}
			trans.On("BeginTransaction", ctx).
				Return(tx, nil)
			trans.On("InsertGame", ctx, tx,
				mock.MatchedBy(func(game data.Game) bool {
					return assert.Equal(t, uint8(0), game.CurrentPlayerOrder) &&
						assert.Len(t, game.BoardBase, 25) &&
						assert.Equal(t, make([]uint8, 25), game.BoardPositioning) &&
						assert.Empty(t, game.Players) &&
						assert.Empty(t, game.Id)
				}),
			).
				Return(nil)
			trans.On("InsertGamePlayer", ctx, tx,
				mock.MatchedBy(func(game data.Game) bool {
					return assert.Equal(t, uint8(0), game.CurrentPlayerOrder) &&
						assert.Len(t, game.BoardBase, 25) &&
						assert.Equal(t, make([]uint8, 25), game.BoardPositioning) &&
						assert.Empty(t, game.Players) &&
						assert.Equal(t, gameId, game.Id)
				}),
				players[0],
			).
				Return(nil)
			trans.On("FinalizeTransaction", tx, nil).
				Return(finalizeError)

			svc := service.NewService(trans, make(map[string]dictionary.Dictionary))
			return svc.NewGame(ctx, playerId, numberOfPlayer)
		}
		// Can be happened anywhere
		t.Run("ErrorFinalizeTransaction", func(t *testing.T) {
			unexpectedError := errors.New("unexpected error")
			game, err := testSuite(t, unexpectedError)
			assert.EqualError(t, err, unexpectedError.Error())
			assert.Empty(t, game)
		})
		t.Run("SuccessFinalizeTransaction", func(t *testing.T) {
			game, err := testSuite(t, nil)
			if assert.NoError(t, err) && assert.NotEmpty(t, game) {
				assert.Equal(t, uint8(0), game.CurrentPlayerOrder)
				assert.Len(t, game.BoardBase, 25)
				assert.Equal(t, data.ONGOING, game.State)
				assert.Equal(t, make([]uint8, 25), game.BoardPositioning)
				assert.Equal(t, players[:1], game.Players)
				assert.Equal(t, gameId, game.Id)
			}
		})
	})
}
