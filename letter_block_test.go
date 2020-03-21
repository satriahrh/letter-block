package letter_block_test

import (
	"context"
	"fmt"
	"github.com/satriahrh/letter-block"
	"github.com/satriahrh/letter-block/data"
	"github.com/satriahrh/letter-block/mock"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestApplicationNewGame(t *testing.T) {
	type DataTest struct {
		Usernames   []string
		BoardSize   uint
		MaxStrength uint
	}

	// len(usernames) >= 2
	usernames := []string{"sarjono", "mukti"}

	// boardSize >= 5
	boardSize := uint(5)

	// maximumStrength >= 2
	maxStrength := uint(2)

	ctx := context.TODO()
	dt, _ := data.NewData(&mock.Mysql{})
	application, _ := letter_block.NewApplication(dt)

	suiteApplicationNewGameSuccess := func(t *testing.T, usernames []string, boardSize, maximumStrength uint) {
		game, err := application.NewGame(ctx, usernames, boardSize, maximumStrength)
		if err != nil {
			t.Fatalf("not expecting any error, found %v", err)
		}

		if game.CurrentTurn != 0 {
			t.Errorf("CurrentTurn, expected 0, got %v", game.CurrentTurn)
		}

		if game.MaximumStrength != maximumStrength {
			t.Errorf("MaximumStrength,expected %v, got %v", maximumStrength, maximumStrength)
		}

		if uint(len(game.Board)) != boardSize {
			t.Errorf("board size expected %v, got %v", boardSize, len(game.Board))
		} else {
			for _, row := range game.Board {
				if uint(len(row)) != boardSize {
					t.Errorf("there is board row size expected %v, got %v", boardSize, len(row))
				}
			}
		}

		actualUsernames := make([]string, len(game.Players))
		for i, player := range game.Players {
			actualUsernames[i] = player.Username
		}
		assert.ElementsMatch(t, actualUsernames, usernames, "username doesn't matched")
	}

	t.Run("Success", func(t *testing.T) {
		suiteApplicationNewGameSuccess(t, usernames, boardSize, maxStrength)
	})

	t.Run("ExpectedError", func(t *testing.T) {
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
				"ThereIsPlayerNotFound",
				[]DataTest{
					{[]string{"notfound", "sarjono"}, boardSize, maxStrength},
					{[]string{"sarjono", "notfound"}, boardSize, maxStrength},
				},
				letter_block.ErrorPlayerNotFound,
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

	t.Run("UnexpectedError", func(t *testing.T) {
		for _, testCase := range [] struct {
			Name          string
			DataTests     []DataTest
		}{
			{
				"QueryingPlayer",
				[]DataTest{
					{append(usernames, "unexpected"), boardSize, maxStrength},
				},
			},
		} {
			t.Run(testCase.Name, func(t *testing.T) {
				for i, dataTest := range testCase.DataTests {
					t.Run(fmt.Sprintf("Case%v", i), func(t *testing.T) {
						_, err := application.NewGame(ctx, dataTest.Usernames, dataTest.BoardSize, dataTest.MaxStrength)
						assert.Error(t, err, "expecting unexpected error")
					})
				}
			})
		}
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
