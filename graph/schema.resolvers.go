package graph

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.

import (
	"context"

	"github.com/satriahrh/letter-block/data"
	"github.com/satriahrh/letter-block/graph/generated"
	"github.com/satriahrh/letter-block/graph/model"
	"github.com/satriahrh/letter-block/middleware/auth"
)

func (r *mutationResolver) NewGame(ctx context.Context, input model.NewGame) (*model.Game, error) {
	user := auth.ForContext(ctx)

	game, err := r.application.NewGame(ctx, user.PlayerId, uint8(input.NumberOfPlayer))
	if err != nil {
		return nil, err
	}

	return serializeGame(game), nil
}

func (r *mutationResolver) TakeTurn(ctx context.Context, input model.TakeTurn) (*model.Game, error) {
	user := auth.ForContext(ctx)

	gameId := parseGameId(input.GameID)
	word := parseWord(input.Word)

	game, err := r.application.TakeTurn(ctx, gameId, user.PlayerId, word)
	if err != nil {
		return nil, err
	}

	serializedGame := serializeGame(game)

	r.mutex.Lock()
	if len(r.gameSubscriber[gameId]) > 0 {
		game, err = r.application.GetGame(ctx, gameId)
		if err != nil {
			r.mutex.Unlock()
			return serializedGame, nil
		}
		serializedGame = serializeGame(game)
	}
	for _, subscriber := range r.gameSubscriber[gameId] {
		subscriber <- serializedGame
	}
	r.mutex.Unlock()

	if game.State != data.END {
		delete(r.gameSubscriber, gameId)
	}

	return serializedGame, nil
}

func (r *mutationResolver) JoinGame(ctx context.Context, input model.JoinGame) (*model.Game, error) {
	user := auth.ForContext(ctx)

	gameId := parseGameId(input.GameID)

	game, err := r.application.Join(ctx, gameId, user.PlayerId)
	if err != nil {
		return nil, err
	}

	return serializeGame(game), nil
}

func (r *queryResolver) MyGames(ctx context.Context) ([]*model.Game, error) {
	user := auth.ForContext(ctx)

	games, err := r.application.GetGames(ctx, user.PlayerId)
	if err != nil {
		return nil, err
	}

	return serializeGames(games), nil
}

func (r *queryResolver) GetGame(ctx context.Context, gameID string) (*model.Game, error) {
	gameId := parseGameId(gameID)

	game, err := r.application.GetGame(ctx, gameId)
	if err != nil {
		return nil, err
	}

	return serializeGame(game), nil
}

func (r *subscriptionResolver) ListenGame(ctx context.Context, gameID string) (<-chan *model.Game, error) {
	gameId := parseGameId(gameID)
	if _, err := r.application.GetGame(ctx, gameId); err != nil {
		return nil, err
	}
	user := auth.ForContext(ctx)

	r.mutex.Lock()
	{
		gameSubscriber := make(GameSubscriber, 1)
		if r.gameSubscriber[gameId] == nil {
			r.gameSubscriber[gameId] = make(map[data.PlayerId]GameSubscriber)
		}
		r.gameSubscriber[gameId][user.PlayerId] = gameSubscriber
	}
	r.mutex.Unlock()

	go func() {
		<-ctx.Done()
		r.mutex.Lock()
		delete(r.gameSubscriber[gameId], user.PlayerId)
		r.mutex.Unlock()
	}()

	return r.gameSubscriber[gameId][user.PlayerId], nil
}

// Mutation returns generated.MutationResolver implementation.
func (r *Resolver) Mutation() generated.MutationResolver { return &mutationResolver{r} }

// Query returns generated.QueryResolver implementation.
func (r *Resolver) Query() generated.QueryResolver { return &queryResolver{r} }

// Subscription returns generated.SubscriptionResolver implementation.
func (r *Resolver) Subscription() generated.SubscriptionResolver { return &subscriptionResolver{r} }

type mutationResolver struct{ *Resolver }
type queryResolver struct{ *Resolver }
type subscriptionResolver struct{ *Resolver }
