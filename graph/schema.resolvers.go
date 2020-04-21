package graph

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.

import (
	"context"
	"fmt"

	"github.com/satriahrh/letter-block/graph/generated"
	"github.com/satriahrh/letter-block/graph/model"
)

func (r *mutationResolver) NewGame(ctx context.Context, input model.NewGame) (*model.Game, error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *mutationResolver) TakeTurn(ctx context.Context, input model.TakeTurn) (*model.Game, error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *queryResolver) MyGames(ctx context.Context) ([]*model.Game, error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *queryResolver) GetGame(ctx context.Context, gameID string) (*model.Game, error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *subscriptionResolver) ListenGame(ctx context.Context, gameID string) (<-chan *model.Game, error) {
	panic(fmt.Errorf("not implemented"))
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