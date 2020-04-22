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
		{ID: uint64(time.Now().UnixNano()), Username: usernames[0]},
		{ID: uint64(time.Now().UnixNano()), Username: usernames[1]},
	}
	gameID           = uint64(time.Now().UnixNano())
	playerID         = players[0].ID
	gamePlayerID     = uint64(time.Now().UnixNano())
	currentOrder     = uint8(1)
	boardBase        = []uint8{22, 14, 17, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24}
	boardPositioning = []uint8{2, 2, 2, 1, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	maxStrength      = uint8(2)
	wordString       = "word"
)

var (
	gameColumn       = []string{"current_player_ID", "board_base", "board_positioning", "max_strength"}
	gamePlayerColumn = []string{"game_ID", "player_ID", "ordering"}
	playerColumn     = []string{"ID", "username"}
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
	boardPositioning := make([]uint8, 25)
	game := data.Game{
		CurrentOrder:     currentOrder,
		BoardBase:        boardBase,
		BoardPositioning: boardPositioning,
		MaxStrength:      maxStrength,
		State:            data.ONGOING,
	}

	t.Run("ErrorExecContext", func(t *testing.T) {
		prep := testPreparation(t)

		unexpectedError := errors.New("unexpected error")
		tx := prep.tx(func() {
			prep.sqlMock.ExpectExec("INSERT INTO games").
				WithArgs(currentOrder, boardBase, boardPositioning, maxStrength, data.ONGOING).
				WillReturnError(unexpectedError)
		})

		_, err := prep.transactional.InsertGame(prep.ctx, tx, game)
		assert.EqualError(t, err, unexpectedError.Error(), "unexpected error")
	})
	t.Run("Success", func(t *testing.T) {
		prep := testPreparation(t)

		tx := prep.tx(func() {
			prep.sqlMock.ExpectExec("INSERT INTO games").
				WithArgs(currentOrder, boardBase, boardPositioning, maxStrength, data.ONGOING).
				WillReturnResult(sqlmock.NewResult(int64(gameID), 1))
		})

		game, err := prep.transactional.InsertGame(prep.ctx, tx, game)
		if assert.NoError(t, err) {
			assert.Equal(t, gameID, game.ID)
			assert.Equal(t, currentOrder, game.CurrentOrder)
			assert.Equal(t, boardBase, game.BoardBase)
			assert.Equal(t, make([]uint8, 25), game.BoardPositioning)
			assert.Equal(t, data.ONGOING, game.State)
			assert.Equal(t, maxStrength, game.MaxStrength)
			assert.Empty(t, game.Players)
		}
	})
}

func TestTransactional_InsertGamePlayerBulk(t *testing.T) {
	game := data.Game{
		ID: gameID,
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
						gameID, players[0].ID, 1,
						gameID, players[1].ID, 2,
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
					gameID, players[0].ID, 1,
					gameID, players[1].ID, 2,
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
	t.Run("Success", func(t *testing.T) {
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
					AddRow(players[0].ID, players[0].Username).
					AddRow(players[1].ID, players[1].Username),
			)

		actualPlayers, err := prep.transactional.GetPlayersByUsernames(prep.ctx, usernames)
		if assert.NoError(t, err) {
			assert.Equal(t, players, actualPlayers)
		}

	})
}

