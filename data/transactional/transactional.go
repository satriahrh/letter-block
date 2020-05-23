package transactional

import (
	"context"
	"database/sql"
	"log"

	_ "github.com/go-sql-driver/mysql"

	"github.com/satriahrh/letter-block/data"
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
		Isolation: sql.LevelRepeatableRead,
		ReadOnly:  false,
	})
	if err != nil {
		log.Println(err)
		return nil, err
	}
	return tx, nil
}

func (t *Transactional) FinalizeTransaction(tx *sql.Tx, err error) error {
	if err != nil {
		if errRollback := tx.Rollback(); errRollback != nil {
			log.Println(errRollback)
			return errRollback
		}
		return err
	}
	return tx.Commit()
}

func (t *Transactional) InsertGame(ctx context.Context, tx *sql.Tx, game data.Game) (data.Game, error) {
	result, err := tx.ExecContext(
		ctx,
		"INSERT INTO games (current_player_order, number_of_player, board_base, board_positioning, state) VALUES (?, ?, ?, ?, ?)",
		game.CurrentPlayerOrder, game.NumberOfPlayer, game.BoardBase, game.BoardPositioning, game.State,
	)
	if err != nil {
		log.Println(err)
		return data.Game{}, err
	}

	gameIdInt64, _ := result.LastInsertId()
	game.Id = data.GameId(gameIdInt64)

	return game, nil
}

func (t *Transactional) InsertGamePlayer(ctx context.Context, tx *sql.Tx, game data.Game, player data.Player) (data.Game, error) {
	_, err := tx.ExecContext(
		ctx,
		"INSERT INTO games_players (game_id, player_id) VALUES (?, ?)",
		game.Id, player.Id,
	)
	if err != nil {
		log.Println(err)
		return data.Game{}, err
	}

	game.Players = append(game.Players, player)
	return game, err
}

func (t *Transactional) GetPlayerById(ctx context.Context, playerId data.PlayerId) (player data.Player, err error) {
	row := t.db.QueryRowContext(
		ctx, "SELECT id, username FROM players WHERE id = ?", playerId,
	)

	err = row.Scan(&player.Id, &player.Username)
	if err != nil {
		log.Println(err)
		return
	}

	return
}

func (t *Transactional) GetPlayersByGameId(ctx context.Context, gameId data.GameId) (players []data.Player, err error) {
	rows, err := t.db.QueryContext(ctx,
		`SELECT id, username
		FROM players
			INNER JOIN (
				SELECT player_id FROM games_players WHERE game_id = ?
			) as game_players 
			ON game_players.player_id = players.id`,
		gameId,
	)
	if err != nil {
		log.Println(err)
		return
	}

	for rows.Next() {
		var player data.Player
		err = rows.Scan(&player.Id, &player.Username)
		if err != nil {
			return
		}
		players = append(players, player)
	}

	return
}

func (t *Transactional) GetGameById(ctx context.Context, tx *sql.Tx, gameId data.GameId) (game data.Game, err error) {
	query := "SELECT current_player_order, number_of_player, board_base, board_positioning, state FROM games WHERE id = ?"
	args := []interface{}{gameId}

	var row *sql.Row
	if tx != nil {
		row = tx.QueryRowContext(ctx, query, args...)
	} else {
		row = t.db.QueryRowContext(ctx, query, args...)
	}

	err = row.Scan(&game.CurrentPlayerOrder, &game.NumberOfPlayer, &game.BoardBase, &game.BoardPositioning, &game.State)
	if err != nil {
		return
	}

	game.Id = gameId
	return
}

func (t *Transactional) GetGamePlayersByGameId(ctx context.Context, tx *sql.Tx, gameId data.GameId) (gamePlayers []data.GamePlayer, err error) {
	rows, err := tx.QueryContext(ctx, "SELECT player_id FROM games_players WHERE game_id = ?", gameId)
	if err != nil {
		return []data.GamePlayer{}, err
	}
	defer func() {
		_ = rows.Close()
	}()

	for rows.Next() {
		gamePlayer := data.GamePlayer{GameId: gameId}
		err = rows.Scan(&gamePlayer.PlayerId)
		if err != nil {
			return
		}
		gamePlayers = append(gamePlayers, gamePlayer)
	}

	return
}

