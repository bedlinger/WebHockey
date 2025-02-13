package hockey

type Game struct {
	GameId  string  `json:"gameId"`
	Player1 *Player `json:"player1"`
	Player2 *Player `json:"player2"`
}

func NewGame(id string) *Game {
	return &Game{GameId: id}
}

func (g *Game) AddPlayer(p *Player) {
	if g.Player1 == nil {
		g.Player1 = p
	} else if g.Player2 == nil {
		g.Player2 = p
	} else {
		panic("Game is full")
	}
}

func (g *Game) Start() {
	go g.Player1.Start()
	go g.Player2.Start()
}
