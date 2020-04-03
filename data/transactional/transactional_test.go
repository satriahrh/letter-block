package transactional_test

import (
	"github.com/satriahrh/letter-block/data"
	"github.com/satriahrh/letter-block/data/transactional"

	"context"
	"database/sql"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jinzhu/copier"
	"github.com/stretchr/testify/assert"
)

type Preparation struct {
	sqlMock       sqlmock.Sqlmock
	ctx           context.Context
	transactional *transactional.Transactional
	tx            func(func()) *sql.Tx
}

var (
	usernames = []string{"sarjono", "mukti"}
	players   = []data.Player{
		{Id: uint64(time.Now().UnixNano()), Username: usernames[0]},
		{Id: uint64(time.Now().UnixNano()), Username: usernames[1]},
	}
	gameId          = uint64(time.Now().UnixNano())
	playerId        = players[0].Id
	gamePlayerId    = uint64(time.Now().UnixNano())
	currentPlayerId = playerId
	boardBase       = []uint8{22, 14, 17, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24}
	boarPositioning = make([]uint8, 25)
	maxStrength     = uint8(2)
	wordString      = "word"
)

var (
	gameColumn       = []string{"current_player_id", "board_base"}
	gamePlayerColumn = []string{"game_id", "player_id"}
	playerColumn     = []string{"id", "username"}
)

func testPreparation(t *testing.T) Preparation {
	ctx := context.TODO()
	db, sqlMock, err := sqlmock.New()
	if !assert.NoError(t, err, "sqlmock") {
		t.FailNow()
	}
	trans := transactional.NewTransactional(db)

	beginTx := func(expectation func()) *sql.Tx {
		sqlMock.ExpectBegin()
		tx, _ := db.Begin()

		expectation()
		return tx
	}

	return Preparation{sqlMock, ctx, trans, beginTx}
}

func TestTransactional_BeginTransaction(t *testing.T) {
	t.Run("ErrorBeginTrx", func(t *testing.T) {
		prep := testPreparation(t)

		unexpectedError := errors.New("unexpected error")
		prep.sqlMock.ExpectBegin().
			WillReturnError(unexpectedError)

		_, err := prep.transactional.BeginTransaction(prep.ctx)
		assert.EqualError(t, err, unexpectedError.Error(), "unexpected error")
	})
	t.Run("Success", func(t *testing.T) {
		prep := testPreparation(t)

		prep.sqlMock.ExpectBegin()

		tx, err := prep.transactional.BeginTransaction(prep.ctx)
		if assert.NoError(t, err, "no error") {
			assert.NotEmpty(t, tx, "return non empty transaction")
		}

	})
}

func TestTransactional_FinalizeTransaction(t *testing.T) {
	t.Run("ErrIsNotNill", func(t *testing.T) {
		unexpectedError := errors.New("unexpected error")
		unexpectedRollbackError := errors.New("unexpected rollback error")

		t.Run("ErrorRollbackTrx", func(t *testing.T) {
			prep := testPreparation(t)

			tx := prep.tx(func() {
				prep.sqlMock.ExpectRollback().
					WillReturnError(unexpectedRollbackError)
			})

			err := prep.transactional.FinalizeTransaction(tx, unexpectedError)
			assert.EqualError(t, err, unexpectedRollbackError.Error(), "unexpected rollback error")
		})
		t.Run("SuccessRollback", func(t *testing.T) {
			prep := testPreparation(t)

			tx := prep.tx(func() {
				prep.sqlMock.ExpectRollback()
			})

			err := prep.transactional.FinalizeTransaction(tx, unexpectedError)
			assert.EqualError(t, err, unexpectedError.Error(), "unexpected error")
		})
	})
	t.Run("Commit", func(t *testing.T) {
		t.Run("ReturnNilError", func(t *testing.T) {
			prep := testPreparation(t)

			tx := prep.tx(func() {
				prep.sqlMock.ExpectCommit()
			})

			err := prep.transactional.FinalizeTransaction(tx, nil)
			assert.NoError(t, err, "no error")
		})
		t.Run("ReturnError", func(t *testing.T) {
			unexpectedError := errors.New("unexpected error")
			prep := testPreparation(t)

			tx := prep.tx(func() {
				prep.sqlMock.ExpectCommit().
					WillReturnError(unexpectedError)
			})

			err := prep.transactional.FinalizeTransaction(tx, nil)
			assert.EqualError(t, err, unexpectedError.Error(), "commit return an error")
		})
	})
}

