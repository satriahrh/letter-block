package transactional_test

import (
	"github.com/satriahrh/letter-block/data/transactional"

	"context"
	"database/sql"
	"errors"
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
