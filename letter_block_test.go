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
		application, _ := letter_block.NewApplication(dt)
		game, err := application.NewGame(ctx, usernames, boardSize, maxStrength)
		if !assert.NoError(t, err, "not expecting any error") {
			t.FailNow()
		}

		assert.Zero(t, game.CurrentTurn, "define first turn")
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
	})

	t.Run("ValidationError", func(t *testing.T) {
		t.Run("NonDependencyError", func(t *testing.T) {
			dt, _ := dataCreation(t)
			application, _ := letter_block.NewApplication(dt)

			for _, testCase := range [] struct {
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

			for _, testCase := range [] struct {
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

								application, _ := letter_block.NewApplication(dt)

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

								application, _ := letter_block.NewApplication(dt)

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
			mock.ExpectQuery("SELECT (.+) FROM players").
				WillReturnError(errors.New("unexpected error"))

			application, _ := letter_block.NewApplication(dt)
			_, err := application.NewGame(ctx, usernames, boardSize, maxStrength)
			assert.Error(t, err, "expecting unexpected error")

		})
	})
}

func TestApplicationTakeTurn(t *testing.T) {
	t.Run("SuccessContinue", func(t *testing.T) {

	})

	t.Run("SuccessVictory", func(t *testing.T) {

	})

	t.Run("NotHisTurn", func(t *testing.T) {

	})

	t.Run("NotValid", func(t *testing.T) {

	})
}
