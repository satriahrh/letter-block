package graph

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require here.

import (
	"log"
	"strconv"

	"github.com/satriahrh/letter-block"
	"github.com/satriahrh/letter-block/data"
	"github.com/satriahrh/letter-block/graph/model"
)

const (
	GAME_ID_BASE = 64
)

type Resolver struct {
	application letter_block.LogicOfApplication
}

func NewResolver(application letter_block.LogicOfApplication) *Resolver {
	return &Resolver{application}
}

func serializeGame(game data.Game) *model.Game {
	return &model.Game{
		ID:                 strconv.FormatUint(uint64(game.Id), GAME_ID_BASE),
		CurrentPlayerOrder: int(game.CurrentPlayerOrder),
		BoardBase: func() []int {
			boardBase := make([]int, len(game.BoardBase))
			for i, bb := range game.BoardBase {
				boardBase[i] = int(bb)
			}
			return boardBase
		}(),
		BoardPositioning: func() []int {
			boardPositioning := make([]int, len(game.BoardPositioning))
			for i, bp := range game.BoardPositioning {
				boardPositioning[i] = int(bp)
			}
			return boardPositioning
		}(),
		NumberOfPlayer: int(game.NumberOfPlayer),
	}
}

func parseGameId(rawGameId string) data.GameId {
	gameId, err := strconv.ParseUint(rawGameId, GAME_ID_BASE, 64)
	if err != nil {
		log.Println(err)
		return data.GameId(0)
	}
	return data.GameId(gameId)
}

func parseWord(rawWord []int) []uint8 {
	word := make([]uint8, len(rawWord))
	for i, w := range rawWord {
		word[i] = uint8(w)
	}
	return word
}
