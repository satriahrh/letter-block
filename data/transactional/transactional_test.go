package transactional_test

import (
	"github.com/satriahrh/letter-block/data"
	"github.com/satriahrh/letter-block/data/transactional"

	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
)

type Preparation struct {
	sqlMock       sqlmock.Sqlmock
	ctx           context.Context
	transactional *transactional.Transactional
	tx            func(func()) *sql.Tx
}

var (
	players = []data.Player{
		{Id: data.PlayerId(time.Now().UnixNano()), Username: "sarjono"},
		{Id: data.PlayerId(time.Now().UnixNano()), Username: "mukti"},
	}
	gameId           = data.GameId(time.Now().UnixNano())
	playerId         = players[0].Id
	gamePlayerId     = data.GamePlayerId(time.Now().UnixNano())
	currentOrder     = uint8(1)
	boardBase        = []uint8{22, 14, 17, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24}
	boardPositioning = []uint8{2, 2, 2, 1, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	wordString       = "word"
	timestamp        = time.Now()
	fingerprint      = `69df370f86b026724a73c68599a60a5ce1d19a5c6df8b33e0fc24e8f6310c668372aeee8ed4929ae8b4f646da799230dbc205af61f36794a9a89b1cc093fb648`
)

var (
	gameColumn       = []string{"current_player_order", "number_of_player", "board_base", "board_positioning", "state"}
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
		CurrentPlayerOrder: currentOrder,
		NumberOfPlayer:     2,
		BoardBase:          boardBase,
		BoardPositioning:   boardPositioning,
		State:              data.ONGOING,
	}

	t.Run("ErrorExecContext", func(t *testing.T) {
		prep := testPreparation(t)

		unexpectedError := errors.New("unexpected error")
		tx := prep.tx(func() {
			prep.sqlMock.ExpectExec("INSERT INTO games").
				WithArgs(game.CurrentPlayerOrder, game.NumberOfPlayer, game.BoardBase, game.BoardPositioning, game.State).
				WillReturnError(unexpectedError)
		})

		_, err := prep.transactional.InsertGame(prep.ctx, tx, game)
		assert.EqualError(t, err, unexpectedError.Error(), "unexpected error")
	})
	t.Run("Success", func(t *testing.T) {
		prep := testPreparation(t)

		tx := prep.tx(func() {
			prep.sqlMock.ExpectExec("INSERT INTO games").
				WithArgs(game.CurrentPlayerOrder, game.NumberOfPlayer, game.BoardBase, game.BoardPositioning, game.State).
				WillReturnResult(sqlmock.NewResult(int64(gameId), 1))
		})

		actualGame, err := prep.transactional.InsertGame(prep.ctx, tx, game)
		if assert.NoError(t, err) {
			expectedGame := game
			expectedGame.Id = gameId
			assert.Equal(t, expectedGame, actualGame)
		}
	})
}

func TestTransactional_InsertGamePlayer(t *testing.T) {
	t.Run("ErrorExecContext", func(t *testing.T) {
		prep := testPreparation(t)

		unexpectedError := errors.New("unexpected error")
		tx := prep.tx(func() {
			prep.sqlMock.ExpectExec("INSERT INTO games_players").
				WithArgs(gameId, playerId).
				WillReturnError(unexpectedError)
		})

		_, err := prep.transactional.InsertGamePlayer(prep.ctx, tx,
			data.Game{Id: gameId}, data.Player{Id: playerId})
		assert.EqualError(t, err, unexpectedError.Error(), "unexpected error")
	})
	t.Run("Success", func(t *testing.T) {
		prep := testPreparation(t)

		tx := prep.tx(func() {
			prep.sqlMock.ExpectExec("INSERT INTO games_players").
				WithArgs(gameId, players[0].Id).
				WillReturnResult(sqlmock.NewResult(1, 1))
		})

		game, err := prep.transactional.InsertGamePlayer(prep.ctx, tx,
			data.Game{Id: gameId}, players[0])
		if assert.NoError(t, err) {
			assert.Equal(t, data.Game{
				Id:      gameId,
				Players: players[:1],
			}, game)
		}
	})
}

