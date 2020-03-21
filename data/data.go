package data

type Data struct {
	Mysql Logic
}

func NewData(m Logic) (*Data, error) {
	return &Data{
		Mysql: m,
	}, nil
}

type Player struct {
	Username string `json:"username"`
}