package letter_block_test

import (
	"context"
	"database/sql"
	"errors"
	"github.com/satriahrh/letter-block"
	"github.com/satriahrh/letter-block/data"
	"github.com/satriahrh/letter-block/dictionary"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"testing"
	"time"
)

var (
	gameId = uint64(time.Now().UnixNano())

	// len(usernames) >= 2
	usernames = []string{"sarjono", "mukti"}

	players = []data.Player{
		{Id: uint64(time.Now().UnixNano()), Username: usernames[0]},
		{Id: uint64(time.Now().UnixNano()), Username: usernames[1]},
	}

	playerId = players[0].Id

	gamePlayerId = uint64(time.Now().UnixNano())

	// boardSize >= 5
	boardSize = uint8(5)

	word      = []uint16{0, 1, 2, 3}
	boardBase = []uint8{22, 14, 17, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24}

	// maximumStrength >= 2
	maxStrength = uint8(2)

	ctx = context.TODO()

	tx = &sql.Tx{}
)

type Dictionary struct {
	mock.Mock
}

func (d *Dictionary) LemmaIsValid(lemma string) (result bool, err error) {
	args := d.Called(lemma)
	return args.Bool(0), args.Error(1)
}

type Transactional struct {
	mock.Mock
}

func (t *Transactional) BeginTransaction(ctx context.Context) (tx *sql.Tx, err error) {
	args := t.Called(ctx)
	tx = args.Get(0).(*sql.Tx)
	err = args.Error(1)
	return
}

func (t *Transactional) FinalizeTransaction(tx *sql.Tx, err error) error {
	expectedError := t.Called(tx, err).Error(0)
	if expectedError != nil {
		return expectedError
	}
	return err
}

func (t *Transactional) InsertGame(ctx context.Context, tx *sql.Tx, game data.Game) (data.Game, error) {
	args := t.Called(ctx, tx, game)
	err := args.Error(0)
	if err != nil {
		game = data.Game{}
	}
	game.Id = gameId
	return game, err
}

func (t *Transactional) InsertGamePlayerBulk(ctx context.Context, tx *sql.Tx, game data.Game, players []data.Player) (data.Game, error) {
	args := t.Called(ctx, tx, game, players)
	err := args.Error(0)
	if err != nil {
		game = data.Game{}
	}
	game.Players = players
	return game, err
}

func (t *Transactional) GetPlayersByUsernames(ctx context.Context, usernames []string) (players []data.Player, err error) {
	args := t.Called(ctx, usernames)
	players = args.Get(0).([]data.Player)
	err = args.Error(1)
	return
}

func (t *Transactional) GetGameById(ctx context.Context, tx *sql.Tx, gameId uint64) (game data.Game, err error) {
	args := t.Called(ctx, tx, gameId)
	game = args.Get(0).(data.Game)
	err = args.Error(1)
	return
}

func (t *Transactional) GetGamePlayerById(ctx context.Context, gamePlayerId uint64) (gameId uint64, playerId uint64, err error) {
	args := t.Called(ctx, gamePlayerId)
	gameId = args.Get(0).(uint64)
	playerId = args.Get(1).(uint64)
	err = args.Error(2)
	return
}

