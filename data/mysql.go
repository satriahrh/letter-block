package data

import (
	"context"
	"database/sql"
)

type Mysql struct {
	DB *sql.DB
}

type LogicOfMysql interface {
	GetPlayerByUsername(context.Context, string) (Player, error)
	GetPlayersByUsernames(context.Context, []string) ([]Player, error)
}

func (m *Mysql) GetPlayerByUsername(ctx context.Context, username string) (Player, error) {
	return Player{}, nil
}

func (m *Mysql) GetPlayersByUsernames(ctx context.Context, usernames []string) ([]Player, error) {
	rows, err := m.DB.QueryContext(ctx, "SELECT * FROM players WHERE usernames IN ?", stringsToSqlArray(usernames))
	if err != nil {
		return []Player{}, err
	}
	defer rows.Close()

	players := make([]Player, 0)
	for rows.Next() {
		player := Player{}
		err := rows.Scan(&player.ID, &player.Username)
		if err != nil {
			return []Player{}, err
		}
		players = append(players, player)
	}

	return players, nil
}