func TestTransactional_InsertGame(t *testing.T) {
	game := data.Game{
		CurrentPlayerId:  currentPlayerId,
		BoardBase:        boardBase,
		BoardPositioning: boarPositioning,
		MaxStrength:      maxStrength,
	}

	t.Run("ErrorExecContext", func(t *testing.T) {
		prep := testPreparation(t)

		unexpectedError := errors.New("unexpected error")
		tx := prep.tx(func() {
			prep.sqlMock.ExpectExec("INSERT INTO games").
				WithArgs(currentPlayerId, boardBase, boarPositioning, maxStrength).
				WillReturnError(unexpectedError)
		})

		_, err := prep.transactional.InsertGame(prep.ctx, tx, game)
		assert.EqualError(t, err, unexpectedError.Error(), "unexpected error")
	})
	t.Run("Success", func(t *testing.T) {
		prep := testPreparation(t)

		tx := prep.tx(func() {
			prep.sqlMock.ExpectExec("INSERT INTO games").
				WithArgs(currentPlayerId, boardBase, boarPositioning, maxStrength).
				WillReturnResult(sqlmock.NewResult(int64(gameId), 1))
		})

		game, err := prep.transactional.InsertGame(prep.ctx, tx, game)
		if assert.NoError(t, err) {
			assert.Equal(t, gameId, game.Id)
			assert.Equal(t, currentPlayerId, game.CurrentPlayerId)
			assert.Equal(t, boardBase, game.BoardBase)
			assert.Equal(t, boarPositioning, game.BoardPositioning)
			assert.Equal(t, maxStrength, game.MaxStrength)
			assert.Empty(t, game.Players)
		}
	})
}

func TestTransactional_InsertGamePlayerBulk(t *testing.T) {
	game := data.Game{
		Id: gameId,
	}

	t.Run("ErrorExecContext", func(t *testing.T) {
		t.Run("NoData", func(t *testing.T) {
			prep := testPreparation(t)

			unexpectedError := errors.New("unexpected error")
			tx := prep.tx(func() {
				prep.sqlMock.ExpectExec("INSERT INTO game_player").
					WithArgs().
					WillReturnError(unexpectedError)
			})

			_, err := prep.transactional.InsertGamePlayerBulk(prep.ctx, tx, game, []data.Player{})
			assert.EqualError(t, err, unexpectedError.Error(), "unexpected error")
		})
		t.Run("Unexpected", func(t *testing.T) {
			prep := testPreparation(t)

			unexpectedError := errors.New("unexpected error")
			tx := prep.tx(func() {
				prep.sqlMock.ExpectExec("INSERT INTO game_player").
					WithArgs(
						gameId, players[0].Id,
						gameId, players[1].Id,
					).
					WillReturnError(unexpectedError)
			})

			_, err := prep.transactional.InsertGamePlayerBulk(prep.ctx, tx, game, players)
			assert.EqualError(t, err, unexpectedError.Error(), "unexpected error")
		})
	})
	t.Run("Success", func(t *testing.T) {
		prep := testPreparation(t)

		tx := prep.tx(func() {
			prep.sqlMock.ExpectExec("INSERT INTO game_player").
				WithArgs(
					gameId, players[0].Id,
					gameId, players[1].Id,
				).
				WillReturnResult(sqlmock.NewResult(1, int64(len(players))))
		})

		var expectedGame data.Game
		_ = copier.Copy(&expectedGame, &game)
		expectedGame.Players = players

		actualGame, err := prep.transactional.InsertGamePlayerBulk(prep.ctx, tx, game, players)
		if assert.NoError(t, err) {
			assert.Equal(t, expectedGame, actualGame)
		}
	})
}

