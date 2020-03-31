package transactional_test

import (
	"context"
	"database/sql"
	"errors"
	"github.com/satriahrh/letter-block/data/transactional"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
)

type Preparation struct {
	db      *sql.DB
	sqlMock sqlmock.Sqlmock
	ctx     context.Context
}

var (
	gameId          = uint64(time.Now().UnixNano())
	currentPlayerId = uint64(time.Now().UnixNano())
	boardBase       = []uint8{22, 14, 17, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24}
)

func testPreparation(t *testing.T) Preparation {
	ctx := context.TODO()
	db, sqlMock, err := sqlmock.New()
	if !assert.NoError(t, err, "sqlmock") {
		t.FailNow()
	}

	return Preparation{db, sqlMock, ctx}
}

func TestTransactional_BeginTransaction(t *testing.T) {
	t.Run("ErrorBeginTrx", func(t *testing.T) {
		preparation := testPreparation(t)
		trans := transactional.NewTransactional(preparation.db)

		unexpectedError := errors.New("unexpected error")
		preparation.sqlMock.ExpectBegin().
			WillReturnError(unexpectedError)

		_, err := trans.BeginTransaction(preparation.ctx)
		assert.EqualError(t, err, unexpectedError.Error(), "unexpected error")
	})
	t.Run("Success", func(t *testing.T) {
		preparation := testPreparation(t)
		trans := transactional.NewTransactional(preparation.db)

		preparation.sqlMock.ExpectBegin()

		tx, err := trans.BeginTransaction(preparation.ctx)
		if assert.NoError(t, err, "no error") {
			assert.NotEmpty(t, tx, "return non empty transaction")
		}

	})
}

func beginTx(db *sql.DB) *sql.Tx {
	tx, _ := db.Begin()
	return tx
}

func TestTransactional_FinalizeTransaction(t *testing.T) {
	t.Run("ErrIsNotNill", func(t *testing.T) {
		unexpectedError := errors.New("unexpected error")
		unexpectedRollbackError := errors.New("unexpected rollback error")

		t.Run("ErrorRollbackTrx", func(t *testing.T) {
			preparation := testPreparation(t)
			trans := transactional.NewTransactional(preparation.db)

			preparation.sqlMock.ExpectBegin()
			preparation.sqlMock.ExpectRollback().
				WillReturnError(unexpectedRollbackError)

			err := trans.FinalizeTransaction(beginTx(preparation.db), unexpectedError)
			assert.EqualError(t, err, unexpectedRollbackError.Error(), "unexpected rollback error")
		})
		t.Run("SuccessRollback", func(t *testing.T) {
			preparation := testPreparation(t)
			trans := transactional.NewTransactional(preparation.db)

			preparation.sqlMock.ExpectBegin()
			preparation.sqlMock.ExpectRollback()

			err := trans.FinalizeTransaction(beginTx(preparation.db), unexpectedError)
			assert.EqualError(t, err, unexpectedError.Error(), "unexpected error")
		})
	})
	t.Run("Commit", func(t *testing.T) {
		t.Run("ReturnNilError", func(t *testing.T) {
			preparation := testPreparation(t)
			trans := transactional.NewTransactional(preparation.db)

			preparation.sqlMock.ExpectBegin()
			preparation.sqlMock.ExpectCommit()

			err := trans.FinalizeTransaction(beginTx(preparation.db), nil)
			assert.NoError(t, err, "no error")
		})
		t.Run("ReturnError", func(t *testing.T) {
			unexpectedError := errors.New("unexpected error")
			preparation := testPreparation(t)
			trans := transactional.NewTransactional(preparation.db)

			preparation.sqlMock.ExpectBegin()
			preparation.sqlMock.ExpectCommit().
				WillReturnError(unexpectedError)

			err := trans.FinalizeTransaction(beginTx(preparation.db), nil)
			assert.EqualError(t, err, unexpectedError.Error(), "commit return an error")
		})
	})
}

func TestTransactional_GetGameByID(t *testing.T) {
	t.Run("ErrorScanning", func(t *testing.T) {
		t.Run("DueErrorQuerying", func(t *testing.T) {
			preparation := testPreparation(t)
			trans := transactional.NewTransactional(preparation.db)

			preparation.sqlMock.ExpectBegin()
			unexpectedError := errors.New("unexpected error")
			preparation.sqlMock.ExpectQuery("SELECT (.+) FROM games").
				WithArgs(gameId).
				WillReturnError(unexpectedError)

			_, err := trans.GetGameByID(preparation.ctx, beginTx(preparation.db), gameId)
			assert.EqualError(t, err, unexpectedError.Error(), "unexpected error")
		})
		t.Run("DueNoRow", func(t *testing.T) {
			preparation := testPreparation(t)
			trans := transactional.NewTransactional(preparation.db)

			preparation.sqlMock.ExpectBegin()
			gameColumn := []string{"current_player_id", "board_base"}
			preparation.sqlMock.ExpectQuery("SELECT (.+) FROM games").
				WithArgs(gameId).
				WillReturnRows(
					sqlmock.NewRows(gameColumn),
				)

			_, err := trans.GetGameByID(preparation.ctx, beginTx(preparation.db), gameId)
			assert.EqualError(t, err, sql.ErrNoRows.Error(), "unexpected error")
		})
	})
	t.Run("Success", func(t *testing.T) {
		preparation := testPreparation(t)
		trans := transactional.NewTransactional(preparation.db)

		preparation.sqlMock.ExpectBegin()
		gameColumn := []string{"current_player_id", "board_base"}
		preparation.sqlMock.ExpectQuery("SELECT (.+) FROM games").
			WithArgs(gameId).
			WillReturnRows(
				sqlmock.NewRows(gameColumn).
					AddRow(currentPlayerId, boardBase),
			)

		game, err := trans.GetGameByID(preparation.ctx, beginTx(preparation.db), gameId)
		assert.NoError(t, err, "no error")
		assert.Equal(t, gameId, game.ID, "equal")
		assert.Equal(t, currentPlayerId, game.CurrentPlayerID, "equal")
		assert.Empty(t, game.Players, "no player query")
		assert.Empty(t, game.MaxStrength, "not selected yet")
		assert.Equal(t, boardBase, game.BoardBase, "board base")
		assert.Empty(t, game.BoardPositioning, "not selected yet")
	})
}
