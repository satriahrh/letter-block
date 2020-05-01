package letter_block_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/satriahrh/letter-block"
	"github.com/satriahrh/letter-block/data"
	"github.com/satriahrh/letter-block/dictionary"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var (
	gameId = data.GameId(time.Now().UnixNano())

	// len(usernames) >= 2
	usernames = []string{"sarjono", "mukti"}

	players = []data.Player{
		{Id: data.PlayerId(time.Now().UnixNano()), Username: usernames[0]},
		{Id: data.PlayerId(time.Now().UnixNano()), Username: usernames[1]},
	}

	gamePlayers = []data.GamePlayer{
		{Id: data.GamePlayerId(time.Now().UnixNano()), PlayerId: players[0].Id, GameId: gameId},
		{Id: data.GamePlayerId(time.Now().UnixNano()), PlayerId: players[1].Id, GameId: gameId},
	}

	numberOfPlayer = uint8(5)

	playerId = players[0].Id

	gamePlayerId = gamePlayers[0].Id

	word      = []uint8{0, 1, 2, 3}
	boardBase = []uint8{22, 14, 17, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24}

	unexpectedError = errors.New("unexpected error")

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

func (t *Transactional) InsertGamePlayer(ctx context.Context, tx *sql.Tx, game data.Game, player data.Player) (data.Game, error) {
	args := t.Called(ctx, tx, game, player)
	err := args.Error(0)
	if err != nil {
		game = data.Game{}
	}
	game.Players = []data.Player{player}
	return game, err
}

func (t *Transactional) GetPlayerById(ctx context.Context, playerId data.PlayerId) (player data.Player, err error) {
	args := t.Called(playerId)
	player = args.Get(0).(data.Player)
	err = args.Error(1)
	player.Id = playerId
	return
}

func (t *Transactional) GetGameById(ctx context.Context, tx *sql.Tx, gameId data.GameId) (game data.Game, err error) {
	args := t.Called(ctx, tx, gameId)
	game = args.Get(0).(data.Game)
	err = args.Error(1)
	game.Id = gameId
	return
}

func (t *Transactional) GetGamePlayersByGameId(ctx context.Context, tx *sql.Tx, gameId data.GameId) (gamePlayers []data.GamePlayer, err error) {
	args := t.Called(ctx, tx, gameId)
	gamePlayers = args.Get(0).([]data.GamePlayer)
	err = args.Error(1)
	return
}

func (t *Transactional) LogPlayedWord(ctx context.Context, tx *sql.Tx, gameId data.GameId, playerId data.PlayerId, word string) error {
	return t.Called(ctx, tx, gameId, playerId).Error(0)
}

func (t *Transactional) UpdateGame(ctx context.Context, tx *sql.Tx, game data.Game) error {
	return t.Called().Error(0)
}

func TestApplicationNewGame(t *testing.T) {
	t.Run("ErrorNumberOfPlayer", func(t *testing.T) {
		testSuite := func(t *testing.T, sample uint8) {
			application := letter_block.NewApplication(&Transactional{}, make(map[string]dictionary.Dictionary))
			_, err := application.NewGame(ctx, playerId, sample)
			assert.EqualError(t, err, letter_block.ErrorNumberOfPlayer.Error())
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

		application := letter_block.NewApplication(trans, make(map[string]dictionary.Dictionary))
		_, err := application.NewGame(ctx, playerId, numberOfPlayer)
		assert.EqualError(t, err, sql.ErrNoRows.Error())
	})
	t.Run("ErrorBeginTransaction", func(t *testing.T) {
		trans := &Transactional{}
		trans.On("GetPlayerById", playerId).
			Return(players[0], nil)
		trans.On("BeginTransaction", ctx).
			Return(&sql.Tx{}, sql.ErrConnDone)

		application := letter_block.NewApplication(trans, make(map[string]dictionary.Dictionary))
		_, err := application.NewGame(ctx, playerId, numberOfPlayer)
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

		application := letter_block.NewApplication(trans, make(map[string]dictionary.Dictionary))
		_, err := application.NewGame(ctx, playerId, numberOfPlayer)
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

		application := letter_block.NewApplication(trans, make(map[string]dictionary.Dictionary))
		_, err := application.NewGame(ctx, playerId, numberOfPlayer)
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

			application := letter_block.NewApplication(trans, make(map[string]dictionary.Dictionary))
			return application.NewGame(ctx, playerId, numberOfPlayer)
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

func TestApplicationTakeTurn(t *testing.T) {
	t.Run("ErrorBeginTransaction", func(t *testing.T) {
		trans := &Transactional{}
		trans.On("BeginTransaction", ctx).
			Return(&sql.Tx{}, sql.ErrConnDone)

		application := letter_block.NewApplication(trans, make(map[string]dictionary.Dictionary))
		_, err := application.TakeTurn(ctx, gameId, playerId, word)
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

		application := letter_block.NewApplication(trans, make(map[string]dictionary.Dictionary))
		_, err := application.TakeTurn(ctx, gameId, playerId, word)
		assert.EqualError(t, err, sql.ErrConnDone.Error())
	})
	t.Run("ErrorGameIsUnplayable", func(t *testing.T) {
		testSuite := func(t *testing.T, state data.GameState) {
			trans := &Transactional{}
			trans.On("BeginTransaction", ctx).
				Return(tx, nil)
			trans.On("GetGameById", ctx, tx, gameId).
				Return(data.Game{
					CurrentPlayerOrder: 2, BoardBase: boardBase, State: state,
				}, nil)
			trans.On("FinalizeTransaction", tx, letter_block.ErrorGameIsUnplayable).
				Return(nil)

			application := letter_block.NewApplication(trans, make(map[string]dictionary.Dictionary))
			_, err := application.TakeTurn(ctx, gameId, playerId, word)
			assert.EqualError(t, err, letter_block.ErrorGameIsUnplayable.Error())
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
				BoardBase: boardBase, State: data.ONGOING,
			}, nil)
		trans.On("GetGamePlayersByGameId", ctx, tx, gameId).
			Return([]data.GamePlayer{}, sql.ErrConnDone)
		trans.On("FinalizeTransaction", tx, sql.ErrConnDone).
			Return(nil)

		application := letter_block.NewApplication(trans, make(map[string]dictionary.Dictionary))
		_, err := application.TakeTurn(ctx, gameId, playerId, word)
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
					BoardBase: boardBase, State: data.ONGOING,
				}, nil)
			trans.On("GetGamePlayersByGameId", ctx, tx, gameId).
				Return(gamePlayers, nil)
			trans.On("FinalizeTransaction", tx, letter_block.ErrorNotYourTurn).
				Return(nil)

			application := letter_block.NewApplication(trans, make(map[string]dictionary.Dictionary))
			_, err := application.TakeTurn(ctx, gameId, playerId, word)
			assert.EqualError(t, err, letter_block.ErrorNotYourTurn.Error())
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
				BoardBase: boardBase, State: data.ONGOING,
			}, nil)
		trans.On("GetGamePlayersByGameId", ctx, tx, gameId).
			Return([]data.GamePlayer{
				{GameId: gameId, PlayerId: playerId},
				{GameId: gameId, PlayerId: players[1].Id},
			}, nil)
		trans.On("FinalizeTransaction", tx, letter_block.ErrorDoesntMakeWord).
			Return(nil)

		application := letter_block.NewApplication(trans, make(map[string]dictionary.Dictionary))
		_, err := application.TakeTurn(ctx, gameId, playerId, append(word, word[0]))
		assert.EqualError(t, err, letter_block.ErrorDoesntMakeWord.Error())
	})
	t.Run("ErrorValidatingLemma", func(t *testing.T) {
		trans := &Transactional{}
		trans.On("BeginTransaction", ctx).
			Return(tx, nil)
		trans.On("GetGameById", ctx, tx, gameId).
			Return(data.Game{
				CurrentPlayerOrder: 0, NumberOfPlayer: 2,
				BoardBase: boardBase, State: data.ONGOING,
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

		application := letter_block.NewApplication(trans, map[string]dictionary.Dictionary{
			"id-id": dict,
		})
		_, err := application.TakeTurn(ctx, gameId, playerId, word)
		assert.EqualError(t, err, unexpectedError.Error())
	})
	t.Run("ErrorWordInvalid", func(t *testing.T) {
		trans := &Transactional{}
		trans.On("BeginTransaction", ctx).
			Return(tx, nil)
		trans.On("GetGameById", ctx, tx, gameId).
			Return(data.Game{
				CurrentPlayerOrder: 0, NumberOfPlayer: 2,
				BoardBase: boardBase, State: data.ONGOING,
			}, nil)
		trans.On("GetGamePlayersByGameId", ctx, tx, gameId).
			Return([]data.GamePlayer{
				{GameId: gameId, PlayerId: playerId},
				{GameId: gameId, PlayerId: players[1].Id},
			}, nil)
		trans.On("FinalizeTransaction", tx, letter_block.ErrorWordInvalid).
			Return(nil)

		dict := &Dictionary{}

		dict.On("LemmaIsValid", "word").
			Return(false, nil)

		application := letter_block.NewApplication(trans, map[string]dictionary.Dictionary{
			"id-id": dict,
		})
		_, err := application.TakeTurn(ctx, gameId, playerId, word)
		assert.EqualError(t, err, letter_block.ErrorWordInvalid.Error())
	})
	t.Run("ErrorLogPlayedWord", func(t *testing.T) {
		t.Run("Unexpected", func(t *testing.T) {
			trans := &Transactional{}
			trans.On("BeginTransaction", ctx).
				Return(tx, nil)
			trans.On("GetGameById", ctx, tx, gameId).
				Return(data.Game{
					CurrentPlayerOrder: 0, NumberOfPlayer: 2,
					BoardBase: boardBase, State: data.ONGOING,
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

			application := letter_block.NewApplication(trans, map[string]dictionary.Dictionary{
				"id-id": dict,
			})
			_, err := application.TakeTurn(ctx, gameId, playerId, word)
			assert.EqualError(t, err, unexpectedError.Error())
		})
		t.Run("WordHavePlayed", func(t *testing.T) {
			trans := &Transactional{}
			trans.On("BeginTransaction", ctx).
				Return(tx, nil)
			trans.On("GetGameById", ctx, tx, gameId).
				Return(data.Game{
					CurrentPlayerOrder: 0, NumberOfPlayer: 2,
					BoardBase: boardBase, State: data.ONGOING,
				}, nil)
			trans.On("GetGamePlayersByGameId", ctx, tx, gameId).
				Return([]data.GamePlayer{
					{GameId: gameId, PlayerId: playerId},
					{GameId: gameId, PlayerId: players[1].Id},
				}, nil)
			trans.On("LogPlayedWord", ctx, tx, gameId, playerId).
				Return(errors.New("---Error 2601---"))
			trans.On("FinalizeTransaction", tx, letter_block.ErrorWordHavePlayed).
				Return(nil)

			dict := &Dictionary{}

			dict.On("LemmaIsValid", "word").
				Return(true, nil)

			application := letter_block.NewApplication(trans, map[string]dictionary.Dictionary{
				"id-id": dict,
			})
			_, err := application.TakeTurn(ctx, gameId, playerId, word)
			assert.EqualError(t, err, letter_block.ErrorWordHavePlayed.Error())
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
					BoardBase: boardBase, State: data.ONGOING,
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

			dict.On("LemmaIsValid", "worde").
				Return(true, nil)

			application := letter_block.NewApplication(trans, map[string]dictionary.Dictionary{
				"id-id": dict,
			})
			game, err := application.TakeTurn(ctx, gameId, playerId, []uint8{0, 1, 2, 3, 4})
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
					BoardBase: boardBase, State: data.ONGOING,
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

			dict.On("LemmaIsValid", "worde").
				Return(true, nil)

			application := letter_block.NewApplication(trans, map[string]dictionary.Dictionary{
				"id-id": dict,
			})
			game, err := application.TakeTurn(ctx, gameId, players[currentPlayerOrder].Id, []uint8{0, 1, 2, 3, 4})
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
					BoardBase: boardBase, State: data.ONGOING,
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

			dict.On("LemmaIsValid", "worde").
				Return(true, nil)

			application := letter_block.NewApplication(trans, map[string]dictionary.Dictionary{
				"id-id": dict,
			})
			game, err := application.TakeTurn(ctx, gameId, playerId, []uint8{0, 1, 2, 3, 4})
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
				BoardBase: boardBase, State: data.ONGOING,
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

		application := letter_block.NewApplication(trans, map[string]dictionary.Dictionary{
			"id-id": dict,
		})
		_, err := application.TakeTurn(ctx, gameId, playerId, word)
		assert.EqualError(t, err, unexpectedError.Error())
	})
}

func TestApplication_Join(t *testing.T) {
	t.Run("ErrorBeginTransaction", func(t *testing.T) {
		trans := &Transactional{}
		trans.On("BeginTransaction", ctx).
			Return(&sql.Tx{}, sql.ErrConnDone)

		application := letter_block.NewApplication(trans, make(map[string]dictionary.Dictionary))
		_, err := application.Join(ctx, gameId, playerId)
		assert.EqualError(t, err, sql.ErrConnDone.Error())
	})
	t.Run("ErrorGetGameById", func(t *testing.T) {
		trans := &Transactional{}
		trans.On("BeginTransaction", ctx).
			Return(tx, nil)
		trans.On("GetGameById", ctx, tx, gameId).
			Return(data.Game{}, sql.ErrNoRows)
		trans.On("FinalizeTransaction", tx, sql.ErrNoRows).
			Return(nil)

		application := letter_block.NewApplication(trans, make(map[string]dictionary.Dictionary))
		_, err := application.Join(ctx, gameId, playerId)
		assert.EqualError(t, err, sql.ErrNoRows.Error())
	})
	t.Run("ErrorGetPlayerById", func(t *testing.T) {
		trans := &Transactional{}
		trans.On("BeginTransaction", ctx).
			Return(tx, nil)
		trans.On("GetGameById", ctx, tx, gameId).
			Return(data.Game{
				CurrentPlayerOrder: 1, NumberOfPlayer: 2,
				BoardBase: boardBase, State: data.ONGOING,
			}, nil)
		trans.On("GetPlayerById", playerId).
			Return(data.Player{}, sql.ErrNoRows)
		trans.On("FinalizeTransaction", tx, sql.ErrNoRows).
			Return(nil)

		application := letter_block.NewApplication(trans, make(map[string]dictionary.Dictionary))
		_, err := application.Join(ctx, gameId, playerId)
		assert.EqualError(t, err, sql.ErrNoRows.Error())
	})
	t.Run("ErrorGetGamePlayersByGameId", func(t *testing.T) {
		trans := &Transactional{}
		trans.On("BeginTransaction", ctx).
			Return(tx, nil)
		trans.On("GetGameById", ctx, tx, gameId).
			Return(data.Game{
				CurrentPlayerOrder: 1, NumberOfPlayer: 2,
				BoardBase: boardBase, State: data.ONGOING,
			}, nil)
		trans.On("GetPlayerById", playerId).
			Return(players[1], nil)
		trans.On("GetGamePlayersByGameId", ctx, tx, gameId).
			Return([]data.GamePlayer{}, sql.ErrConnDone)
		trans.On("FinalizeTransaction", tx, sql.ErrConnDone).
			Return(nil)

		application := letter_block.NewApplication(trans, make(map[string]dictionary.Dictionary))
		_, err := application.Join(ctx, gameId, playerId)
		assert.EqualError(t, err, sql.ErrConnDone.Error())
	})
	t.Run("ErrorPlayerIsEnough", func(t *testing.T) {
		trans := &Transactional{}
		trans.On("BeginTransaction", ctx).
			Return(tx, nil)
		trans.On("GetGameById", ctx, tx, gameId).
			Return(data.Game{
				CurrentPlayerOrder: 1, NumberOfPlayer: uint8(len(gamePlayers)),
				BoardBase: boardBase, State: data.ONGOING,
			}, nil)
		trans.On("GetPlayerById", playerId).
			Return(players[1], nil)
		trans.On("GetGamePlayersByGameId", ctx, tx, gameId).
			Return(gamePlayers, nil)
		trans.On("FinalizeTransaction", tx, letter_block.ErrorPlayerIsEnough).
			Return(nil)

		application := letter_block.NewApplication(trans, make(map[string]dictionary.Dictionary))
		_, err := application.Join(ctx, gameId, playerId)
		assert.EqualError(t, err, letter_block.ErrorPlayerIsEnough.Error())
	})
	t.Run("ErrorInsertGamePlayer", func(t *testing.T) {
		game := data.Game{
			Id: gameId, CurrentPlayerOrder: 1, NumberOfPlayer: uint8(len(gamePlayers)),
			BoardBase: boardBase, State: data.ONGOING,
		}
		trans := &Transactional{}
		trans.On("BeginTransaction", ctx).
			Return(tx, nil)
		trans.On("GetGameById", ctx, tx, gameId).
			Return(game, nil)
		trans.On("GetPlayerById", playerId).
			Return(players[1], nil)
		trans.On("GetGamePlayersByGameId", ctx, tx, gameId).
			Return(gamePlayers[:0], nil)
		trans.On("InsertGamePlayer", ctx, tx, game, players[1]).
			Return(unexpectedError)
		trans.On("FinalizeTransaction", tx, unexpectedError).
			Return(nil)

		application := letter_block.NewApplication(trans, make(map[string]dictionary.Dictionary))
		_, err := application.Join(ctx, gameId, playerId)
		assert.EqualError(t, err, unexpectedError.Error())
	})
	t.Run("Success", func(t *testing.T) {
		game := data.Game{
			Id: gameId, CurrentPlayerOrder: 1, NumberOfPlayer: uint8(len(gamePlayers)),
			BoardBase: boardBase, State: data.ONGOING,
		}
		trans := &Transactional{}
		trans.On("BeginTransaction", ctx).
			Return(tx, nil)
		trans.On("GetGameById", ctx, tx, gameId).
			Return(game, nil)
		trans.On("GetPlayerById", players[1].Id).
			Return(players[1], nil)
		trans.On("GetGamePlayersByGameId", ctx, tx, gameId).
			Return(gamePlayers[:1], nil)
		trans.On("InsertGamePlayer", ctx, tx, game, players[1]).
			Return(nil)
		trans.On("FinalizeTransaction", tx, nil).
			Return(nil)

		application := letter_block.NewApplication(trans, make(map[string]dictionary.Dictionary))
		game, err := application.Join(ctx, gameId, players[1].Id)
		assert.NoError(t, err)
	})
}
