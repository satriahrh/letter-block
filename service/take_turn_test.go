package service_test

import (
	"database/sql"
	"errors"
	"fmt"
	"testing"

	"github.com/satriahrh/letter-block/data"
	"github.com/satriahrh/letter-block/dictionary"
	"github.com/satriahrh/letter-block/service"
	"github.com/stretchr/testify/assert"
)

func TestApplicationTakeTurn(t *testing.T) {
	t.Run("ErrorBeginTransaction", func(t *testing.T) {
		trans := &Transactional{}
		trans.On("BeginTransaction", ctx).
			Return(&sql.Tx{}, sql.ErrConnDone)

		svc := service.NewService(trans, make(map[string]dictionary.Dictionary))
		_, err := svc.TakeTurn(ctx, gameId, playerId, word)
		assert.EqualError(t, err, sql.ErrConnDone.Error())
	})
	t.Run("ErrorGetGameById", func(t *testing.T) {
		trans := &Transactional{}
		trans.On("BeginTransaction", ctx).
			Return(tx, nil)
		trans.On("GetGameById", ctx, tx, gameId).
			Return(data.Game{}, sql.ErrConnDone)
		trans.On("FinalizeTransaction", tx, sql.ErrConnDone).
			Return(nil)

		svc := service.NewService(trans, make(map[string]dictionary.Dictionary))
		_, err := svc.TakeTurn(ctx, gameId, playerId, word)
		assert.EqualError(t, err, sql.ErrConnDone.Error())
	})
	t.Run("ErrorGameIsUnplayable", func(t *testing.T) {
		testSuite := func(t *testing.T, state data.GameState) {
			trans := &Transactional{}
			trans.On("BeginTransaction", ctx).
				Return(tx, nil)
			trans.On("GetGameById", ctx, tx, gameId).
				Return(data.Game{
					CurrentPlayerOrder: 2, BoardBase: boardBaseFresh(), State: state,
				}, nil)
			trans.On("FinalizeTransaction", tx, service.ErrorGameIsUnplayable).
				Return(nil)

			svc := service.NewService(trans, make(map[string]dictionary.Dictionary))
			_, err := svc.TakeTurn(ctx, gameId, playerId, word)
			assert.EqualError(t, err, service.ErrorGameIsUnplayable.Error())
		}
		t.Run("Created", func(t *testing.T) {
			testSuite(t, data.CREATED)
		})
		t.Run("End", func(t *testing.T) {
			testSuite(t, data.END)
		})
	})
	t.Run("ErrorGetGamePlayersByGameId", func(t *testing.T) {
		trans := &Transactional{}
		trans.On("BeginTransaction", ctx).
			Return(tx, nil)
		trans.On("GetGameById", ctx, tx, gameId).
			Return(data.Game{
				CurrentPlayerOrder: 2, NumberOfPlayer: 2,
				BoardBase: boardBaseFresh(), State: data.ONGOING,
			}, nil)
		trans.On("GetGamePlayersByGameId", ctx, tx, gameId).
			Return([]data.GamePlayer{}, sql.ErrConnDone)
		trans.On("FinalizeTransaction", tx, sql.ErrConnDone).
			Return(nil)

		svc := service.NewService(trans, make(map[string]dictionary.Dictionary))
		_, err := svc.TakeTurn(ctx, gameId, playerId, word)
		assert.EqualError(t, err, sql.ErrConnDone.Error())
	})
	t.Run("ErrorNotYourTurn", func(t *testing.T) {
		testSuite := func(t *testing.T, gamePlayers []data.GamePlayer) {
			trans := &Transactional{}
			trans.On("BeginTransaction", ctx).
				Return(tx, nil)
			trans.On("GetGameById", ctx, tx, gameId).
				Return(data.Game{
					CurrentPlayerOrder: 1, NumberOfPlayer: 2,
					BoardBase: boardBaseFresh(), State: data.ONGOING,
				}, nil)
			trans.On("GetGamePlayersByGameId", ctx, tx, gameId).
				Return(gamePlayers, nil)
			trans.On("FinalizeTransaction", tx, service.ErrorNotYourTurn).
				Return(nil)

			svc := service.NewService(trans, make(map[string]dictionary.Dictionary))
			_, err := svc.TakeTurn(ctx, gameId, playerId, word)
			assert.EqualError(t, err, service.ErrorNotYourTurn.Error())
		}
		t.Run("WaitingForOtherPlayer", func(t *testing.T) {
			testSuite(t, []data.GamePlayer{
				{GameId: gameId, PlayerId: playerId},
			})
		})
		t.Run("NotYourTurn", func(t *testing.T) {
			testSuite(t, []data.GamePlayer{
				{GameId: gameId, PlayerId: playerId},
				{GameId: gameId, PlayerId: playerId + 1},
			})
		})
	})
	t.Run("ErrorDoesntMakeWord", func(t *testing.T) {
		trans := &Transactional{}
		trans.On("BeginTransaction", ctx).
			Return(tx, nil)
		trans.On("GetGameById", ctx, tx, gameId).
			Return(data.Game{
				CurrentPlayerOrder: 0, NumberOfPlayer: 2,
				BoardBase: boardBaseFresh(), State: data.ONGOING,
				LetterBank: letterBank,
			}, nil)
		trans.On("GetGamePlayersByGameId", ctx, tx, gameId).
			Return([]data.GamePlayer{
				{GameId: gameId, PlayerId: playerId},
				{GameId: gameId, PlayerId: players[1].Id},
			}, nil)
		trans.On("FinalizeTransaction", tx, service.ErrorDoesntMakeWord).
			Return(nil)

		svc := service.NewService(trans, make(map[string]dictionary.Dictionary))
		_, err := svc.TakeTurn(ctx, gameId, playerId, append(word, word[0]))
		assert.EqualError(t, err, service.ErrorDoesntMakeWord.Error())
	})
	t.Run("ErrorValidatingLemma", func(t *testing.T) {
		trans := &Transactional{}
		trans.On("BeginTransaction", ctx).
			Return(tx, nil)
		trans.On("GetGameById", ctx, tx, gameId).
			Return(data.Game{
				CurrentPlayerOrder: 0, NumberOfPlayer: 2,
				BoardBase: boardBaseFresh(), State: data.ONGOING,
				LetterBank: letterBank,
			}, nil)
		trans.On("GetGamePlayersByGameId", ctx, tx, gameId).
			Return([]data.GamePlayer{
				{GameId: gameId, PlayerId: playerId},
				{GameId: gameId, PlayerId: players[1].Id},
			}, nil)
		unexpectedError := errors.New("unexpected error")
		trans.On("FinalizeTransaction", tx, unexpectedError).
			Return(nil)

		dict := &Dictionary{}

		dict.On("LemmaIsValid", "word").
			Return(false, unexpectedError)

		svc := service.NewService(trans, map[string]dictionary.Dictionary{
			"id-id": dict,
		})
		_, err := svc.TakeTurn(ctx, gameId, playerId, word)
		assert.EqualError(t, err, unexpectedError.Error())
		fmt.Println(boardBase)
	})
	t.Run("ErrorWordInvalid", func(t *testing.T) {
		trans := &Transactional{}
		trans.On("BeginTransaction", ctx).
			Return(tx, nil)
		trans.On("GetGameById", ctx, tx, gameId).
			Return(data.Game{
				CurrentPlayerOrder: 0, NumberOfPlayer: 2,
				BoardBase: boardBaseFresh(), State: data.ONGOING,
				LetterBank: letterBank,
			}, nil)
		trans.On("GetGamePlayersByGameId", ctx, tx, gameId).
			Return([]data.GamePlayer{
				{GameId: gameId, PlayerId: playerId},
				{GameId: gameId, PlayerId: players[1].Id},
			}, nil)
		trans.On("FinalizeTransaction", tx, service.ErrorWordInvalid).
			Return(nil)

		dict := &Dictionary{}

		dict.On("LemmaIsValid", "word").
			Return(false, nil)

		svc := service.NewService(trans, map[string]dictionary.Dictionary{
			"id-id": dict,
		})
		_, err := svc.TakeTurn(ctx, gameId, playerId, word)
		assert.EqualError(t, err, service.ErrorWordInvalid.Error())
	})
	t.Run("ErrorLogPlayedWord", func(t *testing.T) {
		t.Run("Unexpected", func(t *testing.T) {
			trans := &Transactional{}
			trans.On("BeginTransaction", ctx).
				Return(tx, nil)
			trans.On("GetGameById", ctx, tx, gameId).
				Return(data.Game{
					CurrentPlayerOrder: 0, NumberOfPlayer: 2,
					BoardBase: boardBaseFresh(), State: data.ONGOING,
					LetterBank: letterBank,
				}, nil)
			trans.On("GetGamePlayersByGameId", ctx, tx, gameId).
				Return([]data.GamePlayer{
					{GameId: gameId, PlayerId: playerId},
					{GameId: gameId, PlayerId: players[1].Id},
				}, nil)
			unexpectedError := errors.New("unexpected error")
			trans.On("LogPlayedWord", ctx, tx, gameId, playerId).
				Return(unexpectedError)
			trans.On("FinalizeTransaction", tx, unexpectedError).
				Return(nil)

			dict := &Dictionary{}

			dict.On("LemmaIsValid", "word").
				Return(true, nil)

			svc := service.NewService(trans, map[string]dictionary.Dictionary{
				"id-id": dict,
			})
			_, err := svc.TakeTurn(ctx, gameId, playerId, word)
			assert.EqualError(t, err, unexpectedError.Error())
		})
		t.Run("WordHavePlayed", func(t *testing.T) {
			trans := &Transactional{}
			trans.On("BeginTransaction", ctx).
				Return(tx, nil)
			trans.On("GetGameById", ctx, tx, gameId).
				Return(data.Game{
					CurrentPlayerOrder: 0, NumberOfPlayer: 2,
					BoardBase: boardBaseFresh(), State: data.ONGOING,
					LetterBank: letterBank,
				}, nil)
			trans.On("GetGamePlayersByGameId", ctx, tx, gameId).
				Return([]data.GamePlayer{
					{GameId: gameId, PlayerId: playerId},
					{GameId: gameId, PlayerId: players[1].Id},
				}, nil)
			trans.On("LogPlayedWord", ctx, tx, gameId, playerId).
				Return(errors.New("---Error 2601---"))
			trans.On("FinalizeTransaction", tx, service.ErrorWordHavePlayed).
				Return(nil)

			dict := &Dictionary{}

			dict.On("LemmaIsValid", "word").
				Return(true, nil)

			svc := service.NewService(trans, map[string]dictionary.Dictionary{
				"id-id": dict,
			})
			_, err := svc.TakeTurn(ctx, gameId, playerId, word)
			assert.EqualError(t, err, service.ErrorWordHavePlayed.Error())
		})
	})
	t.Run("Positioning", func(t *testing.T) {
		positioningSuite := func(boardPositioning, expectedBoardPositioning []uint8) {
			trans := &Transactional{}
			trans.On("BeginTransaction", ctx).
				Return(tx, nil)
			trans.On("GetGameById", ctx, tx, gameId).
				Return(data.Game{
					CurrentPlayerOrder: 0, NumberOfPlayer: 2, BoardPositioning: boardPositioning,
					BoardBase: boardBaseFresh(), State: data.ONGOING,
					LetterBank: letterBank,
				}, nil)
			trans.On("GetGamePlayersByGameId", ctx, tx, gameId).
				Return([]data.GamePlayer{
					{GameId: gameId, PlayerId: playerId},
					{GameId: gameId, PlayerId: players[1].Id},
				}, nil)
			trans.On("LogPlayedWord", ctx, tx, gameId, playerId).
				Return(nil)
			trans.On("UpdateGame").
				Return(nil)
			trans.On("FinalizeTransaction", tx, nil).
				Return(nil)

			dict := &Dictionary{}

			dict.On("LemmaIsValid", "worda").
				Return(true, nil)

			svc := service.NewService(trans, map[string]dictionary.Dictionary{
				"id-id": dict,
			})
			game, err := svc.TakeTurn(ctx, gameId, playerId, []uint8{0, 1, 2, 3, 4})
			if assert.NoError(t, err) {
				assert.Equal(t, expectedBoardPositioning, game.BoardPositioning)
			}
		}
		t.Run("Vacant", func(t *testing.T) {
			boardPositioning := make([]uint8, 25)
			expectedBoardPositioning := []uint8{1, 1, 1, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
			positioningSuite(boardPositioning, expectedBoardPositioning)
		})
		t.Run("AcquiredByUs", func(t *testing.T) {
			t.Run("NotMax", func(t *testing.T) {
				boardPositioning := []uint8{1, 1, 1, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
				expectedBoardPositioning := []uint8{4, 4, 4, 4, 4, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
				positioningSuite(boardPositioning, expectedBoardPositioning)
			})
			t.Run("Max", func(t *testing.T) {
				boardPositioning := []uint8{4, 4, 4, 4, 4, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
				expectedBoardPositioning := []uint8{4, 4, 4, 4, 4, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
				positioningSuite(boardPositioning, expectedBoardPositioning)
			})
		})
		t.Run("AcquiredByThem", func(t *testing.T) {
			t.Run("Strong", func(t *testing.T) {
				boardPositioning := []uint8{5, 5, 5, 5, 5, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
				expectedBoardPositioning := []uint8{2, 2, 2, 2, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
				positioningSuite(boardPositioning, expectedBoardPositioning)
			})
			t.Run("Weak", func(t *testing.T) {
				boardPositioning := []uint8{2, 2, 2, 2, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
				expectedBoardPositioning := []uint8{1, 1, 1, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
				positioningSuite(boardPositioning, expectedBoardPositioning)
			})
		})
		t.Run("Mix", func(t *testing.T) {
			boardPositioning := []uint8{0, 1, 4, 2, 5, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
			expectedBoardPositioning := []uint8{1, 4, 4, 1, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
			positioningSuite(boardPositioning, expectedBoardPositioning)
		})
	})
	t.Run("Ordering", func(t *testing.T) {
		orderingSuite := func(currentPlayerOrder, nextOrder uint8) {
			trans := &Transactional{}
			trans.On("BeginTransaction", ctx).
				Return(tx, nil)
			trans.On("GetGameById", ctx, tx, gameId).
				Return(data.Game{
					CurrentPlayerOrder: currentPlayerOrder, NumberOfPlayer: 2, BoardPositioning: make([]uint8, 25),
					BoardBase: boardBaseFresh(), State: data.ONGOING,
					LetterBank: letterBank,
				}, nil)
			trans.On("GetGamePlayersByGameId", ctx, tx, gameId).
				Return([]data.GamePlayer{
					{GameId: gameId, PlayerId: players[0].Id},
					{GameId: gameId, PlayerId: players[1].Id},
				}, nil)
			trans.On("LogPlayedWord", ctx, tx, gameId, players[currentPlayerOrder].Id).
				Return(nil)
			trans.On("UpdateGame").
				Return(nil)
			trans.On("FinalizeTransaction", tx, nil).
				Return(nil)

			dict := &Dictionary{}

			dict.On("LemmaIsValid", "worda").
				Return(true, nil)

			svc := service.NewService(trans, map[string]dictionary.Dictionary{
				"id-id": dict,
			})
			game, err := svc.TakeTurn(ctx, gameId, players[currentPlayerOrder].Id, []uint8{0, 1, 2, 3, 4})
			if assert.NoError(t, err) {
				assert.Equal(t, nextOrder, game.CurrentPlayerOrder)
			}
		}
		t.Run("NotExceeding", func(t *testing.T) {
			orderingSuite(0, 1)
		})
		t.Run("Exceeding", func(t *testing.T) {
			orderingSuite(1, 0)
		})
	})
	t.Run("GameIsEnding", func(t *testing.T) {
		testSuite := func(t *testing.T, boardPositioning []uint8, expectedEnd bool) {
			trans := &Transactional{}
			trans.On("BeginTransaction", ctx).
				Return(tx, nil)
			trans.On("GetGameById", ctx, tx, gameId).
				Return(data.Game{
					CurrentPlayerOrder: 0, NumberOfPlayer: 2, BoardPositioning: boardPositioning,
					BoardBase: boardBaseFresh(), State: data.ONGOING,
					LetterBank: letterBank,
				}, nil)
			trans.On("GetGamePlayersByGameId", ctx, tx, gameId).
				Return([]data.GamePlayer{
					{GameId: gameId, PlayerId: players[0].Id},
					{GameId: gameId, PlayerId: players[1].Id},
				}, nil)
			trans.On("LogPlayedWord", ctx, tx, gameId, playerId).
				Return(nil)
			trans.On("UpdateGame").
				Return(nil)
			trans.On("FinalizeTransaction", tx, nil).
				Return(nil)

			dict := &Dictionary{}

			dict.On("LemmaIsValid", "worda").
				Return(true, nil)

			svc := service.NewService(trans, map[string]dictionary.Dictionary{
				"id-id": dict,
			})
			game, err := svc.TakeTurn(ctx, gameId, playerId, []uint8{0, 1, 2, 3, 4})
			if assert.NoError(t, err) {
				if expectedEnd {
					assert.Equal(t, data.END, game.State)
				} else {
					assert.Equal(t, data.ONGOING, game.State)
				}
			}
		}
		t.Run("No", func(t *testing.T) {
			testSuite(t, make([]uint8, 25), false)
		})
		t.Run("Yes", func(t *testing.T) {
			boardPositioning := make([]uint8, 25)
			for i := range boardPositioning {
				if i > 4 {
					boardPositioning[i] = 1
				}
			}
			testSuite(t, boardPositioning, true)
		})
	})
	t.Run("ErrorUpdateGame", func(t *testing.T) {
		trans := &Transactional{}
		trans.On("BeginTransaction", ctx).
			Return(tx, nil)
		trans.On("GetGameById", ctx, tx, gameId).
			Return(data.Game{
				CurrentPlayerOrder: 0, NumberOfPlayer: 2, BoardPositioning: make([]uint8, 25),
				BoardBase: boardBaseFresh(), State: data.ONGOING,
				LetterBank: letterBank,
			}, nil)
		trans.On("GetGamePlayersByGameId", ctx, tx, gameId).
			Return([]data.GamePlayer{
				{GameId: gameId, PlayerId: players[0].Id},
				{GameId: gameId, PlayerId: players[1].Id},
			}, nil)
		trans.On("LogPlayedWord", ctx, tx, gameId, playerId).
			Return(nil)
		unexpectedError := errors.New("unexpected error")
		trans.On("UpdateGame").
			Return(unexpectedError)
		trans.On("FinalizeTransaction", tx, unexpectedError).
			Return(nil)

		dict := &Dictionary{}

		dict.On("LemmaIsValid", "word").
			Return(true, nil)

		svc := service.NewService(trans, map[string]dictionary.Dictionary{
			"id-id": dict,
		})
		_, err := svc.TakeTurn(ctx, gameId, playerId, word)
		assert.EqualError(t, err, unexpectedError.Error())
	})
}