func TestTransactional_GetPlayerById(t *testing.T) {
	t.Run("ErrorScanning", func(t *testing.T) {
		t.Run("DueErrorQuerying", func(t *testing.T) {
			prep := testPreparation(t)

			unexpectedError := errors.New("unexpected error")
			prep.sqlMock.ExpectQuery("SELECT (.+) FROM players").
				WithArgs(playerId).
				WillReturnError(unexpectedError)

			_, err := prep.transactional.GetPlayerById(prep.ctx, playerId)
			assert.EqualError(t, err, unexpectedError.Error(), "unexpected error")
		})
		t.Run("DueNoRow", func(t *testing.T) {
			prep := testPreparation(t)

			prep.sqlMock.ExpectQuery("SELECT (.+) FROM players").
				WithArgs(playerId).
				WillReturnRows(
					sqlmock.NewRows(playerColumn),
				)

			_, err := prep.transactional.GetPlayerById(prep.ctx, playerId)
			assert.EqualError(t, err, sql.ErrNoRows.Error(), "no row")
		})
	})
	t.Run("Success", func(t *testing.T) {
		prep := testPreparation(t)

		prep.sqlMock.ExpectQuery("SELECT (.+) FROM players").
			WithArgs(playerId).
			WillReturnRows(
				sqlmock.NewRows(playerColumn).
					AddRow(players[0].Id, players[0].Username),
			)

		player, err := prep.transactional.GetPlayerById(prep.ctx, playerId)
		if assert.NoError(t, err, "no error") {
			assert.Equal(t, players[0], player)
		}
	})
}

func TestTransactional_GetGamePlayersByGameId(t *testing.T) {
	t.Run("ErrorQuerying", func(t *testing.T) {
		prep := testPreparation(t)

		unexpectedError := errors.New("unexpected error")
		tx := prep.tx(func() {
			prep.sqlMock.ExpectQuery("SELECT (.+) FROM games_players").
				WithArgs(gameId).
				WillReturnError(unexpectedError)
		})

		_, err := prep.transactional.GetGamePlayersByGameId(prep.ctx, tx, gameId)
		assert.EqualError(t, err, unexpectedError.Error(), "unexpected error")
	})
	t.Run("ErrorScanning", func(t *testing.T) {
		prep := testPreparation(t)

		tx := prep.tx(func() {
			prep.sqlMock.ExpectQuery("SELECT (.+) FROM games_players").
				WithArgs(gameId).
				WillReturnRows(
					sqlmock.NewRows([]string{"player_id"}).
						AddRow("a"),
				)
		})

		_, err := prep.transactional.GetGamePlayersByGameId(prep.ctx, tx, gameId)
		assert.Error(t, err)
	})
	t.Run("NoGamePlayersFound", func(t *testing.T) {
		prep := testPreparation(t)

		tx := prep.tx(func() {
			prep.sqlMock.ExpectQuery("SELECT (.+) FROM games_players").
				WithArgs(gameId).
				WillReturnRows(
					sqlmock.NewRows([]string{"player_id"}),
				)
		})

		gamePlayers, err := prep.transactional.GetGamePlayersByGameId(prep.ctx, tx, gameId)
		if assert.NoError(t, err, "no error") {
			assert.Empty(t, gamePlayers)
		}
	})
	t.Run("Success", func(t *testing.T) {
		prep := testPreparation(t)

		tx := prep.tx(func() {
			prep.sqlMock.ExpectQuery("SELECT (.+) FROM games_players").
				WithArgs(gameId).
				WillReturnRows(
					sqlmock.NewRows([]string{"player_id"}).
						AddRow(playerId).
						AddRow(playerId + 1),
				)
		})

		gamePlayers, err := prep.transactional.GetGamePlayersByGameId(prep.ctx, tx, gameId)
		if assert.NoError(t, err, "no error") {
			expectedGamePlayers := []data.GamePlayer{
				{GameId: gameId, PlayerId: playerId},
				{GameId: gameId, PlayerId: playerId + 1},
			}
			assert.Equal(t, expectedGamePlayers, gamePlayers)
		}
	})
}

