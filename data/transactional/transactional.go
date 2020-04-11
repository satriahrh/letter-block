package transactional

import (
	"github.com/satriahrh/letter-block/data"

	"context"
	"database/sql"
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
		"INSERT INTO games (current_order, board_base, board_positioning, max_strength, state) VALUES (?, ?, ?, ?, ?)",
		game.CurrentOrder, game.BoardBase, game.BoardPositioning, game.MaxStrength, game.State,
	)
	if err != nil {
		return data.Game{}, err
	}

	gameIdInt64, _ := result.LastInsertId()
	game.Id = uint64(gameIdInt64)

	return game, nil
}

func (t *Transactional) InsertGamePlayerBulk(ctx context.Context, tx *sql.Tx, game data.Game, players []data.Player) (data.Game, error) {
	gamePlayerArgs := ""
	gamePlayers := make([]interface{}, 3*len(players))

	for i, player := range players {
		gamePlayerArgs += "(?,?,?)"
		if i != len(game.Players)-1 {
			gamePlayerArgs += ","
		}
		gamePlayers[i*3] = game.Id
		gamePlayers[i*3+1] = player.Id
		gamePlayers[i*3+2] = i + 1
	}
	_, err := tx.ExecContext(
		ctx,
		fmt.Sprintf(
			"INSERT INTO game_player (game_id, player_id, ordering) VALUES %v",
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

func (t *Transactional) InsertGamePlayer(ctx context.Context, tx *sql.Tx, game data.Game, player data.Player) (data.Game, error) {
	_, err := tx.ExecContext(
		ctx,
		"INSERT INTO game_player (game_id, player_id, ordering) VALUES (?, ?, ?)",
		game.Id, player.Id, len(game.Players)+1,
	)
	if err == nil {
		game.Players = append(game.Players, player)
	}
	return game, err
}

func (t *Transactional) GetPlayerById(ctx context.Context, playerId uint64) (data.Player, error) {
	return data.Player{}, nil
}

func (t *Transactional) GetPlayersByUsernames(ctx context.Context, usernames []string) ([]data.Player, error) {
	rows, err := t.db.QueryContext(ctx, "SELECT id, username FROM players WHERE usernames IN ?", stringsToSqlArray(usernames))
	if err != nil {
		return []data.Player{}, err
	}
	defer func() {
		_ = rows.Close()
	}()

	players := make([]data.Player, 0)
	for rows.Next() {
		player := data.Player{}
		err := rows.Scan(&player.Id, &player.Username)
		if err != nil {
			return []data.Player{}, err
		}
		players = append(players, player)
	}

	return players, nil
}

func (t *Transactional) GetGameById(ctx context.Context, tx *sql.Tx, gameId uint64) (game data.Game, err error) {
	row := tx.QueryRowContext(
		ctx, "SELECT current_order, board_base, board_positioning, max_strength FROM games WHERE id = ?", gameId,
	)

	err = row.Scan(&game.CurrentOrder, &game.BoardBase, &game.BoardPositioning, &game.MaxStrength)
	if err != nil {
		return
	}

	game.Id = gameId
	return
}

func (t *Transactional) GetGamePlayerById(ctx context.Context, gamePlayerId uint64) (gamePlayer data.GamePlayer, err error) {
	row := t.db.QueryRowContext(ctx, "SELECT game_id, player_id, ordering FROM game_player WHERE id = ?", gamePlayerId)

	err = row.Scan(&gamePlayer.GameId, &gamePlayer.PlayerId, &gamePlayer.Ordering)
	if err != nil {
		return
	}

	return
}

func (t *Transactional) GetGamePlayersByGameId(ctx context.Context, tx *sql.Tx, gameId uint64) (gamePlayers []data.GamePlayer, err error) {
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

func (t *Transactional) LogPlayedWord(ctx context.Context, tx *sql.Tx, gameId, playerId uint64, word string) error {
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
		game.BoardPositioning, game.CurrentOrder, game.State, game.Id,
	)
	return err
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
