package letterblock_test

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
	gameID = uint64(time.Now().UnixNano())

	// len(usernames) >= 2
	usernames = []string{"sarjono", "mukti"}

	players = []data.Player{
		{ID: uint64(time.Now().UnixNano()), Username: usernames[0]},
		{ID: uint64(time.Now().UnixNano()), Username: usernames[1]},
	}

	playerID = players[0].ID

	gamePlayerID = uint64(time.Now().UnixNano())

	// boardSize >= 5
	boardSize = uint8(5)

	word      = []uint8{0, 1, 2, 3}
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
	game.ID = gameID
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

func (t *Transactional) GetGameByID(ctx context.Context, tx *sql.Tx, gameID uint64) (game data.Game, err error) {
	args := t.Called(ctx, tx, gameID)
	game = args.Get(0).(data.Game)
	err = args.Error(1)
	game.ID = gameID
	return
}

func (t *Transactional) GetGamePlayerByID(ctx context.Context, gamePlayerID uint64) (gamePlayer data.GamePlayer, err error) {
	args := t.Called(ctx, gamePlayerID)
	gamePlayer = args.Get(0).(data.GamePlayer)
	err = args.Error(1)
	return
}

func (t *Transactional) GetGamePlayersByGameID(ctx context.Context, tx *sql.Tx, gameID uint64) (gamePlayers []data.GamePlayer, err error) {
	args := t.Called(ctx, tx, gameID)
	gamePlayers = args.Get(0).([]data.GamePlayer)
	err = args.Error(1)
	return
}

func (t *Transactional) LogPlayedWord(ctx context.Context, tx *sql.Tx, gameID, playerID uint64, word string) error {
	return t.Called(ctx, tx, gameID, playerID).Error(0)
}

func (t *Transactional) UpdateGame(ctx context.Context, tx *sql.Tx, game data.Game) error {
	return t.Called().Error(0)
}

