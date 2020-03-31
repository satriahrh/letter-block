package transactional

import (
	"github.com/satriahrh/letter-block/data"

	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"

	_ "github.com/go-sql-driver/mysql"
)

type Transactional struct {
	db *sql.DB
}

func NewTransactional(db *sql.DB) *Transactional {
	return &Transactional{
		db: db,
	}
}

func (t *Transactional) BeginTransaction(ctx context.Context) (*sql.Tx, error) {
	tx, err := t.db.BeginTx(ctx, &sql.TxOptions{
		Isolation: sql.LevelWriteCommitted,
		ReadOnly:  false,
	})
	if err != nil {
		return nil, err
	}
	return tx, nil
}

func (t *Transactional) FinalizeTransaction(tx driver.Tx, err error) error {
	if err != nil {
		if errRollback := tx.Rollback(); errRollback != nil {
			return errRollback
		}
		return err
	}
	return tx.Commit()
}

func (t *Transactional) Transaction(ctx context.Context, options *sql.TxOptions, transaction func(*sql.Tx) error) error {
	tx, err := t.db.BeginTx(ctx, &sql.TxOptions{
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

func (t *Transactional) InsertGame(ctx context.Context, game data.Game) (data.Game, error) {
	gamePlayerArgs := ""
	gamePlayers := make([]interface{}, 2*len(game.Players))

	options := &sql.TxOptions{
		Isolation: sql.LevelWriteCommitted,
		ReadOnly:  false,
	}
	err := t.Transaction(ctx, options, func(tx *sql.Tx) error {
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
		return data.Game{}, err
	}

	return game, nil
}

func (t *Transactional) GetPlayersByUsernames(ctx context.Context, usernames []string) ([]data.Player, error) {
	rows, err := t.db.QueryContext(ctx, "SELECT * FROM players WHERE usernames IN ?", stringsToSqlArray(usernames))
	if err != nil {
		return []data.Player{}, err
	}
	defer rows.Close()

	players := make([]data.Player, 0)
	for rows.Next() {
		player := data.Player{}
		err := rows.Scan(&player.ID, &player.Username)
		if err != nil {
			return []data.Player{}, err
		}
		players = append(players, player)
	}

	return players, nil
}

func (t *Transactional) GetGameByID(ctx context.Context, tx *sql.Tx, gameID uint64) (data.Game, error) {
	var game data.Game
	row := tx.QueryRowContext(
		ctx,
		"SELECT current_player_id, board_base FROM games WHERE id = ?",
		gameID,
	)
	err := row.Scan(&game.CurrentPlayerID, &game.BoardBase)
	if err != nil {
		return data.Game{}, err
	}
	game.ID = gameID
	return game, nil
}

func (t *Transactional) GetGamePlayerByID(ctx context.Context, gamePlayerID uint64) (uint64, uint64, error) {
	row := t.db.QueryRowContext(ctx, "SELECT game_id, player_id FROM game_player WHERE id = ?", gamePlayerID)

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