func TestTransactional_GetPlayersByGameId(t *testing.T) {
	query := `SELECT (.+) FROM players INNER JOIN \( SELECT (.+) FROM games_players WHERE game_id = \? \) as game_players ON game_players.player_id = players.id`
	t.Run("ErrorQueryContext", func(t *testing.T) {
		prep := testPreparation(t)

		unexpectedError := errors.New("unexpected error")
		prep.sqlMock.ExpectQuery(query).
			WithArgs(gameId).
			WillReturnError(unexpectedError)

		_, err := prep.transactional.GetPlayersByGameId(prep.ctx, gameId)
		assert.EqualError(t, err, unexpectedError.Error())
	})
	t.Run("ErrorScanning", func(t *testing.T) {
		prep := testPreparation(t)

		prep.sqlMock.ExpectQuery(query).
			WithArgs(gameId).
			WillReturnRows(
				sqlmock.NewRows(playerColumn).
					AddRow("v", "v"),
			)

		_, err := prep.transactional.GetPlayersByGameId(prep.ctx, gameId)
		assert.Error(t, err)
	})
	t.Run("Success", func(t *testing.T) {
		prep := testPreparation(t)

		prep.sqlMock.ExpectQuery(query).
			WithArgs(gameId).
			WillReturnRows(
				sqlmock.NewRows(playerColumn).
					AddRow(players[0].Id, players[0].Username).
					AddRow(players[1].Id, players[1].Username),
			)

		actual, err := prep.transactional.GetPlayersByGameId(prep.ctx, gameId)
		if assert.NoError(t, err) {
			assert.Equal(t, players, actual)
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
		expectedGame := data.Game{
			Id:                 gameId,
			CurrentPlayerOrder: currentOrder,
			NumberOfPlayer:     2,
			BoardBase:          boardBase,
			BoardPositioning:   boardPositioning,
		}
		testSuite := func(prep Preparation, tx *sql.Tx, gameId data.GameId) {
			game, err := prep.transactional.GetGameById(prep.ctx, tx, gameId)
			if assert.NoError(t, err, "no error") {
				assert.Equal(t, expectedGame, game)
			}
		}
		t.Run("WithTransaction", func(t *testing.T) {
			prep := testPreparation(t)

			tx := prep.tx(func() {
				prep.sqlMock.ExpectQuery("SELECT (.+) FROM games").
					WithArgs(gameId).
					WillReturnRows(
						sqlmock.NewRows(gameColumn).
							AddRow(
								expectedGame.CurrentPlayerOrder, expectedGame.NumberOfPlayer, expectedGame.BoardBase,
								expectedGame.BoardPositioning, expectedGame.State,
							),
					)
			})

			testSuite(prep, tx, gameId)
		})
		t.Run("WithoutTransaction", func(t *testing.T) {
			prep := testPreparation(t)

			prep.sqlMock.ExpectQuery("SELECT (.+) FROM games").
				WithArgs(gameId).
				WillReturnRows(
					sqlmock.NewRows(gameColumn).
						AddRow(
							expectedGame.CurrentPlayerOrder, expectedGame.NumberOfPlayer, expectedGame.BoardBase,
							expectedGame.BoardPositioning, expectedGame.State,
						),
				)

			testSuite(prep, nil, gameId)
		})
	})
}

func TestTransactional_GetGamesByPlayerId(t *testing.T) {
	query := `SELECT (.+) FROM games INNER JOIN \( SELECT (.+) FROM games_players WHERE player_id = \? \) as played_games ON played_games.game_id = games.id`
	t.Run("ErrorQueryContext", func(t *testing.T) {
		prep := testPreparation(t)

		unexpectedError := errors.New("unexpected error")
		prep.sqlMock.ExpectQuery(query).
			WithArgs(playerId).
			WillReturnError(unexpectedError)

		_, err := prep.transactional.GetGamesByPlayerId(prep.ctx, playerId)
		assert.EqualError(t, err, unexpectedError.Error())
	})
	gameColumn := []string{"id", "current_player_order", "number_of_player", "board_base", "board_positioning", "state"}
	t.Run("ErrorScanning", func(t *testing.T) {
		prep := testPreparation(t)

		prep.sqlMock.ExpectQuery(query).
			WithArgs(playerId).
			WillReturnRows(
				sqlmock.NewRows(gameColumn).
					AddRow(1, 2, 3, 4, 5, "v"),
			)

		_, err := prep.transactional.GetGamesByPlayerId(prep.ctx, playerId)
		assert.Error(t, err)
	})
	t.Run("Success", func(t *testing.T) {
		prep := testPreparation(t)

		expectedGame := data.Game{
			Id:                 gameId,
			CurrentPlayerOrder: 1,
			NumberOfPlayer:     2,
			State:              data.ONGOING,
			BoardBase:          boardBase,
			BoardPositioning:   boardPositioning,
		}
		prep.sqlMock.ExpectQuery(query).
			WithArgs(playerId).
			WillReturnRows(
				sqlmock.NewRows(gameColumn).
					AddRow(
						expectedGame.Id, expectedGame.CurrentPlayerOrder, expectedGame.NumberOfPlayer, expectedGame.BoardBase,
						expectedGame.BoardPositioning, expectedGame.State,
					),
			)

		games, err := prep.transactional.GetGamesByPlayerId(prep.ctx, playerId)
		if assert.NoError(t, err) {
			assert.Equal(t, []data.Game{expectedGame}, games)
		}
	})
}

func TestTransactional_LogPlayedWord(t *testing.T) {
	t.Run("ErrorExecContext", func(t *testing.T) {
		prep := testPreparation(t)

		unexpectedError := errors.New("unexpected error")
		tx := prep.tx(func() {
			prep.sqlMock.ExpectExec("INSERT INTO played_words").
				WithArgs(gameId, wordString, playerId).
				WillReturnError(unexpectedError)
		})

		err := prep.transactional.LogPlayedWord(prep.ctx, tx, gameId, playerId, wordString)
		assert.EqualError(t, err, unexpectedError.Error())
	})
	t.Run("Success", func(t *testing.T) {
		prep := testPreparation(t)

		tx := prep.tx(func() {
			prep.sqlMock.ExpectExec("INSERT INTO played_words").
				WithArgs(gameId, wordString, playerId).
				WillReturnResult(sqlmock.NewResult(time.Now().UnixNano(), 1))
		})

		err := prep.transactional.LogPlayedWord(prep.ctx, tx, gameId, playerId, wordString)
		assert.NoError(t, err)
	})
}

func TestTransactional_GetPlayedWordsByGameId(t *testing.T) {
	query := `SELECT (.+) FROM played_words WHERE game_id = \?`
	t.Run("ErrorQueryContext", func(t *testing.T) {
		prep := testPreparation(t)

		unexpectedError := errors.New("unexpected error")
		prep.sqlMock.ExpectQuery(query).
			WithArgs(gameId).
			WillReturnError(unexpectedError)

		_, err := prep.transactional.GetPlayedWordsByGameId(prep.ctx, gameId)
		assert.EqualError(t, err, unexpectedError.Error())
	})
	playedWordColumn := []string{"word", "player_id"}
	t.Run("ErrorScanning", func(t *testing.T) {
		prep := testPreparation(t)

		prep.sqlMock.ExpectQuery(query).
			WithArgs(gameId).
			WillReturnRows(
				sqlmock.NewRows(playedWordColumn).
					AddRow("KATA", "a"),
			)

		_, err := prep.transactional.GetPlayedWordsByGameId(prep.ctx, gameId)
		assert.Error(t, err)
	})
	t.Run("Success", func(t *testing.T) {
		prep := testPreparation(t)

		playedWords := []data.PlayedWord{
			{players[0].Id, "KATA"},
			{players[1].Id, "KITA"},
		}
		prep.sqlMock.ExpectQuery(query).
			WithArgs(gameId).
			WillReturnRows(
				sqlmock.NewRows(playedWordColumn).
					AddRow(playedWords[0].Word, playedWords[0].PlayerId).
					AddRow(playedWords[1].Word, playedWords[1].PlayerId),
			)

		actual, err := prep.transactional.GetPlayedWordsByGameId(prep.ctx, gameId)
		if assert.NoError(t, err) {
			assert.Equal(t, playedWords, actual)
		}
	})
}

func TestTransactional_UpdateGame(t *testing.T) {
	t.Run("ErrorExecContext", func(t *testing.T) {
		prep := testPreparation(t)

		unexpectedError := errors.New("unexpected error")
		tx := prep.tx(func() {
			prep.sqlMock.ExpectExec("UPDATE games SET").
				WithArgs(boardPositioning, boardBase, currentOrder, data.END, gameId).
				WillReturnError(unexpectedError)
		})

		err := prep.transactional.UpdateGame(
			prep.ctx, tx, data.Game{
				Id: gameId, BoardPositioning: boardPositioning, BoardBase: boardBase, CurrentPlayerOrder: currentOrder,
				State: data.END,
			},
		)
		assert.EqualError(t, err, unexpectedError.Error())
	})
	t.Run("Success", func(t *testing.T) {
		prep := testPreparation(t)

		tx := prep.tx(func() {
			prep.sqlMock.ExpectExec("UPDATE games SET").
				WithArgs(boardPositioning, boardBase, currentOrder, data.END, gameId).
				WillReturnResult(sqlmock.NewResult(time.Now().UnixNano(), 1))
		})

		err := prep.transactional.UpdateGame(
			prep.ctx, tx, data.Game{
				Id: gameId, BoardPositioning: boardPositioning, BoardBase: boardBase, CurrentPlayerOrder: currentOrder,
				State: data.END,
			},
		)
		assert.NoError(t, err)
	})
}

func TestTransactional_UpsertPlayer(t *testing.T) {
	t.Run("ErrorExecContext", func(t *testing.T) {
		prep := testPreparation(t)

		unexpectedError := errors.New("unexpected error")
		tx := prep.tx(func() {
			prep.sqlMock.ExpectExec("INSERT IGNORE INTO players").
				WithArgs(fingerprint, players[0].Username, players[0].Username).
				WillReturnError(unexpectedError)
		})

		err := prep.transactional.UpsertPlayer(
			prep.ctx, tx, data.Player{
				Username:          players[0].Username,
				DeviceFingerprint: data.DeviceFingerprint(fingerprint),
			},
		)
		assert.EqualError(t, err, unexpectedError.Error())
	})
	t.Run("Success", func(t *testing.T) {
		prep := testPreparation(t)

		tx := prep.tx(func() {
			prep.sqlMock.ExpectExec("INSERT IGNORE INTO players").
				WithArgs(fingerprint, players[0].Username, players[0].Username).
				WillReturnResult(sqlmock.NewResult(time.Now().UnixNano(), 1))
		})

		err := prep.transactional.UpsertPlayer(
			prep.ctx, tx, data.Player{
				Username:          players[0].Username,
				DeviceFingerprint: data.DeviceFingerprint(fingerprint),
			},
		)
		assert.NoError(t, err)
	})
}

func TestTransactional_GetPlayerByDeviceFingerprint(t *testing.T) {
	t.Run("ErrorQueryContext", func(t *testing.T) {
		prep := testPreparation(t)

		tx := prep.tx(func() {
			prep.sqlMock.ExpectQuery(`SELECT (.+) FROM players WHERE device_fingerprint = \?`).
				WithArgs(fingerprint).
				WillReturnError(sql.ErrConnDone)
		})

		_, err := prep.transactional.GetPlayerByDeviceFingerprint(prep.ctx, tx, data.DeviceFingerprint(fingerprint))
		assert.EqualError(t, err, sql.ErrConnDone.Error())
	})
	t.Run("Success", func(t *testing.T) {
		prep := testPreparation(t)

		tx := prep.tx(func() {
			prep.sqlMock.ExpectQuery(`SELECT (.+) FROM players WHERE device_fingerprint = \?`).
				WithArgs(fingerprint).
				WillReturnRows(
					sqlmock.NewRows([]string{"id", "username", "device_fingerprint", "session_expired_in"}).
						AddRow(playerId, players[0].Username, fingerprint, timestamp.Unix()),
				)
		})

		player, err := prep.transactional.GetPlayerByDeviceFingerprint(prep.ctx, tx, data.DeviceFingerprint(fingerprint))
		if assert.NoError(t, err) {
			assert.Equal(t, data.Player{Id: playerId, Username: players[0].Username, DeviceFingerprint: data.DeviceFingerprint(fingerprint), SessionExpiredAt: timestamp.Unix()}, player)
		}
	})
}
