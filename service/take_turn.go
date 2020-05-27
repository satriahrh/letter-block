package service

import (
	"context"
	"math/rand"
	"regexp"

	"github.com/satriahrh/letter-block/data"
)

func (a *application) TakeTurn(ctx context.Context, gameId data.GameId, playerId data.PlayerId, word []uint8) (game data.Game, err error) {
	tx, err := a.transactional.BeginTransaction(ctx)
	if err != nil {
		return
	}
	defer func() {
		err = a.transactional.FinalizeTransaction(tx, err)
	}()

	game, err = a.transactional.GetGameById(ctx, tx, gameId)
	if err != nil {
		return
	}

	if game.State != data.ONGOING {
		err = ErrorGameIsUnplayable
		return
	}

	gamePlayers, err := a.transactional.GetGamePlayersByGameId(ctx, tx, gameId)
	if err != nil {
		return
	}

	if uint8(len(gamePlayers))-1 < game.CurrentPlayerOrder { // waiting for other player
		err = ErrorNotYourTurn
		return
	} else if gamePlayers[game.CurrentPlayerOrder].PlayerId != playerId { // not your turn
		err = ErrorNotYourTurn
		return
	}

	wordOnce := make(map[uint8]bool)
	wordByte := make([]byte, len(word))
	for i, wordPosition := range word {
		if wordOnce[wordPosition] {
			err = ErrorDoesntMakeWord
			return
		} else {
			wordOnce[wordPosition] = true
		}
		wordByte[i] = alphabet[game.BoardBase[wordPosition]]
		game.BoardBase[wordPosition] = uint8(rand.Uint64() % 26)
	}

	wordString := string(wordByte)
	var valid bool
	valid, err = a.dictionaries["id-id"].LemmaIsValid(wordString)
	if err != nil {
		return
	}
	if !valid {
		err = ErrorWordInvalid
		return
	}

	err = a.transactional.LogPlayedWord(ctx, tx, game.Id, playerId, wordString)
	if err != nil {
		if exist, _ := regexp.MatchString("Error 2601", err.Error()); exist {
			err = ErrorWordHavePlayed
		}
		return
	}

	positioningSpace := uint8(len(gamePlayers)) + 1
	for _, position := range word {
		boardPosition := game.BoardPositioning[position]
		if boardPosition == 0 {
			game.BoardPositioning[position] = game.CurrentPlayerOrder + 1
		} else {
			ownedBy := boardPosition % positioningSpace
			currentStrength := boardPosition/positioningSpace + 1
			if ownedBy == game.CurrentPlayerOrder+1 {
				if currentStrength < maxStrength {
					game.BoardPositioning[position] += positioningSpace
				}
			} else {
				if currentStrength > 1 {
					game.BoardPositioning[position] -= positioningSpace
				} else {
					game.BoardPositioning[position] = game.CurrentPlayerOrder + 1
				}
			}
		}
	}

	game.CurrentPlayerOrder += 1
	if game.CurrentPlayerOrder >= game.NumberOfPlayer {
		game.CurrentPlayerOrder = 0
	}

	if gameIsEnding(game) {
		game.State = data.END
	}

	err = a.transactional.UpdateGame(ctx, tx, game)
	if err != nil {
		return
	}

	return
}

func gameIsEnding(game data.Game) bool {
	for _, positioning := range game.BoardPositioning {
		if positioning == 0 {
			return false
		}
	}
	return true
}
