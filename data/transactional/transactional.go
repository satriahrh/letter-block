package transactional

import (
	"github.com/satriahrh/letter-block/data"

	"context"
	"database/sql"
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

func (t *Transactional) FinalizeTransaction(tx *sql.Tx, err error) error {
	if err != nil {
		if errRollback := tx.Rollback(); errRollback != nil {
			return errRollback
		}
		return err
	}
	return tx.Commit()
}

func (t *Transactional) InsertGame(ctx context.Context, tx *sql.Tx, game data.Game) (data.Game, error) {
	result, err := tx.ExecContext(
		ctx,
		"INSERT INTO games (current_player_order, board_base, board_positioning, state) VALUES (?, ?, ?, ?, ?)",
		game.CurrentPlayerOrder, game.BoardBase, game.BoardPositioning, game.State,
	)
	if err != nil {
		return data.Game{}, err
	}

	gameIdInt64, _ := result.LastInsertId()
	game.Id = data.GameId(gameIdInt64)

	return game, nil
}

func (t *Transactional) InsertGamePlayer(ctx context.Context, tx *sql.Tx, game data.Game, player data.Player) (data.Game, error) {
	_, err := tx.ExecContext(
		ctx,
		"INSERT INTO game_player (game_id, player_id) VALUES (?, ?)",
		game.Id, player.Id,
	)
	if err == nil {
		game.Players = append(game.Players, player)
	}
	return game, err
}

func (t *Transactional) GetPlayerById(ctx context.Context, playerId data.PlayerId) (player data.Player, err error) {
	row := t.db.QueryRowContext(
		ctx, "SELECT username FROM players WHERE id = ?", playerId,
	)

	err = row.Scan(&player.Username)
	if err != nil {
		return
	}

	player.Id = playerId
	return
}

func (t *Transactional) GetGameById(ctx context.Context, tx *sql.Tx, gameId data.GameId) (game data.Game, err error) {
	row := tx.QueryRowContext(
		ctx, "SELECT current_order, board_base, board_positioning FROM games WHERE id = ?", gameId,
	)

	err = row.Scan(&game.CurrentPlayerOrder, &game.BoardBase, &game.BoardPositioning)
	if err != nil {
		return
	}

	game.Id = gameId
	return
}

func (t *Transactional) GetGamePlayerById(ctx context.Context, gamePlayerId data.GamePlayerId) (gamePlayer data.GamePlayer, err error) {
	row := t.db.QueryRowContext(ctx, "SELECT game_id, player_id, ordering FROM game_player WHERE id = ?", gamePlayerId)

	err = row.Scan(&gamePlayer.GameId, &gamePlayer.PlayerId, &gamePlayer.Ordering)
	if err != nil {
		return
	}

	return
}

func (t *Transactional) GetGamePlayersByGameId(ctx context.Context, tx *sql.Tx, gameId data.GameId) (gamePlayers []data.GamePlayer, err error) {
	rows, err := tx.QueryContext(ctx, "SELECT player_id, ordering FROM game_player WHERE game_id = ?", gameId)
	if err != nil {
		return []data.GamePlayer{}, err
	}
	defer func() {
		_ = rows.Close()
	}()

	for rows.Next() {
		gamePlayer := data.GamePlayer{GameId: gameId}
		err = rows.Scan(&gamePlayer.PlayerId, &gamePlayer.Ordering)
		if err != nil {
			return
		}
		gamePlayers = append(gamePlayers, gamePlayer)
	}

	return
}

func (t *Transactional) LogPlayedWord(ctx context.Context, tx *sql.Tx, gameId data.GameId, playerId data.PlayerId, word string) error {
	_, err := tx.ExecContext(
		ctx,
		"INSERT INTO played_word (game_id, word, player_id) VALUES (?, ?, ?)",
		gameId, word, playerId,
	)
	if err != nil {
		return err
	}

	return nil
}

func (t *Transactional) UpdateGame(ctx context.Context, tx *sql.Tx, game data.Game) error {
	_, err := tx.ExecContext(ctx,
		"UPDATE game SET board_positioning = ?, current_order = ?, state  = ? WHERE id = ?",
		game.BoardPositioning, game.CurrentPlayerOrder, game.State, game.Id,
	)
	return err
}
