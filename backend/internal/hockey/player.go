package hockey

import (
	"log"

	"github.com/gorilla/websocket"
)

type Player struct {
	Name string          `json:"name"`
	Conn *websocket.Conn `json:"-"` // Don't serialize this field
}

func NewPlayer(name string, conn *websocket.Conn) *Player {
	return &Player{
		Name: name,
		Conn: conn,
	}
}

func (p *Player) Start() {
	for {
		_, msg, err := p.Conn.ReadMessage()
		if err != nil {
			log.Println(err)
			return
		}

		log.Printf("%s sent: %s", p.Conn.RemoteAddr(), string(msg))

		if err := p.Conn.WriteMessage(websocket.TextMessage, msg); err != nil {
			log.Println(err)
			return
		}
	}
}