func TestApplicationNewGame(t *testing.T) {
	t.Run("ErrorPlayerInsufficient", func(t *testing.T) {
		t.Run("LessThanTwo", func(t *testing.T) {
			application := letterblock.NewApplication(&Transactional{}, make(map[string]dictionary.Dictionary))
			_, err := application.NewGame(ctx, usernames[:1], boardSize, maxStrength)
			assert.EqualError(t, err, letterblock.ErrorPlayerInsufficient.Error())
		})
	})
	t.Run("ErrorBoardSizeInsufficient", func(t *testing.T) {
		t.Run("LessThanFive", func(t *testing.T) {
			application := letterblock.NewApplication(&Transactional{}, make(map[string]dictionary.Dictionary))
			_, err := application.NewGame(ctx, usernames, boardSize-1, maxStrength)
			assert.EqualError(t, err, letterblock.ErrorBoardSizeInsufficient.Error())
		})
	})
	t.Run("ErrorMaximumStrengthInsufficient", func(t *testing.T) {
		t.Run("LessThanTwo", func(t *testing.T) {
			application := letterblock.NewApplication(&Transactional{}, make(map[string]dictionary.Dictionary))
			_, err := application.NewGame(ctx, usernames, boardSize, maxStrength-1)
			assert.EqualError(t, err, letterblock.ErrorMaximumStrengthInsufficient.Error())
		})
	})
	t.Run("ErrorRetrievePlayers", func(t *testing.T) {
		t.Run("ErrorQuerying", func(t *testing.T) {
			trans := &Transactional{}
			trans.On("GetPlayersByUsernames", ctx, usernames).
				Return([]data.Player{}, sql.ErrConnDone)

			application := letterblock.NewApplication(trans, make(map[string]dictionary.Dictionary))
			_, err := application.NewGame(ctx, usernames, boardSize, maxStrength)
			assert.EqualError(t, err, sql.ErrConnDone.Error())
		})
		t.Run("ErrorPlayerNotFound", func(t *testing.T) {
			trans := &Transactional{}
			trans.On("GetPlayersByUsernames", ctx, usernames).
				Return(players[:1], nil)

			application := letterblock.NewApplication(trans, make(map[string]dictionary.Dictionary))
			_, err := application.NewGame(ctx, usernames, boardSize, maxStrength)
			assert.EqualError(t, err, letterblock.ErrorPlayerNotFound.Error())
		})
	})
	t.Run("ErrorBeginTransaction", func(t *testing.T) {
		trans := &Transactional{}
		trans.On("GetPlayersByUsernames", ctx, usernames).
			Return(players, nil)
		trans.On("BeginTransaction", ctx).
			Return(&sql.Tx{}, sql.ErrConnDone)

		application := letterblock.NewApplication(trans, make(map[string]dictionary.Dictionary))
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
				return assert.Equal(t, uint8(1), game.CurrentOrder) &&
					assert.Equal(t, maxStrength, game.MaxStrength) &&
					assert.Len(t, game.BoardBase, int(boardSize*boardSize)) &&
					assert.Equal(t, make([]uint8, boardSize*boardSize), game.BoardPositioning) &&
					assert.Empty(t, game.Players) &&
					assert.Empty(t, game.ID)
			}),
		).
			Return(unexpectedError)
		trans.On("FinalizeTransaction", tx, unexpectedError).
			Return(nil)

		application := letterblock.NewApplication(trans, make(map[string]dictionary.Dictionary))
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
				return assert.Equal(t, uint8(1), game.CurrentOrder) &&
					assert.Equal(t, maxStrength, game.MaxStrength) &&
					assert.Len(t, game.BoardBase, int(boardSize*boardSize)) &&
					assert.Equal(t, make([]uint8, boardSize*boardSize), game.BoardPositioning) &&
					assert.Empty(t, game.Players) &&
					assert.Empty(t, game.ID)
			}),
		).
			Return(nil)
		unexpectedError := errors.New("unexpected error")
		trans.On("InsertGamePlayerBulk", ctx, tx,
			mock.MatchedBy(func(game data.Game) bool {
				return assert.Equal(t, uint8(1), game.CurrentOrder) &&
					assert.Equal(t, maxStrength, game.MaxStrength) &&
					assert.Len(t, game.BoardBase, int(boardSize*boardSize)) &&
					assert.Equal(t, make([]uint8, boardSize*boardSize), game.BoardPositioning) &&
					assert.Empty(t, game.Players) &&
					assert.Equal(t, gameID, game.ID)
			}),
			players,
		).
			Return(unexpectedError)
		trans.On("FinalizeTransaction", tx, unexpectedError).
			Return(nil)

		application := letterblock.NewApplication(trans, make(map[string]dictionary.Dictionary))
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
					return assert.Equal(t, uint8(1), game.CurrentOrder) &&
						assert.Equal(t, maxStrength, game.MaxStrength) &&
						assert.Len(t, game.BoardBase, int(boardSize*boardSize)) &&
						assert.Equal(t, data.ONGOING, game.State) &&
						assert.Equal(t, make([]uint8, boardSize*boardSize), game.BoardPositioning) &&
						assert.Empty(t, game.Players) &&
						assert.Empty(t, game.ID)
				}),
			).
				Return(nil)
			trans.On("InsertGamePlayerBulk", ctx, tx,
				mock.MatchedBy(func(game data.Game) bool {
					return assert.Equal(t, uint8(1), game.CurrentOrder) &&
						assert.Equal(t, maxStrength, game.MaxStrength) &&
						assert.Len(t, game.BoardBase, int(boardSize*boardSize)) &&
						assert.Equal(t, make([]uint8, boardSize*boardSize), game.BoardPositioning) &&
						assert.Empty(t, game.Players) &&
						assert.Equal(t, gameID, game.ID)
				}),
				players,
			).
				Return(nil)
			trans.On("FinalizeTransaction", tx, nil).
				Return(finalizeError)

			application := letterblock.NewApplication(trans, make(map[string]dictionary.Dictionary))
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
				assert.Equal(t, uint8(1), game.CurrentOrder)
				assert.Equal(t, maxStrength, game.MaxStrength)
				assert.Len(t, game.BoardBase, int(boardSize*boardSize))
				assert.Equal(t, data.ONGOING, game.State)
				assert.Equal(t, make([]uint8, boardSize*boardSize), game.BoardPositioning)
				assert.Equal(t, players, game.Players)
				assert.Equal(t, gameID, game.ID)
			}
		})
	})
}

