package letter_block_test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/satriahrh/letter-block"
	"github.com/satriahrh/letter-block/data"
	"github.com/satriahrh/letter-block/data/transactional"
	"github.com/satriahrh/letter-block/dictionary"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"testing"
)

type Dictionary struct {
	mock.Mock
}

func (d *Dictionary) LemmaIsValid(lemma string) (result bool, err error) {
	args := d.Called(lemma)
	return args.Bool(0), args.Error(1)
}

var transactionalCreation = func(t *testing.T) (data.Transactional, sqlmock.Sqlmock) {
	db, mock, err := sqlmock.New()
	if !assert.NoError(t, err, "sqlmock") {
		t.FailNow()
	}

	dataMysql := transactional.NewTransactional(db)

	return dataMysql, mock
}

func TestApplicationNewGame(t *testing.T) {
	type DataTest struct {
		Usernames   []string
		BoardSize   uint8
		MaxStrength uint8
	}

	// len(usernames) >= 2
	usernames := []string{"sarjono", "mukti"}

	// boardSize >= 5
	boardSize := uint8(5)

	// maximumStrength >= 2
	maxStrength := uint8(2)

	ctx := context.TODO()

	t.Run("Success", func(t *testing.T) {
		dt, sqlMock := transactionalCreation(t)
		playersColumn := []string{"id", "username"}
		sqlMock.ExpectQuery("SELECT (.+) FROM players").
			WithArgs("('sarjono','mukti')").
			WillReturnRows(
				sqlmock.NewRows(playersColumn).
					AddRow(1, "sarjono").
					AddRow(2, "mukti"),
			)

		sqlMock.ExpectBegin()
		sqlMock.ExpectExec("INSERT INTO games").
			WithArgs(uint64(1), sqlmock.AnyArg(), make([]uint8, boardSize*boardSize), maxStrength).
			WillReturnResult(sqlmock.NewResult(1, 1))
		sqlMock.ExpectExec("INSERT INTO game_player").
			WithArgs(1, 1, 1, 2).
			WillReturnResult(sqlmock.NewResult(1, 1))
		sqlMock.ExpectCommit()

		application := letter_block.NewApplication(dt, make(map[string]dictionary.Dictionary))
		game, err := application.NewGame(ctx, usernames, boardSize, maxStrength)
		if !assert.NoError(t, err, "not expecting any error") {
			t.FailNow()
		}

		assert.Equal(t, uint64(1), game.CurrentPlayerID, "define first turn")
		assert.Equal(t, maxStrength, game.MaxStrength, "fixed maximum strength")
		assert.ElementsMatch(t, make([]uint8, boardSize*boardSize), game.BoardPositioning, "no player own each slot of the board")
		if assert.Len(t, game.BoardBase, int(boardSize*boardSize), "number of slot") {
			assert.NotEqual(t, make([]uint8, boardSize*boardSize), game.BoardBase, "slot should be randomized")
		}

		if assert.Len(t, game.Players, len(usernames), "number of player shuld equal") {
			assert.Condition(t, func() (success bool) {
				for i, player := range game.Players {
					if player.Username != usernames[i] {
						return false
					}
				}
				return true
			}, "username should arranged as it is")
		}
		assert.Equal(t, uint64(1), game.ID, "gameID")
	})
	t.Run("ValidationError", func(t *testing.T) {
		t.Run("NonDependencyError", func(t *testing.T) {
			dt, _ := transactionalCreation(t)
			application := letter_block.NewApplication(dt, make(map[string]dictionary.Dictionary))

			for _, testCase := range []struct {
				Name          string
				DataTests     []DataTest
				ExpectedError error
			}{
				{
					"PlayerLessThanTwo",
					[]DataTest{
						{[]string{"sarjono"}, boardSize, maxStrength},
						{[]string{}, boardSize, maxStrength},
					},
					letter_block.ErrorPlayerInsufficient,
				},
				{
					"BoardSizeLessThanFive",
					[]DataTest{
						{usernames, 4, maxStrength},
						{usernames, 3, maxStrength},
						{usernames, 2, maxStrength},
						{usernames, 1, maxStrength},
						{usernames, 0, maxStrength},
					},
					letter_block.ErrorBoardSizeInsufficient,
				},
				{
					"MaxStrengthLessThanTwo",
					[]DataTest{
						{usernames, boardSize, 1},
						{usernames, boardSize, 0},
					},
					letter_block.ErrorMaximumStrengthInsufficient,
				},
			} {
				t.Run(testCase.Name, func(t *testing.T) {
					for i, dataTest := range testCase.DataTests {
						t.Run(fmt.Sprintf("Case%v", i), func(t *testing.T) {
							_, err := application.NewGame(ctx, dataTest.Usernames, dataTest.BoardSize, dataTest.MaxStrength)
							assert.EqualError(t, err, testCase.ExpectedError.Error(), "expecting an error")
						})
					}
				})
			}
		})
		t.Run("NumberOfPlayerError", func(t *testing.T) {
			type DataTestWithPreparation struct {
				DataTest
				Preparation func() *letter_block.Application
			}

			for _, testCase := range []struct {
				Name          string
				DataTests     []DataTestWithPreparation
				ExpectedError error
			}{
				{
					"ThereIsPlayerNotFound",
					[]DataTestWithPreparation{
						{
							DataTest{[]string{"notfound", "sarjono"}, boardSize, maxStrength},
							func() *letter_block.Application {
								dt, sqlMock := transactionalCreation(t)
								playersColumn := []string{"id", "username"}

								application := letter_block.NewApplication(dt, make(map[string]dictionary.Dictionary))

								sqlMock.ExpectQuery("SELECT (.+) FROM players").
									WithArgs("('notfound','sarjono')").
									WillReturnRows(sqlmock.NewRows(playersColumn).
										AddRow(1, "sarjono"))

								return application
							},
						},
						{
							DataTest{[]string{"sarjono", "notfound"}, boardSize, maxStrength},
							func() *letter_block.Application {
								dt, sqlMock := transactionalCreation(t)
								playersColumn := []string{"id", "username"}

								application := letter_block.NewApplication(dt, make(map[string]dictionary.Dictionary))

								sqlMock.ExpectQuery("SELECT (.+) FROM players").
									WithArgs("('sarjono','notfound')").
									WillReturnRows(sqlmock.NewRows(playersColumn).
										AddRow(1, "sarjono"))

								return application
							},
						},
					},
					letter_block.ErrorPlayerNotFound,
				},
			} {
				t.Run(testCase.Name, func(t *testing.T) {
					for i, dataTest := range testCase.DataTests {
						t.Run(fmt.Sprintf("Case%v", i), func(t *testing.T) {
							application := dataTest.Preparation()
							_, err := application.NewGame(ctx, dataTest.Usernames, dataTest.BoardSize, dataTest.MaxStrength)
							assert.EqualError(t, err, testCase.ExpectedError.Error(), "expecting an error")
						})
					}
				})
			}
		})
	})
	t.Run("UnexpectedError", func(t *testing.T) {
		t.Run("FromQueryingPlayer", func(t *testing.T) {
			dt, sqlMock := transactionalCreation(t)

			unexpectedError := errors.New("select from players unexpected error")
			sqlMock.ExpectQuery("SELECT (.+) FROM players").
				WillReturnError(unexpectedError)

			application := letter_block.NewApplication(dt, make(map[string]dictionary.Dictionary))
			_, err := application.NewGame(ctx, usernames, boardSize, maxStrength)
			assert.EqualError(t, err, unexpectedError.Error(), "unexpected error")
		})
		t.Run("FromInsertingGame", func(t *testing.T) {
			unexpectedError := errors.New("insert into games unexpected error")
			testSuite := func(rollbackExpectation func(sqlmock.Sqlmock) error) {
				dt, sqlMock := transactionalCreation(t)

				playersColumn := []string{"id", "username"}
				sqlMock.ExpectQuery("SELECT (.+) FROM players").
					WithArgs("('sarjono','mukti')").
					WillReturnRows(
						sqlmock.NewRows(playersColumn).
							AddRow(1, "sarjono").
							AddRow(2, "mukti"),
					)
				sqlMock.ExpectBegin()
				sqlMock.ExpectExec("INSERT INTO games").
					WillReturnError(unexpectedError)

				expectedError := rollbackExpectation(sqlMock)

				application := letter_block.NewApplication(dt, make(map[string]dictionary.Dictionary))
				_, err := application.NewGame(ctx, usernames, boardSize, maxStrength)
				assert.EqualError(t, err, expectedError.Error(), "unexpected error")
			}
			t.Run("RollbackFailed", func(t *testing.T) {
				testSuite(func(sqlMock sqlmock.Sqlmock) error {
					rollbackError := errors.New("rollback unexpected error")
					sqlMock.ExpectRollback().WillReturnError(rollbackError)
					return rollbackError
				})
			})
			t.Run("RollbackSuccess", func(t *testing.T) {
				testSuite(func(sqlMock sqlmock.Sqlmock) error {
					sqlMock.ExpectRollback()
					return unexpectedError
				})
			})
		})
		t.Run("FromInsertingGamePlayer", func(t *testing.T) {
			unexpectedError := errors.New("insert into game_player unexpected error")
			testSuite := func(rollbackExpectation func(sqlmock.Sqlmock) error) {
				dt, sqlMock := transactionalCreation(t)

				playersColumn := []string{"id", "username"}
				sqlMock.ExpectQuery("SELECT (.+) FROM players").
					WithArgs("('sarjono','mukti')").
					WillReturnRows(
						sqlmock.NewRows(playersColumn).
							AddRow(1, "sarjono").
							AddRow(2, "mukti"),
					)
				sqlMock.ExpectBegin()
				sqlMock.ExpectExec("INSERT INTO games").
					WithArgs(1, sqlmock.AnyArg(), make([]uint8, boardSize*boardSize), maxStrength).
					WillReturnResult(sqlmock.NewResult(1, 1))
				sqlMock.ExpectExec("INSERT INTO game_player").
					WillReturnError(unexpectedError)

				expectedError := rollbackExpectation(sqlMock)

				application := letter_block.NewApplication(dt, make(map[string]dictionary.Dictionary))
				_, err := application.NewGame(ctx, usernames, boardSize, maxStrength)
				assert.EqualError(t, err, expectedError.Error(), "unexpected error")
			}
			t.Run("RollbackFailed", func(t *testing.T) {
				testSuite(func(sqlMock sqlmock.Sqlmock) error {
					rollbackError := errors.New("rollback unexpected error")
					sqlMock.ExpectRollback().WillReturnError(rollbackError)
					return rollbackError
				})
			})
			t.Run("RollbackSuccess", func(t *testing.T) {
				testSuite(func(sqlMock sqlmock.Sqlmock) error {
					sqlMock.ExpectRollback()
					return unexpectedError
				})
			})
		})
		t.Run("FromCommit", func(t *testing.T) {
			dt, sqlMock := transactionalCreation(t)

			playersColumn := []string{"id", "username"}
			sqlMock.ExpectQuery("SELECT (.+) FROM players").
				WithArgs("('sarjono','mukti')").
				WillReturnRows(
					sqlmock.NewRows(playersColumn).
						AddRow(1, "sarjono").
						AddRow(2, "mukti"),
				)
			sqlMock.ExpectBegin()
			sqlMock.ExpectExec("INSERT INTO games").
				WithArgs(uint64(1), sqlmock.AnyArg(), make([]uint8, boardSize*boardSize), maxStrength).
				WillReturnResult(sqlmock.NewResult(1, 1))
			sqlMock.ExpectExec("INSERT INTO game_player").
				WithArgs(1, 1, 1, 2).
				WillReturnResult(sqlmock.NewResult(1, 1))
			unexpectedError := errors.New("commit error")
			sqlMock.ExpectCommit().
				WillReturnError(unexpectedError)

			application := letter_block.NewApplication(dt, make(map[string]dictionary.Dictionary))
			_, err := application.NewGame(ctx, usernames, boardSize, maxStrength)
			assert.EqualError(t, err, unexpectedError.Error(), "unexpected error")
		})
	})
}

