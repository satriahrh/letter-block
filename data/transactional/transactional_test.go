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

func sqlMockCreation(t *testing.T) (*sql.DB, sqlmock.Sqlmock) {
	db, mock, err := sqlmock.New()
	if !assert.NoError(t, err, "sqlmock") {
		t.FailNow()
	}

	return db, mock
}

func TestTransactional_BeginTransaction(t *testing.T) {
	ctx := context.TODO()

	t.Run("ErrorBeginTrx", func(t *testing.T) {
		db, sqlMock := sqlMockCreation(t)
		trans := transactional.NewTransactional(db)

		unexpectedError := errors.New("unexpected error")
		sqlMock.ExpectBegin().
			WillReturnError(unexpectedError)

		_, err := trans.BeginTransaction(ctx)
		assert.Error(t, err, unexpectedError.Error(), "unexpected error")
	})
	t.Run("Success", func(t *testing.T) {
		db, sqlMock := sqlMockCreation(t)
		trans := transactional.NewTransactional(db)

		sqlMock.ExpectBegin()

		tx, err := trans.BeginTransaction(ctx)
		if assert.NoError(t, err, "no error") {
			assert.NotEmpty(t, tx, "return non empty transaction")
		}

	})
}
