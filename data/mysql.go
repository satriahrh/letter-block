package data

import (
	"context"
	"database/sql"
)

type Mysql struct {
	DB *sql.DB
}

type Logic interface {
	GetPlayerByUsername(context.Context, string) (Player, error)
	GetPlayersByUsernames(context.Context, []string) ([]Player, error)
}

func NewMysql() (*Mysql, error) {
	db, err := sql.Open("mysql", "root:rootpw@/letter_block_development")
	if err != nil {
		return &Mysql{}, err
	}

	return &Mysql{
		DB: db,
	}, nil
}

func (m *Mysql) GetPlayerByUsername(ctx context.Context, username string) (Player, error) {
	return Player{}, nil
}

func (m *Mysql) GetPlayersByUsernames(ctx context.Context, usernames []string) ([]Player, error) {
	return []Player{}, nil
}