func TestTransactional_GetPlayersByUsernames(t *testing.T) {
	t.Run("ErrorQuerying", func(t *testing.T) {
		prep := testPreparation(t)

		unexpectedError := errors.New("unexpected error")
		prep.sqlMock.ExpectQuery("SELECT (.+) FROM players WHERE usernames IN").
			WithArgs(
				fmt.Sprintf(
					"('%v','%v')",
					usernames[0], usernames[1],
				),
			).
			WillReturnError(unexpectedError)

		_, err := prep.transactional.GetPlayersByUsernames(prep.ctx, usernames)
		assert.EqualError(t, err, unexpectedError.Error())
	})
	t.Run("ErrorScanning", func(t *testing.T) {
		prep := testPreparation(t)

		prep.sqlMock.ExpectQuery("SELECT (.+) FROM players WHERE usernames IN").
			WithArgs(
				fmt.Sprintf(
					"('%v','%v')",
					usernames[0], usernames[1],
				),
			).
			WillReturnRows(
				sqlmock.NewRows(playerColumn).
					AddRow(players[1].Username, players[1].Username),
			)

		_, err := prep.transactional.GetPlayersByUsernames(prep.ctx, usernames)
		assert.Error(t, err)
	})
	t.Run("NoPlayersFound", func(t *testing.T) {
		prep := testPreparation(t)

		prep.sqlMock.ExpectQuery("SELECT (.+) FROM players WHERE usernames IN").
			WithArgs(
				fmt.Sprintf(
					"('%v','%v')",
					usernames[0], usernames[1],
				),
			).
			WillReturnRows(
				sqlmock.NewRows(playerColumn),
			)

		players, err := prep.transactional.GetPlayersByUsernames(prep.ctx, usernames)
		assert.NoError(t, err)
		assert.Empty(t, players)
	})
	t.Run("ErrorScanning", func(t *testing.T) {
		prep := testPreparation(t)

		prep.sqlMock.ExpectQuery("SELECT (.+) FROM players WHERE usernames IN").
			WithArgs(
				fmt.Sprintf(
					"('%v','%v')",
					usernames[0], usernames[1],
				),
			).
			WillReturnRows(
				sqlmock.NewRows(playerColumn).
					AddRow(players[0].Id, players[0].Username).
					AddRow(players[1].Id, players[1].Username),
			)

		actualPlayers, err := prep.transactional.GetPlayersByUsernames(prep.ctx, usernames)
		if assert.NoError(t, err) {
			assert.Equal(t, players, actualPlayers)
		}

	})
}

func TestTransactional_GetGamePlayerById(t *testing.T) {
	t.Run("ErrorScanning", func(t *testing.T) {
		t.Run("DueErrorQuerying", func(t *testing.T) {
			prep := testPreparation(t)

			unexpectedError := errors.New("unexpected error")
			prep.sqlMock.ExpectQuery("SELECT (.+) FROM game_player").
				WithArgs(gamePlayerId).
				WillReturnError(unexpectedError)

			_, _, err := prep.transactional.GetGamePlayerById(prep.ctx, gamePlayerId)
			assert.EqualError(t, err, unexpectedError.Error(), "unexpected error")
		})
		t.Run("DueNoRow", func(t *testing.T) {
			prep := testPreparation(t)

			prep.sqlMock.ExpectQuery("SELECT (.+) FROM game_player").
				WithArgs(gamePlayerId).
				WillReturnRows(
					sqlmock.NewRows(gamePlayerColumn),
				)

			_, _, err := prep.transactional.GetGamePlayerById(prep.ctx, gamePlayerId)
			assert.EqualError(t, err, sql.ErrNoRows.Error(), "no row")
		})
	})
	t.Run("Success", func(t *testing.T) {
		prep := testPreparation(t)

		prep.sqlMock.ExpectQuery("SELECT (.+) FROM game_player").
			WithArgs(gamePlayerId).
			WillReturnRows(
				sqlmock.NewRows(gamePlayerColumn).
					AddRow(gameId, playerId),
			)

		actualGameId, actualPlayerId, err := prep.transactional.GetGamePlayerById(prep.ctx, gamePlayerId)
		if assert.NoError(t, err, "no error") {
			assert.Equal(t, gameId, actualGameId)
			assert.Equal(t, playerId, actualPlayerId)
		}
	})
}

