package data

import "fmt"

type Data struct {
	Mysql LogicOfMysql
}

func NewData(m LogicOfMysql) (*Data, error) {
	return &Data{
		Mysql: m,
	}, nil
}

type Player struct {
	ID       uint64 `json:"id"`
	Username string `json:"username"`
}

type Game struct {
	ID               uint64   `json:"id"`
	CurrentTurn      uint8    `json:"current_turn"`
	Players          []Player `json:"players"`
	MaxStrength      uint8    `json:"max_strength"`
	BoardBase        []uint8  `json:"board_base"`
	BoardPositioning []uint8  `json:"board_positioning"`
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
