package service

import (
	"context"
	"log"

	"github.com/satriahrh/letter-block/data"
)

func (a *application) GetGame(ctx context.Context, gameId data.GameId) (game data.Game, err error) {
	game, err = a.transactional.GetGameById(ctx, nil, gameId)
	if err != nil {
		log.Println(err)
		return
	}

	playedWordsChan := make(chan bool)
	playersChan := make(chan bool)

	go func() {
		game.PlayedWords, _ = a.transactional.GetPlayedWordsByGameId(ctx, gameId)
		playedWordsChan <- true
	}()

	go func() {
		game.Players, _ = a.transactional.GetPlayersByGameId(ctx, gameId)
		playersChan <- true
	}()

	<-playedWordsChan
	<-playersChan

	return
}