func TestTransactional_GetGameById(t *testing.T) {
	t.Run("ErrorScanning", func(t *testing.T) {
		t.Run("DueErrorQuerying", func(t *testing.T) {
			prep := testPreparation(t)

			unexpectedError := errors.New("unexpected error")
			tx := prep.tx(func() {
				prep.sqlMock.ExpectQuery("SELECT (.+) FROM games").
					WithArgs(gameId).
					WillReturnError(unexpectedError)
			})

			_, err := prep.transactional.GetGameById(prep.ctx, tx, gameId)
			assert.EqualError(t, err, unexpectedError.Error(), "unexpected error")
		})
		t.Run("DueNoRow", func(t *testing.T) {
			prep := testPreparation(t)

			tx := prep.tx(func() {
				prep.sqlMock.ExpectQuery("SELECT (.+) FROM games").
					WithArgs(gameId).
					WillReturnRows(
						sqlmock.NewRows(gameColumn),
					)
			})

			_, err := prep.transactional.GetGameById(prep.ctx, tx, gameId)
			assert.EqualError(t, err, sql.ErrNoRows.Error(), "unexpected error")
		})
	})
	t.Run("Success", func(t *testing.T) {
		prep := testPreparation(t)

		tx := prep.tx(func() {
			prep.sqlMock.ExpectQuery("SELECT (.+) FROM games").
				WithArgs(gameId).
				WillReturnRows(
					sqlmock.NewRows(gameColumn).
						AddRow(currentPlayerId, boardBase),
				)
		})

		game, err := prep.transactional.GetGameById(prep.ctx, tx, gameId)
		if assert.NoError(t, err, "no error") {
			assert.Equal(t, gameId, game.Id, "equal")
			assert.Equal(t, currentPlayerId, game.CurrentPlayerId, "equal")
			assert.Empty(t, game.Players, "no player query")
			assert.Empty(t, game.MaxStrength, "not selected yet")
			assert.Equal(t, boardBase, game.BoardBase, "board base")
			assert.Empty(t, game.BoardPositioning, "not selected yet")
		}
	})
}

func TestTransactional_LogPlayedWord(t *testing.T) {
	t.Run("ErrorExecContext", func(t *testing.T) {
		prep := testPreparation(t)

		unexpectedError := errors.New("unexpected error")
		tx := prep.tx(func() {
			prep.sqlMock.ExpectExec("INSERT INTO played_word").
				WithArgs(gameId, wordString, playerId).
				WillReturnError(unexpectedError)
		})

		err := prep.transactional.LogPlayedWord(prep.ctx, tx, gameId, playerId, wordString)
		assert.EqualError(t, err, unexpectedError.Error())
	})
	t.Run("Success", func(t *testing.T) {
		prep := testPreparation(t)

		tx := prep.tx(func() {
			prep.sqlMock.ExpectExec("INSERT INTO played_word").
				WithArgs(gameId, wordString, playerId).
				WillReturnResult(sqlmock.NewResult(time.Now().UnixNano(), 1))
		})

		err := prep.transactional.LogPlayedWord(prep.ctx, tx, gameId, playerId, wordString)
		assert.NoError(t, err)
	})
}