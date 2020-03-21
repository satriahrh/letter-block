package mock

import (
	"context"
	"github.com/pkg/errors"
	"github.com/satriahrh/letter-block/data"
)

var (
	errorUnexpected = errors.New("unexpected error")
	errorNotFound   = errors.New("player not found")
)

type Mysql struct {
}

func (m *Mysql) GetPlayerByUsername(ctx context.Context, username string) (data.Player, error) {
	if username == "notfound" {
		return data.Player{}, errorNotFound
	}
	if username == "unexpected" {
		return data.Player{}, errorUnexpected
	}

	return data.Player{
		Username: username,
	}, nil
}

func (m *Mysql) GetPlayersByUsernames(ctx context.Context, usernames []string) ([]data.Player, error) {
	players := []data.Player{}
	for _, username := range usernames {
		player, err := m.GetPlayerByUsername(ctx, username)
		if err != nil {
			if err != errorNotFound {
				return []data.Player{}, err
			}
		} else {
			players = append(players, player)
		}
	}
	return players, nil
}
