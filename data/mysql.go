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
	BeginTransaction(context.Context, *sql.TxOptions) (*sql.Tx, error)
	FinalizeTransaction(*sql.Tx, error) error
	Transaction(context.Context, *sql.TxOptions, func(*sql.Tx) error) error
	InsertGame(context.Context, Game) (Game, error)
	GetPlayerByUsername(context.Context, string) (Player, error)
	GetPlayersByUsernames(context.Context, []string) ([]Player, error)
	GetGameByID(context.Context, *sql.Tx, uint64) (Game, error)
	GetGamePlayerByID(context.Context, uint64) (uint64, uint64, error)
}

func (m *Mysql) BeginTransaction(ctx context.Context, options *sql.TxOptions) (*sql.Tx, error) {
	tx, err := m.DB.BeginTx(ctx, &sql.TxOptions{
		Isolation: sql.LevelWriteCommitted,
		ReadOnly:  false,
	})
	if err != nil {
		return nil, err
	}
	return tx, nil
}

func (m *Mysql) FinalizeTransaction(tx *sql.Tx, err error) error {
	if err != nil {
		if errRollback := tx.Rollback(); errRollback != nil {
			return errRollback
		}
		return err
	}
	return tx.Commit()
}

func (m *Mysql) Transaction(ctx context.Context, options *sql.TxOptions, transaction func(*sql.Tx) error) error {
	tx, err := m.DB.BeginTx(ctx, &sql.TxOptions{
		Isolation: sql.LevelWriteCommitted,
		ReadOnly:  false,
	})
	if err != nil {
		return err
	}

	err = transaction(tx)
	if err != nil {
		if errRollback := tx.Rollback(); errRollback != nil {
			return errRollback
		}
		return err
	}

	return tx.Commit()
}

func (m *Mysql) InsertGame(ctx context.Context, game Game) (Game, error) {
	gamePlayerArgs := ""
	gamePlayers := make([]interface{}, 2*len(game.Players))

	options := &sql.TxOptions{
		Isolation: sql.LevelWriteCommitted,
		ReadOnly:  false,
	}
	err := m.Transaction(ctx, options, func(tx *sql.Tx) error {
		result, err := tx.ExecContext(
			ctx,
			"INSERT INTO games (current_turn, board_base, board_positioning, max_strength) VALUES (?, ?, ?, ?)",
			game.CurrentPlayerID, game.BoardBase, game.BoardPositioning, game.MaxStrength,
		)
		if err != nil {
			return err
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
			return err
		}
		return nil
	})

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

func (m *Mysql) GetGameByID(ctx context.Context, tx *sql.Tx, gameID uint64) (Game, error) {
	var game Game
	row := tx.QueryRowContext(
		ctx,
		"SELECT current_player_id, board_base FROM games WHERE id = ?",
		gameID,
	)
	err := row.Scan(&game.CurrentPlayerID, &game.BoardBase)
	if err != nil {
		return Game{}, err
	}
	game.ID = gameID
	return game, nil
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
