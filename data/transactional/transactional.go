package transactional

import (
	"github.com/satriahrh/letter-block/data"

	"context"
	"database/sql"
	"fmt"

	// use mysql driver
	_ "github.com/go-sql-driver/mysql"
)

// Transactional implementation of transactional with mysql
type Transactional struct {
	db *sql.DB
}

// NewTransactional transactional constructor
func NewTransactional(db *sql.DB) *Transactional {
	return &Transactional{
		db: db,
	}
}

// BeginTransaction begin the transaction with default options
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

// FinalizeTransaction commit or rollback transaction given error
func (t *Transactional) FinalizeTransaction(tx *sql.Tx, err error) error {
	if err != nil {
		if errRollback := tx.Rollback(); errRollback != nil {
			return errRollback
		}
		return err
	}
	return tx.Commit()
}

// InsertGame insert new game
func (t *Transactional) InsertGame(ctx context.Context, tx *sql.Tx, game data.Game) (data.Game, error) {
	result, err := tx.ExecContext(
		ctx,
		"INSERT INTO games (current_order, board_base, board_positioning, max_strength, state) VALUES (?, ?, ?, ?, ?)",
		game.CurrentOrder, game.BoardBase, game.BoardPositioning, game.MaxStrength, game.State,
	)
	if err != nil {
		return data.Game{}, err
	}

	gameIDInt64, _ := result.LastInsertId()
	game.ID = uint64(gameIDInt64)

	return game, nil
}

// InsertGamePlayerBulk insert game player bulk
func (t *Transactional) InsertGamePlayerBulk(ctx context.Context, tx *sql.Tx, game data.Game, players []data.Player) (data.Game, error) {
	gamePlayerArgs := ""
	gamePlayers := make([]interface{}, 3*len(players))

	for i, player := range players {
		gamePlayerArgs += "(?,?,?)"
		if i != len(game.Players)-1 {
			gamePlayerArgs += ","
		}
		gamePlayers[i*3] = game.ID
		gamePlayers[i*3+1] = player.ID
		gamePlayers[i*3+2] = i + 1
	}
	_, err := tx.ExecContext(
		ctx,
		fmt.Sprintf(
			"INSERT INTO game_player (game_ID, player_ID, ordering) VALUES %v",
			gamePlayerArgs,
		),
		gamePlayers...,
	)

	if err != nil {
		return data.Game{}, err
	}

	game.Players = players
	return game, nil
}

// GetPlayersByUsernames get players by usernames
func (t *Transactional) GetPlayersByUsernames(ctx context.Context, usernames []string) ([]data.Player, error) {
	rows, err := t.db.QueryContext(ctx, "SELECT ID, username FROM players WHERE usernames IN ?", stringsToSQLArray(usernames))
	if err != nil {
		return []data.Player{}, err
	}
	defer func() {
		_ = rows.Close()
	}()

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

// GetGameByID get game by id
func (t *Transactional) GetGameByID(ctx context.Context, tx *sql.Tx, gameID uint64) (game data.Game, err error) {
	row := tx.QueryRowContext(
		ctx, "SELECT current_order, board_base, board_positioning, max_strength FROM games WHERE ID = ?", gameID,
	)

	err = row.Scan(&game.CurrentOrder, &game.BoardBase, &game.BoardPositioning, &game.MaxStrength)
	if err != nil {
		return
	}

	game.ID = gameID
	return
}

// GetGamePlayerByID get game player by id
func (t *Transactional) GetGamePlayerByID(ctx context.Context, gamePlayerID uint64) (gamePlayer data.GamePlayer, err error) {
	row := t.db.QueryRowContext(ctx, "SELECT game_ID, player_ID, ordering FROM game_player WHERE ID = ?", gamePlayerID)

	err = row.Scan(&gamePlayer.GameID, &gamePlayer.PlayerID, &gamePlayer.Ordering)
	if err != nil {
		return
	}

	return
}

// GetGamePlayersByGameID get list of game player by game id
func (t *Transactional) GetGamePlayersByGameID(ctx context.Context, tx *sql.Tx, gameID uint64) (gamePlayers []data.GamePlayer, err error) {
	rows, err := tx.QueryContext(ctx, "SELECT player_ID, ordering FROM game_player WHERE game_ID = ?", gameID)
	if err != nil {
		return []data.GamePlayer{}, err
	}
	defer func() {
		_ = rows.Close()
	}()

	for rows.Next() {
		gamePlayer := data.GamePlayer{GameID: gameID}
		err = rows.Scan(&gamePlayer.PlayerID, &gamePlayer.Ordering)
		if err != nil {
			return
		}
		gamePlayers = append(gamePlayers, gamePlayer)
	}

	return
}

// LogPlayedWord to log word in a game
func (t *Transactional) LogPlayedWord(ctx context.Context, tx *sql.Tx, gameID, playerID uint64, word string) error {
	_, err := tx.ExecContext(
		ctx,
		"INSERT INTO played_word (game_ID, word, player_ID) VALUES (?, ?, ?)",
		gameID, word, playerID,
	)
	if err != nil {
		return err
	}

	return nil
}

// UpdateGame to update game
func (t *Transactional) UpdateGame(ctx context.Context, tx *sql.Tx, game data.Game) error {
	_, err := tx.ExecContext(ctx,
		"UPDATE game SET board_positioning = ?, current_order = ?, state  = ? WHERE ID = ?",
		game.BoardPositioning, game.CurrentOrder, game.State, game.ID,
	)
	return err
}

func stringsToSQLArray(slice []string) string {
	ret := ""
	for i := range slice {
		ret += fmt.Sprintf("'%v'", slice[i])
		if i < len(slice)-1 {
			ret += ","
		}
	}

	return fmt.Sprintf("(%v)", ret)
}
