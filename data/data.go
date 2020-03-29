package data

type Dictionary interface {
	generateKey(lang, key string) string
	Get(lang, key string) (resut bool, exist bool)
	Set(lang, key string, value bool)
}

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
	CurrentPlayerID  uint64   `json:"current_player_id"`
	Players          []Player `json:"players"`
	MaxStrength      uint8    `json:"max_strength"`
	BoardBase        []uint8  `json:"board_base"`
	BoardPositioning []uint8  `json:"board_positioning"`
}
