package letter_block_test

import (
	"context"
	"errors"
	"fmt"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/satriahrh/letter-block"
	"github.com/satriahrh/letter-block/data"
	"github.com/stretchr/testify/assert"
	"testing"
)

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

	dataCreation := func(t *testing.T) (*data.Data, sqlmock.Sqlmock) {
		db, mock, err := sqlmock.New()
		if !assert.NoError(t, err, "sqlmock") {
			t.FailNow()
		}

		dataMysql := &data.Mysql{
			DB: db,
		}
		dt, err := data.NewData(dataMysql)
		if !assert.NoError(t, err, "newdata") {
			t.FailNow()
		}

		return dt, mock
	}

	t.Run("Success", func(t *testing.T) {
		dt, mock := dataCreation(t)
		playersColumn := []string{"id", "username"}
		mock.ExpectQuery("SELECT (.+) FROM players").
			WithArgs("('sarjono','mukti')").
			WillReturnRows(
				mock.NewRows(playersColumn).
					AddRow(1, "sarjono").
					AddRow(2, "mukti"),
			)

		mock.ExpectBegin()
		mock.ExpectExec("INSERT INTO games").
			WithArgs(uint64(1), sqlmock.AnyArg(), make([]uint8, boardSize*boardSize), maxStrength).
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectExec("INSERT INTO game_player").
			WithArgs(1, 1, 1, 2).
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectCommit()

		application := letter_block.NewApplication(dt)
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
			dt, _ := dataCreation(t)
			application := letter_block.NewApplication(dt)

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
								dt, mock := dataCreation(t)
								playersColumn := []string{"id", "username"}

								application := letter_block.NewApplication(dt)

								mock.ExpectQuery("SELECT (.+) FROM players").
									WithArgs("('notfound','sarjono')").
									WillReturnRows(mock.NewRows(playersColumn).
										AddRow(1, "sarjono"))

								return application
							},
						},
						{
							DataTest{[]string{"sarjono", "notfound"}, boardSize, maxStrength},
							func() *letter_block.Application {
								dt, mock := dataCreation(t)
								playersColumn := []string{"id", "username"}

								application := letter_block.NewApplication(dt)

								mock.ExpectQuery("SELECT (.+) FROM players").
									WithArgs("('sarjono','notfound')").
									WillReturnRows(mock.NewRows(playersColumn).
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
			dt, mock := dataCreation(t)

			unexpectedError := errors.New("select from players unexpected error")
			mock.ExpectQuery("SELECT (.+) FROM players").
				WillReturnError(unexpectedError)

			application := letter_block.NewApplication(dt)
			_, err := application.NewGame(ctx, usernames, boardSize, maxStrength)
			assert.EqualError(t, err, unexpectedError.Error(), "unexpected error")
		})
		t.Run("FromInsertingGame", func(t *testing.T) {
			unexpectedError := errors.New("insert into games unexpected error")
			testSuite := func(rollbackExpectation func(sqlmock.Sqlmock) error) {
				dt, mock := dataCreation(t)

				playersColumn := []string{"id", "username"}
				mock.ExpectQuery("SELECT (.+) FROM players").
					WithArgs("('sarjono','mukti')").
					WillReturnRows(
						mock.NewRows(playersColumn).
							AddRow(1, "sarjono").
							AddRow(2, "mukti"),
					)
				mock.ExpectBegin()
				mock.ExpectExec("INSERT INTO games").
					WillReturnError(unexpectedError)

				expectedError := rollbackExpectation(mock)

				application := letter_block.NewApplication(dt)
				_, err := application.NewGame(ctx, usernames, boardSize, maxStrength)
				assert.EqualError(t, err, expectedError.Error(), "unexpected error")
			}
			t.Run("RollbackFailed", func(t *testing.T) {
				testSuite(func(mock sqlmock.Sqlmock) error {
					rollbackError := errors.New("rollback unexpected error")
					mock.ExpectRollback().WillReturnError(rollbackError)
					return rollbackError
				})
			})
			t.Run("RollbackSuccess", func(t *testing.T) {
				testSuite(func(mock sqlmock.Sqlmock) error {
					mock.ExpectRollback()
					return unexpectedError
				})
			})
		})
		t.Run("FromInsertingGamePlayer", func(t *testing.T) {
			unexpectedError := errors.New("insert into game_player unexpected error")
			testSuite := func(rollbackExpectation func(sqlmock.Sqlmock) error) {
				dt, mock := dataCreation(t)

				playersColumn := []string{"id", "username"}
				mock.ExpectQuery("SELECT (.+) FROM players").
					WithArgs("('sarjono','mukti')").
					WillReturnRows(
						mock.NewRows(playersColumn).
							AddRow(1, "sarjono").
							AddRow(2, "mukti"),
					)
				mock.ExpectBegin()
				mock.ExpectExec("INSERT INTO games").
					WithArgs(1, sqlmock.AnyArg(), make([]uint8, boardSize*boardSize), maxStrength).
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectExec("INSERT INTO game_player").
					WillReturnError(unexpectedError)

				expectedError := rollbackExpectation(mock)

				application := letter_block.NewApplication(dt)
				_, err := application.NewGame(ctx, usernames, boardSize, maxStrength)
				assert.EqualError(t, err, expectedError.Error(), "unexpected error")
			}
			t.Run("RollbackFailed", func(t *testing.T) {
				testSuite(func(mock sqlmock.Sqlmock) error {
					rollbackError := errors.New("rollback unexpected error")
					mock.ExpectRollback().WillReturnError(rollbackError)
					return rollbackError
				})
			})
			t.Run("RollbackSuccess", func(t *testing.T) {
				testSuite(func(mock sqlmock.Sqlmock) error {
					mock.ExpectRollback()
					return unexpectedError
				})
			})
		})
		t.Run("FromCommit", func(t *testing.T) {
			dt, mock := dataCreation(t)

			playersColumn := []string{"id", "username"}
			mock.ExpectQuery("SELECT (.+) FROM players").
				WithArgs("('sarjono','mukti')").
				WillReturnRows(
					mock.NewRows(playersColumn).
						AddRow(1, "sarjono").
						AddRow(2, "mukti"),
				)
			mock.ExpectBegin()
			mock.ExpectExec("INSERT INTO games").
				WithArgs(uint64(1), sqlmock.AnyArg(), make([]uint8, boardSize*boardSize), maxStrength).
				WillReturnResult(sqlmock.NewResult(1, 1))
			mock.ExpectExec("INSERT INTO game_player").
				WithArgs(1, 1, 1, 2).
				WillReturnResult(sqlmock.NewResult(1, 1))
			unexpectedError := errors.New("commit error")
			mock.ExpectCommit().
				WillReturnError(unexpectedError)

			application := letter_block.NewApplication(dt)
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

	dataCreation := func(t *testing.T) (*data.Data, sqlmock.Sqlmock) {
		db, mock, err := sqlmock.New()
		if !assert.NoError(t, err, "sqlmock") {
			t.FailNow()
		}

		dataMysql := &data.Mysql{
			DB: db,
		}
		dt, err := data.NewData(dataMysql)
		if !assert.NoError(t, err, "newdata") {
			t.FailNow()
		}

		return dt, mock
	}

	t.Run("ValidationError", func(t *testing.T) {
		t.Run("UnauthorizedError", func(t *testing.T) {
			dt, mock := dataCreation(t)

			gamePlayerColumn := []string{"game_id", "player_id"}
			mock.ExpectQuery("SELECT (.+) FROM game_player").
				WithArgs(1).
				WillReturnRows(
					mock.NewRows(gamePlayerColumn).
						AddRow(1, 2),
				)

			application := letter_block.NewApplication(dt)
			_, err := application.TakeTurn(ctx, gamePlayerID, playerID, word)
			assert.EqualError(t, err, letter_block.ErrorUnauthorized.Error(), "unauthorized error")
		})
		t.Run("GamePlayerIDNotFoundError", func(t *testing.T) {
			dt, mock := dataCreation(t)

			gamePlayerColumn := []string{"game_id", "player_id"}
			mock.ExpectQuery("SELECT (.+) FROM game_player").
				WithArgs(1).
				WillReturnRows(
					mock.NewRows(gamePlayerColumn),
				)

			application := letter_block.NewApplication(dt)
			_, err := application.TakeTurn(ctx, gamePlayerID, playerID, word)
			assert.EqualError(t, err, letter_block.ErrorUnauthorized.Error(), "unauthorized error")
		})
		t.Run("NotYourTurn", func(t *testing.T) {
			dt, mock := dataCreation(t)

			gamePlayerColumn := []string{"game_id", "player_id"}
			mock.ExpectQuery("SELECT (.+) FROM game_player").
				WithArgs(gamePlayerID).
				WillReturnRows(
					mock.NewRows(gamePlayerColumn).
						AddRow(gameID, playerID),
				)

			mock.ExpectBegin()
			gameColumn := []string{"current_player_id", "board_base"}
			mock.ExpectQuery("SELECT (.+) FROM games").
				WithArgs(gameID).
				WillReturnRows(
					mock.NewRows(gameColumn).
						AddRow(playerID+1, boardBase),
				)
			mock.ExpectRollback()

			application := letter_block.NewApplication(dt)
			_, err := application.TakeTurn(ctx, gamePlayerID, playerID, word)
			assert.EqualError(t, err, letter_block.ErrorNotYourTurn.Error(), "not your turn error")
		})
		t.Run("DoesntMakeWordError", func(t *testing.T) {
			dt, mock := dataCreation(t)

			gamePlayerColumn := []string{"game_id", "player_id"}
			mock.ExpectQuery("SELECT (.+) FROM game_player").
				WithArgs(gamePlayerID).
				WillReturnRows(
					mock.NewRows(gamePlayerColumn).
						AddRow(gameID, playerID),
				)

			mock.ExpectBegin()
			gameColumn := []string{"current_player_id", "board_base"}
			mock.ExpectQuery("SELECT (.+) FROM games").
				WithArgs(gameID).
				WillReturnRows(
					mock.NewRows(gameColumn).
						AddRow(playerID, boardBase),
				)
			mock.ExpectRollback()

			application := letter_block.NewApplication(dt)
			_, err := application.TakeTurn(ctx, gamePlayerID, playerID, []uint16{0, 1, 0})
			assert.EqualError(t, err, letter_block.ErrorDoesntMakeWord.Error(), "doesnt make word error")
		})
	})
	t.Run("UnexpectedError", func(t *testing.T) {
		t.Run("FromQueryingGamePlayer", func(t *testing.T) {
			dt, mock := dataCreation(t)

			unexpectedError := errors.New("unexpected error")
			mock.ExpectQuery("SELECT (.+) FROM game_player").
				WithArgs(1).
				WillReturnError(unexpectedError)

			application := letter_block.NewApplication(dt)
			_, err := application.TakeTurn(ctx, gamePlayerID, playerID, word)
			assert.EqualError(t, err, unexpectedError.Error(), "unauthorized error")
		})
		t.Run("FromBeginTransaction", func(t *testing.T) {
			dt, mock := dataCreation(t)

			gamePlayerColumn := []string{"game_id", "player_id"}
			mock.ExpectQuery("SELECT (.+) FROM game_player").
				WithArgs(gamePlayerID).
				WillReturnRows(
					mock.NewRows(gamePlayerColumn).
						AddRow(gameID, playerID),
				)

			unexpectedError := errors.New("unexpected error")
			mock.ExpectBegin().
				WillReturnError(unexpectedError)

			application := letter_block.NewApplication(dt)
			_, err := application.TakeTurn(ctx, gamePlayerID, playerID, word)
			assert.EqualError(t, err, unexpectedError.Error(), "unauthorized error")
		})
		t.Run("FromQueryingGame", func(t *testing.T) {
			dt, mock := dataCreation(t)

			gamePlayerColumn := []string{"game_id", "player_id"}
			mock.ExpectQuery("SELECT (.+) FROM game_player").
				WithArgs(gamePlayerID).
				WillReturnRows(
					mock.NewRows(gamePlayerColumn).
						AddRow(gameID, playerID),
				)

			mock.ExpectBegin()
			unexpectedError := errors.New("unexpected error")
			mock.ExpectQuery("SELECT (.+) FROM games").
				WithArgs(gameID).
				WillReturnError(unexpectedError)
			mock.ExpectRollback()

			application := letter_block.NewApplication(dt)
			_, err := application.TakeTurn(ctx, gamePlayerID, playerID, word)
			assert.EqualError(t, err, unexpectedError.Error(), "unauthorized error")
		})
	})
}
