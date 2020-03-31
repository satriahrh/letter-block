package transactional_test

import (
	"context"
	"database/sql"
	"errors"
	"github.com/satriahrh/letter-block/data/transactional"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
)

type Preparation struct {
	db      *sql.DB
	sqlMock sqlmock.Sqlmock
	ctx     context.Context
}

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
		assert.Error(t, err, unexpectedError.Error(), "unexpected error")
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
			assert.Error(t, err, unexpectedRollbackError.Error(), "unexpected rollback error")
		})
		t.Run("SuccessRollback", func(t *testing.T) {
			preparation := testPreparation(t)
			trans := transactional.NewTransactional(preparation.db)

			preparation.sqlMock.ExpectBegin()
			preparation.sqlMock.ExpectRollback()

			err := trans.FinalizeTransaction(beginTx(preparation.db), unexpectedError)
			assert.Error(t, err, unexpectedError.Error(), "unexpected error")
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
			assert.Error(t, err, unexpectedError.Error(),  "commit return an error")
		})
	})
}