func (t *Transactional) GetGamesByPlayerId(ctx context.Context, playerId data.PlayerId) (games []data.Game, err error) {
	rows, err := t.db.QueryContext(ctx,
		`SELECT id, current_player_order, number_of_player, board_base, board_positioning, state
		FROM games
			INNER JOIN (
				SELECT game_id FROM games_players WHERE player_id = ?
			) as played_games 
			ON played_games.game_id = games.id`,
		playerId,
	)
	if err != nil {
		log.Println(err)
		return
	}

	for rows.Next() {
		var game data.Game
		err = rows.Scan(&game.Id, &game.CurrentPlayerOrder, &game.NumberOfPlayer, &game.BoardBase, &game.BoardPositioning, &game.State)
		if err != nil {
			return
		}
		games = append(games, game)
	}

	return
}

func (t *Transactional) LogPlayedWord(ctx context.Context, tx *sql.Tx, gameId data.GameId, playerId data.PlayerId, word string) error {
	_, err := tx.ExecContext(
		ctx,
		"INSERT INTO played_words (game_id, word, player_id) VALUES (?, ?, ?)",
		gameId, word, playerId,
	)
	if err != nil {
		return err
	}

	return nil
}

func (t *Transactional) GetPlayedWordsByGameId(ctx context.Context, gameId data.GameId) (playedWords []data.PlayedWord, err error) {
	rows, err := t.db.QueryContext(ctx,
		`SELECT word, player_id FROM played_words WHERE game_id = ?`,
		gameId,
	)
	if err != nil {
		log.Println(err)
		return
	}

	for rows.Next() {
		var playedWord data.PlayedWord
		err = rows.Scan(&playedWord.Word, &playedWord.PlayerId)
		if err != nil {
			log.Println(err)
			return
		}
		playedWords = append(playedWords, playedWord)
	}

	return
}

func (t *Transactional) UpdateGame(ctx context.Context, tx *sql.Tx, game data.Game) error {
	_, err := tx.ExecContext(ctx,
		"UPDATE games SET board_positioning = ?, board_base = ?, current_player_order = ?, state  = ? WHERE id = ?",
		game.BoardPositioning, game.BoardBase, game.CurrentPlayerOrder, game.State, game.Id,
	)
	return err
}

func (t *Transactional) UpdatePlayer(ctx context.Context, tx *sql.Tx, player data.Player) error {
	_, err := tx.ExecContext(ctx,
		"UPDATE players SET session_expired_at = ? WHERE id = ?",
		player.SessionExpiredAt, player.Id,
	)
	if err != nil {
		log.Println(err)
	}
	return err
}

func (t *Transactional) UpsertPlayer(ctx context.Context, tx *sql.Tx, player data.Player) (err error) {
	_, err = tx.ExecContext(ctx,
		`INSERT IGNORE INTO players (device_fingerprint, username) VALUES (?, ?) ON DUPLICATE KEY UPDATE username = ?`, player.DeviceFingerprint, player.Username, player.Username,
	)
	if err != nil {
		log.Println(err)
	}
	return
}

func (t *Transactional) GetPlayerByDeviceFingerprint(ctx context.Context, tx *sql.Tx, fingerprint data.DeviceFingerprint) (player data.Player, err error) {
	row := tx.QueryRowContext(
		ctx, "SELECT id, username, device_fingerprint, session_expired_at FROM players WHERE device_fingerprint = ?", fingerprint,
	)

	err = row.Scan(&player.Id, &player.Username, &player.DeviceFingerprint, &player.SessionExpiredAt)
	if err != nil {
		log.Println(err)
		return
	}

	return
}