func TestApplicationTakeTurn(t *testing.T) {
	t.Run("ErrorGetGamePlayerID", func(t *testing.T) {
		trans := &Transactional{}
		trans.On("GetGamePlayerByID", ctx, gamePlayerID).
			Return(data.GamePlayer{}, sql.ErrNoRows)

		application := letterblock.NewApplication(trans, make(map[string]dictionary.Dictionary))
		_, err := application.TakeTurn(ctx, gamePlayerID, playerID, word)
		assert.EqualError(t, err, sql.ErrNoRows.Error())
	})
	t.Run("ErrorUnauthorized", func(t *testing.T) {
		trans := &Transactional{}
		trans.On("GetGamePlayerByID", ctx, gamePlayerID).
			Return(data.GamePlayer{GameID: gameID, PlayerID: playerID + 1}, nil)

		application := letterblock.NewApplication(trans, make(map[string]dictionary.Dictionary))
		_, err := application.TakeTurn(ctx, gamePlayerID, playerID, word)
		assert.EqualError(t, err, letterblock.ErrorUnauthorized.Error())
	})
	t.Run("ErrorBeginTransaction", func(t *testing.T) {
		trans := &Transactional{}
		trans.On("GetGamePlayerByID", ctx, gamePlayerID).
			Return(data.GamePlayer{GameID: gameID, PlayerID: playerID}, nil)
		trans.On("BeginTransaction", ctx).
			Return(&sql.Tx{}, sql.ErrConnDone)

		application := letterblock.NewApplication(trans, make(map[string]dictionary.Dictionary))
		_, err := application.TakeTurn(ctx, gamePlayerID, playerID, word)
		assert.EqualError(t, err, sql.ErrConnDone.Error())
	})
	t.Run("ErrorGetGameByID", func(t *testing.T) {
		trans := &Transactional{}
		trans.On("GetGamePlayerByID", ctx, gamePlayerID).
			Return(data.GamePlayer{GameID: gameID, PlayerID: playerID}, nil)
		trans.On("BeginTransaction", ctx).
			Return(tx, nil)
		trans.On("GetGameByID", ctx, tx, gameID).
			Return(data.Game{}, sql.ErrConnDone)
		trans.On("FinalizeTransaction", tx, sql.ErrConnDone).
			Return(nil)

		application := letterblock.NewApplication(trans, make(map[string]dictionary.Dictionary))
		_, err := application.TakeTurn(ctx, gamePlayerID, playerID, word)
		assert.EqualError(t, err, sql.ErrConnDone.Error())
	})
	t.Run("ErrorGameIsUnplayable", func(t *testing.T) {
		testSuite := func(t *testing.T, state data.GameState) {
			trans := &Transactional{}
			trans.On("GetGamePlayerByID", ctx, gamePlayerID).
				Return(data.GamePlayer{GameID: gameID, PlayerID: playerID, Ordering: 1}, nil)
			trans.On("BeginTransaction", ctx).
				Return(tx, nil)
			trans.On("GetGameByID", ctx, tx, gameID).
				Return(data.Game{
					CurrentOrder: 2,
					BoardBase:    boardBase,
					State:        state,
				}, nil)
			trans.On("FinalizeTransaction", tx, letterblock.ErrorGameIsUnplayable).
				Return(nil)

			application := letterblock.NewApplication(trans, make(map[string]dictionary.Dictionary))
			_, err := application.TakeTurn(ctx, gamePlayerID, playerID, word)
			assert.EqualError(t, err, letterblock.ErrorGameIsUnplayable.Error())
		}
		t.Run("Created", func(t *testing.T) {
			testSuite(t, data.CREATED)
		})
		t.Run("End", func(t *testing.T) {
			testSuite(t, data.END)
		})
	})
	t.Run("ErrorNotYourTurn", func(t *testing.T) {
		trans := &Transactional{}
		trans.On("GetGamePlayerByID", ctx, gamePlayerID).
			Return(data.GamePlayer{GameID: gameID, PlayerID: playerID, Ordering: 1}, nil)
		trans.On("BeginTransaction", ctx).
			Return(tx, nil)
		trans.On("GetGameByID", ctx, tx, gameID).
			Return(data.Game{CurrentOrder: 2, BoardBase: boardBase, State: data.ONGOING}, nil)
		trans.On("FinalizeTransaction", tx, letterblock.ErrorNotYourTurn).
			Return(nil)

		application := letterblock.NewApplication(trans, make(map[string]dictionary.Dictionary))
		_, err := application.TakeTurn(ctx, gamePlayerID, playerID, word)
		assert.EqualError(t, err, letterblock.ErrorNotYourTurn.Error())
	})
	t.Run("ErrorDoesntMakeWord", func(t *testing.T) {
		trans := &Transactional{}
		trans.On("GetGamePlayerByID", ctx, gamePlayerID).
			Return(data.GamePlayer{GameID: gameID, PlayerID: playerID, Ordering: 1}, nil)
		trans.On("BeginTransaction", ctx).
			Return(tx, nil)
		trans.On("GetGameByID", ctx, tx, gameID).
			Return(data.Game{CurrentOrder: 1, BoardBase: boardBase, State: data.ONGOING}, nil)
		trans.On("FinalizeTransaction", tx, letterblock.ErrorDoesntMakeWord).
			Return(nil)

		application := letterblock.NewApplication(trans, make(map[string]dictionary.Dictionary))
		_, err := application.TakeTurn(ctx, gamePlayerID, playerID, append(word, word[0]))
		assert.EqualError(t, err, letterblock.ErrorDoesntMakeWord.Error())
	})
	t.Run("ErrorValidatingLemma", func(t *testing.T) {
		trans := &Transactional{}
		trans.On("GetGamePlayerByID", ctx, gamePlayerID).
			Return(data.GamePlayer{GameID: gameID, PlayerID: playerID, Ordering: 1}, nil)
		trans.On("BeginTransaction", ctx).
			Return(tx, nil)
		trans.On("GetGameByID", ctx, tx, gameID).
			Return(data.Game{CurrentOrder: 1, BoardBase: boardBase, State: data.ONGOING}, nil)
		unexpectedError := errors.New("unexpected error")
		trans.On("FinalizeTransaction", tx, unexpectedError).
			Return(nil)

		dict := &Dictionary{}

		dict.On("LemmaIsValid", "word").
			Return(false, unexpectedError)

		application := letterblock.NewApplication(trans, map[string]dictionary.Dictionary{
			"id-id": dict,
		})
		_, err := application.TakeTurn(ctx, gamePlayerID, playerID, word)
		assert.EqualError(t, err, unexpectedError.Error())
	})
	t.Run("ErrorWordInvalid", func(t *testing.T) {
		trans := &Transactional{}
		trans.On("GetGamePlayerByID", ctx, gamePlayerID).
			Return(data.GamePlayer{GameID: gameID, PlayerID: playerID, Ordering: 1}, nil)
		trans.On("BeginTransaction", ctx).
			Return(tx, nil)
		trans.On("GetGameByID", ctx, tx, gameID).
			Return(data.Game{CurrentOrder: 1, BoardBase: boardBase, State: data.ONGOING}, nil)
		trans.On("FinalizeTransaction", tx, letterblock.ErrorWordInvalid).
			Return(nil)

		dict := &Dictionary{}

		dict.On("LemmaIsValid", "word").
			Return(false, nil)

		application := letterblock.NewApplication(trans, map[string]dictionary.Dictionary{
			"id-id": dict,
		})
		_, err := application.TakeTurn(ctx, gamePlayerID, playerID, word)
		assert.EqualError(t, err, letterblock.ErrorWordInvalid.Error())
	})
	t.Run("ErrorLogPlayedWord", func(t *testing.T) {
		t.Run("Unexpected", func(t *testing.T) {
			trans := &Transactional{}
			trans.On("GetGamePlayerByID", ctx, gamePlayerID).
				Return(data.GamePlayer{GameID: gameID, PlayerID: playerID, Ordering: 1}, nil)
			trans.On("BeginTransaction", ctx).
				Return(tx, nil)
			trans.On("GetGameByID", ctx, tx, gameID).
				Return(data.Game{CurrentOrder: 1, BoardBase: boardBase, State: data.ONGOING}, nil)
			unexpectedError := errors.New("unexpected error")
			trans.On("LogPlayedWord", ctx, tx, gameID, playerID).
				Return(unexpectedError)
			trans.On("FinalizeTransaction", tx, unexpectedError).
				Return(nil)

			dict := &Dictionary{}

			dict.On("LemmaIsValid", "word").
				Return(true, nil)

			application := letterblock.NewApplication(trans, map[string]dictionary.Dictionary{
				"id-id": dict,
			})
			_, err := application.TakeTurn(ctx, gamePlayerID, playerID, word)
			assert.EqualError(t, err, unexpectedError.Error())
		})
		t.Run("WordHavePlayed", func(t *testing.T) {
			trans := &Transactional{}
			trans.On("GetGamePlayerByID", ctx, gamePlayerID).
				Return(data.GamePlayer{GameID: gameID, PlayerID: playerID, Ordering: 1}, nil)
			trans.On("BeginTransaction", ctx).
				Return(tx, nil)
			trans.On("GetGameByID", ctx, tx, gameID).
				Return(data.Game{CurrentOrder: 1, BoardBase: boardBase, State: data.ONGOING}, nil)
			trans.On("LogPlayedWord", ctx, tx, gameID, playerID).
				Return(errors.New("---Error 2601---"))
			trans.On("FinalizeTransaction", tx, letterblock.ErrorWordHavePlayed).
				Return(nil)

			dict := &Dictionary{}

			dict.On("LemmaIsValid", "word").
				Return(true, nil)

			application := letterblock.NewApplication(trans, map[string]dictionary.Dictionary{
				"id-id": dict,
			})
			_, err := application.TakeTurn(ctx, gamePlayerID, playerID, word)
			assert.EqualError(t, err, letterblock.ErrorWordHavePlayed.Error())
		})
	})
	t.Run("ErrorGetGamePlayersByGameID", func(t *testing.T) {
		trans := &Transactional{}
		trans.On("GetGamePlayerByID", ctx, gamePlayerID).
			Return(data.GamePlayer{GameID: gameID, PlayerID: playerID, Ordering: 1}, nil)
		trans.On("BeginTransaction", ctx).
			Return(tx, nil)
		trans.On("GetGameByID", ctx, tx, gameID).Return(data.Game{
			CurrentOrder: 1, BoardBase: boardBase, BoardPositioning: make([]uint8, 25), MaxStrength: maxStrength,
			State: data.ONGOING,
		}, nil)
		trans.On("LogPlayedWord", ctx, tx, gameID, playerID).
			Return(nil)
		unexpectedError := errors.New("unexpected error")
		trans.On("GetGamePlayersByGameID", ctx, tx, gameID).
			Return([]data.GamePlayer{}, unexpectedError)
		trans.On("FinalizeTransaction", tx, unexpectedError).
			Return(nil)

		dict := &Dictionary{}

		dict.On("LemmaIsValid", "word").
			Return(true, nil)

		application := letterblock.NewApplication(trans, map[string]dictionary.Dictionary{
			"id-id": dict,
		})
		_, err := application.TakeTurn(ctx, gamePlayerID, playerID, word)
		assert.EqualError(t, err, unexpectedError.Error())
	})
	t.Run("Positioning", func(t *testing.T) {
		positioningSuite := func(boardPositioning, expectedBoardPositioning []uint8) {
			trans := &Transactional{}
			trans.On("GetGamePlayerByID", ctx, gamePlayerID).
				Return(data.GamePlayer{GameID: gameID, PlayerID: playerID, Ordering: 1}, nil)
			trans.On("BeginTransaction", ctx).
				Return(tx, nil)
			trans.On("GetGameByID", ctx, tx, gameID).Return(data.Game{
				CurrentOrder: 1, BoardBase: boardBase, BoardPositioning: boardPositioning, MaxStrength: maxStrength,
				State: data.ONGOING,
			}, nil)
			trans.On("LogPlayedWord", ctx, tx, gameID, playerID).
				Return(nil)
			trans.On("GetGamePlayersByGameID", ctx, tx, gameID).
				Return([]data.GamePlayer{
					{GameID: gameID, PlayerID: playerID, Ordering: 1},
					{GameID: gameID, PlayerID: players[1].ID, Ordering: 2},
				}, nil)
			trans.On("UpdateGame").
				Return(nil)
			trans.On("FinalizeTransaction", tx, nil).
				Return(nil)

			dict := &Dictionary{}

			dict.On("LemmaIsValid", "worde").
				Return(true, nil)

			application := letterblock.NewApplication(trans, map[string]dictionary.Dictionary{
				"id-id": dict,
			})
			game, err := application.TakeTurn(ctx, gamePlayerID, playerID, []uint8{0, 1, 2, 3, 4})
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
		orderingSuite := func(currentPlayer data.GamePlayer, nextOrder uint8) {
			trans := &Transactional{}
			trans.On("GetGamePlayerByID", ctx, gamePlayerID).
				Return(data.GamePlayer{GameID: gameID, PlayerID: currentPlayer.PlayerID, Ordering: currentPlayer.Ordering}, nil)
			trans.On("BeginTransaction", ctx).
				Return(tx, nil)
			trans.On("GetGameByID", ctx, tx, gameID).Return(data.Game{
				CurrentOrder: currentPlayer.Ordering, BoardBase: boardBase, BoardPositioning: make([]uint8, 25), MaxStrength: maxStrength,
				State: data.ONGOING,
			}, nil)
			trans.On("LogPlayedWord", ctx, tx, gameID, playerID).
				Return(nil)
			trans.On("GetGamePlayersByGameID", ctx, tx, gameID).
				Return([]data.GamePlayer{{}, {}}, nil)
			trans.On("UpdateGame").
				Return(nil)
			trans.On("FinalizeTransaction", tx, nil).
				Return(nil)

			dict := &Dictionary{}

			dict.On("LemmaIsValid", "worde").
				Return(true, nil)

			application := letterblock.NewApplication(trans, map[string]dictionary.Dictionary{
				"id-id": dict,
			})
			game, err := application.TakeTurn(ctx, gamePlayerID, playerID, []uint8{0, 1, 2, 3, 4})
			if assert.NoError(t, err) {
				assert.Equal(t, nextOrder, game.CurrentOrder)
			}
		}
		t.Run("NotExceeding", func(t *testing.T) {
			orderingSuite(data.GamePlayer{
				PlayerID: playerID,
				Ordering: 1,
			}, 2)
		})
		t.Run("Exceeding", func(t *testing.T) {
			orderingSuite(data.GamePlayer{
				PlayerID: playerID,
				Ordering: 2,
			}, 1)
		})
	})
	t.Run("GameIsEnding", func(t *testing.T) {
		testSuite := func(t *testing.T, boardPositioning []uint8, expectedEnd bool) {
			trans := &Transactional{}
			trans.On("GetGamePlayerByID", ctx, gamePlayerID).
				Return(data.GamePlayer{GameID: gameID, PlayerID: playerID, Ordering: 1}, nil)
			trans.On("BeginTransaction", ctx).
				Return(tx, nil)
			trans.On("GetGameByID", ctx, tx, gameID).Return(data.Game{
				CurrentOrder: 1, BoardBase: boardBase, BoardPositioning: boardPositioning, MaxStrength: maxStrength,
				State: data.ONGOING,
			}, nil)
			trans.On("LogPlayedWord", ctx, tx, gameID, playerID).
				Return(nil)
			trans.On("GetGamePlayersByGameID", ctx, tx, gameID).
				Return([]data.GamePlayer{
					{GameID: gameID, PlayerID: playerID, Ordering: 1},
					{GameID: gameID, PlayerID: players[1].ID, Ordering: 2},
				}, nil)
			trans.On("UpdateGame").
				Return(nil)
			trans.On("FinalizeTransaction", tx, nil).
				Return(nil)

			dict := &Dictionary{}

			dict.On("LemmaIsValid", "worde").
				Return(true, nil)

			application := letterblock.NewApplication(trans, map[string]dictionary.Dictionary{
				"id-id": dict,
			})
			game, err := application.TakeTurn(ctx, gamePlayerID, playerID, []uint8{0, 1, 2, 3, 4})
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
		trans.On("GetGamePlayerByID", ctx, gamePlayerID).
			Return(data.GamePlayer{GameID: gameID, PlayerID: playerID, Ordering: 1}, nil)
		trans.On("BeginTransaction", ctx).
			Return(tx, nil)
		trans.On("GetGameByID", ctx, tx, gameID).Return(data.Game{
			CurrentOrder: 1, BoardBase: boardBase, BoardPositioning: make([]uint8, 25), MaxStrength: maxStrength,
			State: data.ONGOING,
		}, nil)
		trans.On("LogPlayedWord", ctx, tx, gameID, playerID).
			Return(nil)
		trans.On("GetGamePlayersByGameID", ctx, tx, gameID).
			Return([]data.GamePlayer{{}, {}}, nil)
		unexpectedError := errors.New("unexpected error")
		trans.On("UpdateGame").
			Return(unexpectedError)
		trans.On("FinalizeTransaction", tx, unexpectedError).
			Return(nil)

		dict := &Dictionary{}

		dict.On("LemmaIsValid", "word").
			Return(true, nil)

		application := letterblock.NewApplication(trans, map[string]dictionary.Dictionary{
			"id-id": dict,
		})
		_, err := application.TakeTurn(ctx, gamePlayerID, playerID, word)
		assert.EqualError(t, err, unexpectedError.Error())
	})
}