func TestTransactional_GetGamePlayerByID(t *testing.T) {
	t.Run("ErrorScanning", func(t *testing.T) {
		t.Run("DueErrorQuerying", func(t *testing.T) {
			prep := testPreparation(t)

			unexpectedError := errors.New("unexpected error")
			prep.sqlMock.ExpectQuery("SELECT (.+) FROM game_player").
				WithArgs(gamePlayerID).
				WillReturnError(unexpectedError)

			_, err := prep.transactional.GetGamePlayerByID(prep.ctx, gamePlayerID)
			assert.EqualError(t, err, unexpectedError.Error(), "unexpected error")
		})
		t.Run("DueNoRow", func(t *testing.T) {
			prep := testPreparation(t)

			prep.sqlMock.ExpectQuery("SELECT (.+) FROM game_player").
				WithArgs(gamePlayerID).
				WillReturnRows(
					sqlmock.NewRows(gamePlayerColumn),
				)

			_, err := prep.transactional.GetGamePlayerByID(prep.ctx, gamePlayerID)
			assert.EqualError(t, err, sql.ErrNoRows.Error(), "no row")
		})
	})
	t.Run("Success", func(t *testing.T) {
		prep := testPreparation(t)

		prep.sqlMock.ExpectQuery("SELECT (.+) FROM game_player").
			WithArgs(gamePlayerID).
			WillReturnRows(
				sqlmock.NewRows(gamePlayerColumn).
					AddRow(gameID, playerID, uint8(1)),
			)

		gamePlayer, err := prep.transactional.GetGamePlayerByID(prep.ctx, gamePlayerID)
		if assert.NoError(t, err, "no error") {
			assert.Equal(t, gameID, gamePlayer.GameID)
			assert.Equal(t, playerID, gamePlayer.PlayerID)
			assert.Equal(t, uint8(1), gamePlayer.Ordering)
		}
	})
}

func TestTransactional_GetGamePlayersByGameID(t *testing.T) {
	t.Run("ErrorQuerying", func(t *testing.T) {
		prep := testPreparation(t)

		unexpectedError := errors.New("unexpected error")
		tx := prep.tx(func() {
			prep.sqlMock.ExpectQuery("SELECT (.+) FROM game_player").
				WithArgs(gameID).
				WillReturnError(unexpectedError)
		})

		_, err := prep.transactional.GetGamePlayersByGameID(prep.ctx, tx, gameID)
		assert.EqualError(t, err, unexpectedError.Error(), "unexpected error")
	})
	t.Run("ErrorScanning", func(t *testing.T) {
		prep := testPreparation(t)

		tx := prep.tx(func() {
			prep.sqlMock.ExpectQuery("SELECT (.+) FROM game_player").
				WithArgs(gamePlayerID).
				WillReturnRows(
					sqlmock.NewRows([]string{"player_ID", "ordering"}).
						AddRow(playerID, "halo"),
				)
		})

		_, err := prep.transactional.GetGamePlayersByGameID(prep.ctx, tx, gameID)
		assert.Error(t, err)
	})
	t.Run("NoGamePlayersFound", func(t *testing.T) {
		prep := testPreparation(t)

		tx := prep.tx(func() {
			prep.sqlMock.ExpectQuery("SELECT (.+) FROM game_player").
				WithArgs(gameID).
				WillReturnRows(
					sqlmock.NewRows([]string{"player_ID", "ordering"}),
				)
		})

		gamePlayers, err := prep.transactional.GetGamePlayersByGameID(prep.ctx, tx, gameID)
		if assert.NoError(t, err, "no error") {
			assert.Empty(t, gamePlayers)
		}
	})
	t.Run("Success", func(t *testing.T) {
		prep := testPreparation(t)

		tx := prep.tx(func() {
			prep.sqlMock.ExpectQuery("SELECT (.+) FROM game_player").
				WithArgs(gameID).
				WillReturnRows(
					sqlmock.NewRows([]string{"player_ID", "ordering"}).
						AddRow(playerID, uint8(1)).
						AddRow(playerID+1, uint8(2)),
				)
		})

		gamePlayers, err := prep.transactional.GetGamePlayersByGameID(prep.ctx, tx, gameID)
		if assert.NoError(t, err, "no error") {
			expectedGamePlayers := []data.GamePlayer{
				{GameID: gameID, PlayerID: playerID, Ordering: 1},
				{GameID: gameID, PlayerID: playerID + 1, Ordering: 2},
			}
			assert.Equal(t, expectedGamePlayers, gamePlayers)
		}
	})
}

