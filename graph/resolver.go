package graph

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require here.

import (
	"log"
	"strconv"
	"sync"

	"github.com/satriahrh/letter-block"
	"github.com/satriahrh/letter-block/data"
	"github.com/satriahrh/letter-block/graph/model"
)

const (
	GAME_ID_BASE   = 36
	PLAYER_ID_BASE = 36
)

type Resolver struct {
	application    letter_block.LogicOfApplication
	mutex          sync.Mutex
	gameSubscriber map[data.GameId]map[data.PlayerId]GameSubscriber
}

type GameSubscriber chan *model.Game

func NewResolver(application letter_block.LogicOfApplication) *Resolver {
	return &Resolver{
		application,
		sync.Mutex{},
		make(map[data.GameId]map[data.PlayerId]GameSubscriber),
	}
}

func serializeGames(games []data.Game) []*model.Game {
	serializedGames := make([]*model.Game, len(games))
	for i, game := range games {
		serializedGames[i] = serializeGame(game)
	}

	return serializedGames
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
		WordPlayed:     serializeWordPlayeds(game.PlayedWords),
		Players:        serializePlayers(game.Players),
	}
}

func serializeWordPlayeds(playedWords []data.PlayedWord) []*model.WordPlayed {
	serializedWordPlayeds := make([]*model.WordPlayed, len(playedWords))
	for i, playedWord := range playedWords {
		serializedWordPlayeds[i] = &model.WordPlayed{
			Player: serializePlayer(data.Player{Id: playedWord.PlayerId}),
			Word:   playedWord.Word,
		}
	}
	return serializedWordPlayeds
}

func serializePlayers(players []data.Player) []*model.Player {
	serializedPlayers := make([]*model.Player, len(players))
	for i, player := range players {
		serializedPlayers[i] = serializePlayer(player)
	}
	return serializedPlayers
}

func serializePlayer(player data.Player) *model.Player {
	return &model.Player{
		ID: strconv.FormatUint(uint64(player.Id), PLAYER_ID_BASE),
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