func TestApplicationNewGame(t *testing.T) {
	t.Run("ErrorPlayerInsufficient", func(t *testing.T) {
		t.Run("LessThanTwo", func(t *testing.T) {
			application := letter_block.NewApplication(&Transactional{}, make(map[string]dictionary.Dictionary))
			_, err := application.NewGame(ctx, usernames[:1], boardSize, maxStrength)
			assert.EqualError(t, err, letter_block.ErrorPlayerInsufficient.Error())
		})
	})
	t.Run("ErrorBoardSizeInsufficient", func(t *testing.T) {
		t.Run("LessThanFive", func(t *testing.T) {
			application := letter_block.NewApplication(&Transactional{}, make(map[string]dictionary.Dictionary))
			_, err := application.NewGame(ctx, usernames, boardSize-1, maxStrength)
			assert.EqualError(t, err, letter_block.ErrorBoardSizeInsufficient.Error())
		})
	})
	t.Run("ErrorMaximumStrengthInsufficient", func(t *testing.T) {
		t.Run("LessThanTwo", func(t *testing.T) {
			application := letter_block.NewApplication(&Transactional{}, make(map[string]dictionary.Dictionary))
			_, err := application.NewGame(ctx, usernames, boardSize, maxStrength-1)
			assert.EqualError(t, err, letter_block.ErrorMaximumStrengthInsufficient.Error())
		})
	})
	t.Run("ErrorRetrievePlayers", func(t *testing.T) {
		t.Run("ErrorQuerying", func(t *testing.T) {
			trans := &Transactional{}
			trans.On("GetPlayersByUsernames", ctx, usernames).
				Return([]data.Player{}, sql.ErrConnDone)

			application := letter_block.NewApplication(trans, make(map[string]dictionary.Dictionary))
			_, err := application.NewGame(ctx, usernames, boardSize, maxStrength)
			assert.EqualError(t, err, sql.ErrConnDone.Error())
		})
		t.Run("ErrorPlayerNotFound", func(t *testing.T) {
			trans := &Transactional{}
			trans.On("GetPlayersByUsernames", ctx, usernames).
				Return(players[:1], nil)

			application := letter_block.NewApplication(trans, make(map[string]dictionary.Dictionary))
			_, err := application.NewGame(ctx, usernames, boardSize, maxStrength)
			assert.EqualError(t, err, letter_block.ErrorPlayerNotFound.Error())
		})
	})
	t.Run("ErrorBeginTransaction", func(t *testing.T) {
		trans := &Transactional{}
		trans.On("GetPlayersByUsernames", ctx, usernames).
			Return(players, nil)
		trans.On("BeginTransaction", ctx).
			Return(&sql.Tx{}, sql.ErrConnDone)

		application := letter_block.NewApplication(trans, make(map[string]dictionary.Dictionary))
		_, err := application.NewGame(ctx, usernames, boardSize, maxStrength)
		assert.EqualError(t, err, sql.ErrConnDone.Error())
	})
	t.Run("ErrorInsertGame", func(t *testing.T) {
		trans := &Transactional{}
		trans.On("GetPlayersByUsernames", ctx, usernames).
			Return(players, nil)
		tx := &sql.Tx{}
		trans.On("BeginTransaction", ctx).
			Return(tx, nil)
		unexpectedError := errors.New("unexpected error")
		trans.On("InsertGame", ctx, tx,
			mock.MatchedBy(func(game data.Game) bool {
				return assert.Equal(t, players[0].Id, game.CurrentPlayerId) &&
					assert.Equal(t, maxStrength, game.MaxStrength) &&
					assert.Len(t, game.BoardBase, int(boardSize*boardSize)) &&
					assert.Equal(t, make([]uint8, boardSize*boardSize), game.BoardPositioning) &&
					assert.Empty(t, game.Players) &&
					assert.Empty(t, game.Id)
			}),
		).
			Return(unexpectedError)
		trans.On("FinalizeTransaction", tx, unexpectedError).
			Return(nil)

		application := letter_block.NewApplication(trans, make(map[string]dictionary.Dictionary))
		_, err := application.NewGame(ctx, usernames, boardSize, maxStrength)
		assert.EqualError(t, err, unexpectedError.Error())
	})
	t.Run("ErrorInsertGamePlayerBulk", func(t *testing.T) {
		trans := &Transactional{}
		trans.On("GetPlayersByUsernames", ctx, usernames).
			Return(players, nil)
		tx := &sql.Tx{}
		trans.On("BeginTransaction", ctx).
			Return(tx, nil)
		trans.On("InsertGame", ctx, tx,
			mock.MatchedBy(func(game data.Game) bool {
				return assert.Equal(t, players[0].Id, game.CurrentPlayerId) &&
					assert.Equal(t, maxStrength, game.MaxStrength) &&
					assert.Len(t, game.BoardBase, int(boardSize*boardSize)) &&
					assert.Equal(t, make([]uint8, boardSize*boardSize), game.BoardPositioning) &&
					assert.Empty(t, game.Players) &&
					assert.Empty(t, game.Id)
			}),
		).
			Return(nil)
		unexpectedError := errors.New("unexpected error")
		trans.On("InsertGamePlayerBulk", ctx, tx,
			mock.MatchedBy(func(game data.Game) bool {
				return assert.Equal(t, players[0].Id, game.CurrentPlayerId) &&
					assert.Equal(t, maxStrength, game.MaxStrength) &&
					assert.Len(t, game.BoardBase, int(boardSize*boardSize)) &&
					assert.Equal(t, make([]uint8, boardSize*boardSize), game.BoardPositioning) &&
					assert.Empty(t, game.Players) &&
					assert.Equal(t, gameId, game.Id)
			}),
			players,
		).
			Return(unexpectedError)
		trans.On("FinalizeTransaction", tx, unexpectedError).
			Return(nil)

		application := letter_block.NewApplication(trans, make(map[string]dictionary.Dictionary))
		_, err := application.NewGame(ctx, usernames, boardSize, maxStrength)
		assert.EqualError(t, err, unexpectedError.Error())
	})
	t.Run("Success", func(t *testing.T) {
		testSuite := func(t *testing.T, finalizeError error) (data.Game, error) {
			trans := &Transactional{}
			trans.On("GetPlayersByUsernames", ctx, usernames).
				Return(players, nil)
			tx := &sql.Tx{}
			trans.On("BeginTransaction", ctx).
				Return(tx, nil)
			trans.On("InsertGame", ctx, tx,
				mock.MatchedBy(func(game data.Game) bool {
					return assert.Equal(t, players[0].Id, game.CurrentPlayerId) &&
						assert.Equal(t, maxStrength, game.MaxStrength) &&
						assert.Len(t, game.BoardBase, int(boardSize*boardSize)) &&
						assert.Equal(t, make([]uint8, boardSize*boardSize), game.BoardPositioning) &&
						assert.Empty(t, game.Players) &&
						assert.Empty(t, game.Id)
				}),
			).
				Return(nil)
			trans.On("InsertGamePlayerBulk", ctx, tx,
				mock.MatchedBy(func(game data.Game) bool {
					return assert.Equal(t, players[0].Id, game.CurrentPlayerId) &&
						assert.Equal(t, maxStrength, game.MaxStrength) &&
						assert.Len(t, game.BoardBase, int(boardSize*boardSize)) &&
						assert.Equal(t, make([]uint8, boardSize*boardSize), game.BoardPositioning) &&
						assert.Empty(t, game.Players) &&
						assert.Equal(t, gameId, game.Id)
				}),
				players,
			).
				Return(nil)
			trans.On("FinalizeTransaction", tx, nil).
				Return(finalizeError)

			application := letter_block.NewApplication(trans, make(map[string]dictionary.Dictionary))
			return application.NewGame(ctx, usernames, boardSize, maxStrength)
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
				assert.Equal(t, players[0].Id, game.CurrentPlayerId)
				assert.Equal(t, maxStrength, game.MaxStrength)
				assert.Len(t, game.BoardBase, int(boardSize*boardSize))
				assert.Equal(t, make([]uint8, boardSize*boardSize), game.BoardPositioning)
				assert.Equal(t, players, game.Players)
				assert.Equal(t, gameId, game.Id)
			}
		})
	})
}