func TestTransactional_GetGameByID(t *testing.T) {
	t.Run("ErrorScanning", func(t *testing.T) {
		t.Run("DueErrorQuerying", func(t *testing.T) {
			prep := testPreparation(t)

			unexpectedError := errors.New("unexpected error")
			tx := prep.tx(func() {
				prep.sqlMock.ExpectQuery("SELECT (.+) FROM games").
					WithArgs(gameID).
					WillReturnError(unexpectedError)
			})

			_, err := prep.transactional.GetGameByID(prep.ctx, tx, gameID)
			assert.EqualError(t, err, unexpectedError.Error(), "unexpected error")
		})
		t.Run("DueNoRow", func(t *testing.T) {
			prep := testPreparation(t)

			tx := prep.tx(func() {
				prep.sqlMock.ExpectQuery("SELECT (.+) FROM games").
					WithArgs(gameID).
					WillReturnRows(
						sqlmock.NewRows(gameColumn),
					)
			})

			_, err := prep.transactional.GetGameByID(prep.ctx, tx, gameID)
			assert.EqualError(t, err, sql.ErrNoRows.Error(), "unexpected error")
		})
	})
	t.Run("Success", func(t *testing.T) {
		prep := testPreparation(t)

		tx := prep.tx(func() {
			prep.sqlMock.ExpectQuery("SELECT (.+) FROM games").
				WithArgs(gameID).
				WillReturnRows(
					sqlmock.NewRows(gameColumn).
						AddRow(currentOrder, boardBase, boardPositioning, maxStrength),
				)
		})

		game, err := prep.transactional.GetGameByID(prep.ctx, tx, gameID)
		if assert.NoError(t, err, "no error") {
			assert.Equal(t, gameID, game.ID, "equal")
			assert.Equal(t, currentOrder, game.CurrentOrder, "equal")
			assert.Empty(t, game.Players, "no player query")
			assert.Equal(t, maxStrength, game.MaxStrength)
			assert.Equal(t, boardBase, game.BoardBase, "board base")
			assert.Equal(t, boardPositioning, game.BoardPositioning)
		}
	})
}

func TestTransactional_LogPlayedWord(t *testing.T) {
	t.Run("ErrorExecContext", func(t *testing.T) {
		prep := testPreparation(t)

		unexpectedError := errors.New("unexpected error")
		tx := prep.tx(func() {
			prep.sqlMock.ExpectExec("INSERT INTO played_word").
				WithArgs(gameID, wordString, playerID).
				WillReturnError(unexpectedError)
		})

		err := prep.transactional.LogPlayedWord(prep.ctx, tx, gameID, playerID, wordString)
		assert.EqualError(t, err, unexpectedError.Error())
	})
	t.Run("Success", func(t *testing.T) {
		prep := testPreparation(t)

		tx := prep.tx(func() {
			prep.sqlMock.ExpectExec("INSERT INTO played_word").
				WithArgs(gameID, wordString, playerID).
				WillReturnResult(sqlmock.NewResult(time.Now().UnixNano(), 1))
		})

		err := prep.transactional.LogPlayedWord(prep.ctx, tx, gameID, playerID, wordString)
		assert.NoError(t, err)
	})
}

func TestTransactional_UpdateGame(t *testing.T) {
	t.Run("ErrorExecContext", func(t *testing.T) {
		prep := testPreparation(t)

		unexpectedError := errors.New("unexpected error")
		tx := prep.tx(func() {
			prep.sqlMock.ExpectExec("UPDATE game SET").
				WithArgs(boardPositioning, currentOrder, data.END, gameID).
				WillReturnError(unexpectedError)
		})

		err := prep.transactional.UpdateGame(
			prep.ctx, tx, data.Game{ID: gameID, BoardPositioning: boardPositioning, CurrentOrder: currentOrder, State: data.END},
		)
		assert.EqualError(t, err, unexpectedError.Error())
	})
	t.Run("Success", func(t *testing.T) {
		prep := testPreparation(t)

		tx := prep.tx(func() {
			prep.sqlMock.ExpectExec("UPDATE game SET").
				WithArgs(boardPositioning, currentOrder, data.END, gameID).
				WillReturnResult(sqlmock.NewResult(time.Now().UnixNano(), 1))
		})

		err := prep.transactional.UpdateGame(
			prep.ctx, tx, data.Game{ID: gameID, BoardPositioning: boardPositioning, CurrentOrder: currentOrder, State: data.END},
		)
		assert.NoError(t, err)
	})
}
