package graph

import (
	"context"
) // THIS CODE IS A STARTING POINT ONLY. IT WILL NOT BE UPDATED WITH SCHEMA CHANGES.

type Resolver struct{}

func (r *Resolver) Mutation() MutationResolver {
	return &mutationResolver{r}
}
func (r *Resolver) Query() QueryResolver {
	return &queryResolver{r}
}
func (r *Resolver) Subscription() SubscriptionResolver {
	return &subscriptionResolver{r}
}

type mutationResolver struct{ *Resolver }

func (r *mutationResolver) NewGame(ctx context.Context, input NewGame) (*Game, error) {
	panic("not implemented")
}

func (r *mutationResolver) TakeTurn(ctx context.Context, input TakeTurn) (*Game, error) {
	panic("not implemented error")
}

type queryResolver struct{ *Resolver }

func (r *queryResolver) MyGames(ctx context.Context) ([]*Game, error) {
	panic("not implemented")
}

func (r *queryResolver) GetGame(ctx context.Context, gameID string) (*Game, error) {
	panic("not implemented")
}

type subscriptionResolver struct{ *Resolver }

func (r *subscriptionResolver) ListenGame(ctx context.Context, gameID string) (<-chan *Game, error) {
	panic("not implemented")
}