func TestApplicationTakeTurn(t *testing.T) {
	t.Run("ErrorGetGamePlayerId", func(t *testing.T) {
		trans := &Transactional{}
		trans.On("GetGamePlayerById", ctx, gamePlayerId).
			Return(uint64(0), uint64(0), sql.ErrNoRows)

		application := letter_block.NewApplication(trans, make(map[string]dictionary.Dictionary))
		_, err := application.TakeTurn(ctx, gamePlayerId, playerId, word)
		assert.EqualError(t, err, sql.ErrNoRows.Error())
	})
	t.Run("ErrorUnauthorized", func(t *testing.T) {
		trans := &Transactional{}
		trans.On("GetGamePlayerById", ctx, gamePlayerId).
			Return(gameId, playerId+1, nil)

		application := letter_block.NewApplication(trans, make(map[string]dictionary.Dictionary))
		_, err := application.TakeTurn(ctx, gamePlayerId, playerId, word)
		assert.EqualError(t, err, letter_block.ErrorUnauthorized.Error())
	})
	t.Run("ErrorBeginTransaction", func(t *testing.T) {
		trans := &Transactional{}
		trans.On("GetGamePlayerById", ctx, gamePlayerId).
			Return(gameId, playerId, nil)
		trans.On("BeginTransaction", ctx).
			Return(&sql.Tx{}, sql.ErrConnDone)

		application := letter_block.NewApplication(trans, make(map[string]dictionary.Dictionary))
		_, err := application.TakeTurn(ctx, gamePlayerId, playerId, word)
		assert.EqualError(t, err, sql.ErrConnDone.Error())
	})
	t.Run("ErrorGetGameById", func(t *testing.T) {
		trans := &Transactional{}
		trans.On("GetGamePlayerById", ctx, gamePlayerId).
			Return(gameId, playerId, nil)
		trans.On("BeginTransaction", ctx).
			Return(tx, nil)
		trans.On("GetGameById", ctx, tx, gameId).
			Return(data.Game{}, sql.ErrConnDone)
		trans.On("FinalizeTransaction", tx, sql.ErrConnDone).
			Return(nil)

		application := letter_block.NewApplication(trans, make(map[string]dictionary.Dictionary))
		_, err := application.TakeTurn(ctx, gamePlayerId, playerId, word)
		assert.EqualError(t, err, sql.ErrConnDone.Error())
	})
	t.Run("ErrorNotYourTurn", func(t *testing.T) {
		trans := &Transactional{}
		trans.On("GetGamePlayerById", ctx, gamePlayerId).
			Return(gameId, playerId, nil)
		trans.On("BeginTransaction", ctx).
			Return(tx, nil)
		trans.On("GetGameById", ctx, tx, gameId).
			Return(data.Game{
				CurrentPlayerId: playerId + 1,
				BoardBase:       boardBase,
			}, nil)
		trans.On("FinalizeTransaction", tx, letter_block.ErrorNotYourTurn).
			Return(nil)

		application := letter_block.NewApplication(trans, make(map[string]dictionary.Dictionary))
		_, err := application.TakeTurn(ctx, gamePlayerId, playerId, word)
		assert.EqualError(t, err, letter_block.ErrorNotYourTurn.Error())
	})
	t.Run("ErrorDoesntMakeWord", func(t *testing.T) {
		trans := &Transactional{}
		trans.On("GetGamePlayerById", ctx, gamePlayerId).
			Return(gameId, playerId, nil)
		trans.On("BeginTransaction", ctx).
			Return(tx, nil)
		trans.On("GetGameById", ctx, tx, gameId).
			Return(data.Game{
				CurrentPlayerId: playerId,
				BoardBase:       boardBase,
			}, nil)
		trans.On("FinalizeTransaction", tx, letter_block.ErrorDoesntMakeWord).
			Return(nil)

		application := letter_block.NewApplication(trans, make(map[string]dictionary.Dictionary))
		_, err := application.TakeTurn(ctx, gamePlayerId, playerId, append(word, word[0]))
		assert.EqualError(t, err, letter_block.ErrorDoesntMakeWord.Error())
	})
	t.Run("ErrorValidatingLemma", func(t *testing.T) {
		trans := &Transactional{}
		trans.On("GetGamePlayerById", ctx, gamePlayerId).
			Return(gameId, playerId, nil)
		trans.On("BeginTransaction", ctx).
			Return(tx, nil)
		trans.On("GetGameById", ctx, tx, gameId).
			Return(data.Game{
				CurrentPlayerId: playerId,
				BoardBase:       boardBase,
			}, nil)
		unexpectedError := errors.New("unexpected error")
		trans.On("FinalizeTransaction", tx, unexpectedError).
			Return(nil)

		dict := &Dictionary{}

		dict.On("LemmaIsValid", "word").
			Return(false, unexpectedError)

		application := letter_block.NewApplication(trans, map[string]dictionary.Dictionary{
			"id-id": dict,
		})
		_, err := application.TakeTurn(ctx, gamePlayerId, playerId, word)
		assert.EqualError(t, err, unexpectedError.Error())
	})
	t.Run("ErrorWordInvalid", func(t *testing.T) {
		trans := &Transactional{}
		trans.On("GetGamePlayerById", ctx, gamePlayerId).
			Return(gameId, playerId, nil)
		trans.On("BeginTransaction", ctx).
			Return(tx, nil)
		trans.On("GetGameById", ctx, tx, gameId).
			Return(data.Game{
				CurrentPlayerId: playerId,
				BoardBase:       boardBase,
			}, nil)
		trans.On("FinalizeTransaction", tx, letter_block.ErrorWordInvalid).
			Return(nil)

		dict := &Dictionary{}

		dict.On("LemmaIsValid", "word").
			Return(false, nil)

		application := letter_block.NewApplication(trans, map[string]dictionary.Dictionary{
			"id-id": dict,
		})
		_, err := application.TakeTurn(ctx, gamePlayerId, playerId, word)
		assert.EqualError(t, err, letter_block.ErrorWordInvalid.Error())
	})
}
