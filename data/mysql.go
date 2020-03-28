package data

import (
	"context"
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
)

type Mysql struct {
	DB *sql.DB
}

type LogicOfMysql interface {
	InsertGame(context.Context, Game) (Game, error)
	GetPlayerByUsername(context.Context, string) (Player, error)
	GetPlayersByUsernames(context.Context, []string) ([]Player, error)
	GetGamePlayerByID(context.Context, uint64) (uint64, uint64, error)
}

func (m *Mysql) InsertGame(ctx context.Context, game Game) (Game, error) {
	gamePlayerArgs := ""
	gamePlayers := make([]interface{}, 2*len(game.Players))

	conn, err := m.DB.Conn(ctx)
	if err != nil {
		return Game{}, err
	}
	defer conn.Close()

	tx, err := conn.BeginTx(ctx, &sql.TxOptions{
		Isolation: sql.LevelWriteCommitted,
		ReadOnly:  false,
	})
	if err != nil {
		return Game{}, err
	}

	result, err := tx.ExecContext(
		ctx,
		"INSERT INTO games (current_turn, board_base, board_positioning, max_strength) VALUES (?, ?, ?, ?)",
		game.CurrentTurn, game.BoardBase, game.BoardPositioning, game.MaxStrength,
	)
	if err != nil {
		if errRollback := tx.Rollback(); errRollback != nil {
			return Game{}, errRollback
		}
		return Game{}, err
	}

	gameIDInt64, _ := result.LastInsertId()
	game.ID = uint64(gameIDInt64)
	for i, player := range game.Players {
		gamePlayerArgs += "(?,?)"
		if i < len(game.Players)-1 {
			gamePlayerArgs += ","
		}
		gamePlayers[i*2] = game.ID
		gamePlayers[i*2+1] = player.ID
	}
	_, err = tx.ExecContext(
		ctx,
		fmt.Sprintf(
			"INSERT INTO game_player (game_id, player_id) VALUES %v",
			gamePlayerArgs,
		),
		gamePlayers...,
	)
	if err != nil {
		if errRollback := tx.Rollback(); errRollback != nil {
			return Game{}, errRollback
		}
		return Game{}, err
	}

	err = tx.Commit()
	if err != nil {
		return Game{}, err
	}
	return game, nil
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

func (m *Mysql) GetGamePlayerByID(ctx context.Context, gamePlayerID uint64) (uint64, uint64, error) {
	row := m.DB.QueryRowContext(ctx, "SELECT game_id, player_id FROM game_player WHERE id = ?", gamePlayerID)

	var gameID, playerID uint64
	err := row.Scan(&gameID, &playerID)
	if err != nil && err == sql.ErrNoRows {
		return 0, 0, nil
	}

	return gameID, playerID, err
}

func stringsToSqlArray(slice []string) string {
	ret := ""
	for i := range slice {
		ret += fmt.Sprintf("'%v'", slice[i])
		if i < len(slice)-1 {
			ret += ","
		}
	}

	return fmt.Sprintf("(%v)", ret)
}