func TestApplicationTakeTurn(t *testing.T) {
	ctx := context.TODO()
	gameID := uint64(1)
	gamePlayerID := uint64(1)
	playerID := uint64(1)
	word := []uint16{0, 1, 2, 3}
	boardBase := []uint8{22, 14, 17, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24}

	t.Run("ValidationError", func(t *testing.T) {
		t.Run("UnauthorizedError", func(t *testing.T) {
			dt, sqlMock := transactionalCreation(t)

			gamePlayerColumn := []string{"game_id", "player_id"}
			sqlMock.ExpectQuery("SELECT (.+) FROM game_player").
				WithArgs(1).
				WillReturnRows(
					sqlmock.NewRows(gamePlayerColumn).
						AddRow(1, 2),
				)

			application := letter_block.NewApplication(dt, make(map[string]dictionary.Dictionary))
			_, err := application.TakeTurn(ctx, gamePlayerID, playerID, word)
			assert.EqualError(t, err, letter_block.ErrorUnauthorized.Error(), "unauthorized error")
		})
		t.Run("GamePlayerIDNotFoundError", func(t *testing.T) {
			dt, sqlMock := transactionalCreation(t)

			gamePlayerColumn := []string{"game_id", "player_id"}
			sqlMock.ExpectQuery("SELECT (.+) FROM game_player").
				WithArgs(1).
				WillReturnRows(
					sqlmock.NewRows(gamePlayerColumn),
				)

			application := letter_block.NewApplication(dt, make(map[string]dictionary.Dictionary))
			_, err := application.TakeTurn(ctx, gamePlayerID, playerID, word)
			assert.EqualError(t, err, sql.ErrNoRows.Error(), "unauthorized error")
		})
		t.Run("NotYourTurn", func(t *testing.T) {
			dt, sqlMock := transactionalCreation(t)

			gamePlayerColumn := []string{"game_id", "player_id"}
			sqlMock.ExpectQuery("SELECT (.+) FROM game_player").
				WithArgs(gamePlayerID).
				WillReturnRows(
					sqlmock.NewRows(gamePlayerColumn).
						AddRow(gameID, playerID),
				)

			sqlMock.ExpectBegin()
			gameColumn := []string{"current_player_id", "board_base"}
			sqlMock.ExpectQuery("SELECT (.+) FROM games").
				WithArgs(gameID).
				WillReturnRows(
					sqlmock.NewRows(gameColumn).
						AddRow(playerID+1, boardBase),
				)
			sqlMock.ExpectRollback()

			application := letter_block.NewApplication(dt, make(map[string]dictionary.Dictionary))
			_, err := application.TakeTurn(ctx, gamePlayerID, playerID, word)
			assert.EqualError(t, err, letter_block.ErrorNotYourTurn.Error(), "not your turn error")
		})
		t.Run("DoesntMakeWordError", func(t *testing.T) {
			dt, sqlMock := transactionalCreation(t)

			gamePlayerColumn := []string{"game_id", "player_id"}
			sqlMock.ExpectQuery("SELECT (.+) FROM game_player").
				WithArgs(gamePlayerID).
				WillReturnRows(
					sqlmock.NewRows(gamePlayerColumn).
						AddRow(gameID, playerID),
				)

			sqlMock.ExpectBegin()
			gameColumn := []string{"current_player_id", "board_base"}
			sqlMock.ExpectQuery("SELECT (.+) FROM games").
				WithArgs(gameID).
				WillReturnRows(
					sqlmock.NewRows(gameColumn).
						AddRow(playerID, boardBase),
				)
			sqlMock.ExpectRollback()

			application := letter_block.NewApplication(dt, make(map[string]dictionary.Dictionary))
			_, err := application.TakeTurn(ctx, gamePlayerID, playerID, []uint16{0, 1, 0})
			assert.EqualError(t, err, letter_block.ErrorDoesntMakeWord.Error(), "doesnt make word error")
		})
	})
	t.Run("UnexpectedError", func(t *testing.T) {
		t.Run("FromQueryingGamePlayer", func(t *testing.T) {
			dt, sqlMock := transactionalCreation(t)

			unexpectedError := errors.New("unexpected error")
			sqlMock.ExpectQuery("SELECT (.+) FROM game_player").
				WithArgs(1).
				WillReturnError(unexpectedError)

			application := letter_block.NewApplication(dt, make(map[string]dictionary.Dictionary))
			_, err := application.TakeTurn(ctx, gamePlayerID, playerID, word)
			assert.EqualError(t, err, unexpectedError.Error(), "unauthorized error")
		})
		t.Run("FromBeginTransaction", func(t *testing.T) {
			dt, sqlMock := transactionalCreation(t)

			gamePlayerColumn := []string{"game_id", "player_id"}
			sqlMock.ExpectQuery("SELECT (.+) FROM game_player").
				WithArgs(gamePlayerID).
				WillReturnRows(
					sqlmock.NewRows(gamePlayerColumn).
						AddRow(gameID, playerID),
				)

			unexpectedError := errors.New("unexpected error")
			sqlMock.ExpectBegin().
				WillReturnError(unexpectedError)

			application := letter_block.NewApplication(dt, make(map[string]dictionary.Dictionary))
			_, err := application.TakeTurn(ctx, gamePlayerID, playerID, word)
			assert.EqualError(t, err, unexpectedError.Error(), "unauthorized error")
		})
		t.Run("FromQueryingGame", func(t *testing.T) {
			dt, sqlMock := transactionalCreation(t)

			gamePlayerColumn := []string{"game_id", "player_id"}
			sqlMock.ExpectQuery("SELECT (.+) FROM game_player").
				WithArgs(gamePlayerID).
				WillReturnRows(
					sqlmock.NewRows(gamePlayerColumn).
						AddRow(gameID, playerID),
				)

			sqlMock.ExpectBegin()
			unexpectedError := errors.New("unexpected error")
			sqlMock.ExpectQuery("SELECT (.+) FROM games").
				WithArgs(gameID).
				WillReturnError(unexpectedError)
			sqlMock.ExpectRollback()

			application := letter_block.NewApplication(dt, make(map[string]dictionary.Dictionary))
			_, err := application.TakeTurn(ctx, gamePlayerID, playerID, word)
			assert.EqualError(t, err, unexpectedError.Error(), "unauthorized error")
		})
	})
	t.Run("ErrorValidatingLemma", func(t *testing.T) {
		unexpectedError := errors.New("unexpected error")
		dt, sqlMock := transactionalCreation(t)

		gamePlayerColumn := []string{"game_id", "player_id"}
		sqlMock.ExpectQuery("SELECT (.+) FROM game_player").
			WithArgs(gamePlayerID).
			WillReturnRows(
				sqlmock.NewRows(gamePlayerColumn).
					AddRow(gameID, playerID),
			)

		sqlMock.ExpectBegin()
		gameColumn := []string{"current_player_id", "board_base"}
		sqlMock.ExpectQuery("SELECT (.+) FROM games").
			WithArgs(gameID).
			WillReturnRows(
				sqlmock.NewRows(gameColumn).
					AddRow(playerID, boardBase),
			)
		sqlMock.ExpectRollback()

		dict := &Dictionary{}
		dictionaries := map[string]dictionary.Dictionary{
			"id-id": dict,
		}
		dict.On("LemmaIsValid", "word").
			Return(false, unexpectedError)

		application := letter_block.NewApplication(dt, dictionaries)
		_, err := application.TakeTurn(ctx, gamePlayerID, playerID, word)
		assert.EqualError(t, err, unexpectedError.Error(), "unauthorized error")
	})
	t.Run("ErrorWordInvalid", func(t *testing.T) {
		dt, sqlMock := transactionalCreation(t)

		gamePlayerColumn := []string{"game_id", "player_id"}
		sqlMock.ExpectQuery("SELECT (.+) FROM game_player").
			WithArgs(gamePlayerID).
			WillReturnRows(
				sqlmock.NewRows(gamePlayerColumn).
					AddRow(gameID, playerID),
			)

		sqlMock.ExpectBegin()
		gameColumn := []string{"current_player_id", "board_base"}
		sqlMock.ExpectQuery("SELECT (.+) FROM games").
			WithArgs(gameID).
			WillReturnRows(
				sqlmock.NewRows(gameColumn).
					AddRow(playerID, boardBase),
			)
		sqlMock.ExpectRollback()

		dict := &Dictionary{}
		dictionaries := map[string]dictionary.Dictionary{
			"id-id": dict,
		}
		dict.On("LemmaIsValid", "word").
			Return(false, nil)

		application := letter_block.NewApplication(dt, dictionaries)
		_, err := application.TakeTurn(ctx, gamePlayerID, playerID, word)
		assert.EqualError(t, err, letter_block.ErrorWordInvalid.Error(), "invalid word")
	})
}
