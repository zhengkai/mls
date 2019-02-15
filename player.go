package mls

import (
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

var (
	playerAutoID uint64
)

// IPlayer 用户
type IPlayer interface {
	LoginInput() (b []byte, err error)
	LoginDecode(b []byte) (login interface{}, ok bool)
	LoginAuth(login interface{}) (ok bool)
	SendManager(login interface{}, pchan chan IPlayer, np IPlayer) (ok bool)
	SetRoom(r IRoom)
	GetWorldID() interface{}
	GetID() interface{}
	SendDeamon()
	ReadDeamon(cp IPlayer)
	ParseMessage(b []byte) (msg interface{}, ok bool)
	Send(b []byte)
	SendStop(b []byte)
	SendBroadcast(pm *websocket.PreparedMessage, stop bool)
	Close()
}

// Player ...
type Player struct {
	ID      interface{}
	WS      *websocket.Conn
	room    IRoom
	confirm chan bool
	isClose bool
	send    chan *PlayerMsg

	CloseCode int
	CloseText string
}

// PlayerMsg ...
type PlayerMsg struct {
	Content interface{}
	Stop    bool
}

// LoginInput ...
func (p *Player) LoginInput() (b []byte, err error) {
	p.WS.SetReadDeadline(time.Now().Add(21 * time.Second))
	_, b, err = p.WS.ReadMessage()
	return
}

// LoginDecode ...
func (p *Player) LoginDecode(b []byte) (login interface{}, ok bool) {

	// p.CloseCode = 1
	// p.CloseText = `login parse error`

	return nil, true
}

// LoginAuth ...
func (p *Player) LoginAuth(login interface{}) (ok bool) {

	// p.CloseCode = 2
	// p.CloseText = `auth fail`

	return true
}

// SendManager ...
func (p *Player) SendManager(login interface{}, pchan chan IPlayer, np IPlayer) (ok bool) {

	p.confirm = make(chan bool)

	pchan <- np

	ok = <-p.confirm

	close(p.confirm)

	return
}

// SetRoom ...
func (p *Player) SetRoom(r IRoom) {
	ok := r != nil
	p.room = r
	p.confirm <- ok
}

// Close ...
func (p *Player) Close() {

	if p.isClose {
		return
	}
	p.isClose = true

	if p.CloseCode > 0 {
		p.WS.SetWriteDeadline(time.Now().Add(30 * time.Second))
		ab := websocket.FormatCloseMessage(p.CloseCode+4000, p.CloseText)
		p.WS.WriteMessage(websocket.CloseMessage, ab)
	}
	p.WS.Close()

	if p.send != nil {
		close(p.send)
		p.send = nil
	}
}

// ParsePlayer 处理通过 ws 连接过来的用户
func ParsePlayer(p IPlayer, pchan chan IPlayer) {
	b, err := p.LoginInput()
	if err != nil {
		return
	}

	closeReason := parsePlayer(p, b, pchan)

	if closeReason {
		p.Close()
	}
}

func parsePlayer(p IPlayer, b []byte, pchan chan IPlayer) (closeReason bool) {

	closeReason = true

	login, ok := p.LoginDecode(b)
	if !ok {
		return
	}

	ok = p.LoginAuth(login)
	if !ok {
		return
	}

	closeReason = false

	ok = p.SendManager(login, pchan, p)
	if !ok {
		return
	}

	p.ReadDeamon(p)

	return
}

// Send 给客户端发消息
func (p *Player) Send(b []byte) {
	p.send <- &PlayerMsg{
		Content: b,
	}
}

// SendStop 给客户端发消息，并关闭连接
func (p *Player) SendStop(b []byte) {
	p.send <- &PlayerMsg{
		Content: b,
		Stop:    true,
	}
}

// SendBroadcast 群发消息专用，
func (p *Player) SendBroadcast(pm *websocket.PreparedMessage, stop bool) {
	p.send <- &PlayerMsg{
		Content: pm,
		Stop:    stop,
	}
}

// ReadDeamon ...
func (p *Player) ReadDeamon(cp IPlayer) {

	var err error
	var ab []byte

	p.WS.SetReadDeadline(time.Time{})
	for {

		_, ab, err = p.WS.ReadMessage()
		if err != nil {
			break
		}

		msg, ok := cp.ParseMessage(ab)
		if !ok {
			break
		}

		p.room.PlayerMsg(cp, msg)
	}

	p.room.PlayerExit(cp)

	p.CloseCode = 0
	p.WS.Close()
}

// ParseMessage ...
func (p *Player) ParseMessage(b []byte) (msg interface{}, ok bool) {
	return string(b), true
}

// GetWorldID ...
func (p *Player) GetWorldID() interface{} {
	return 1
}

// GetID ...
func (p *Player) GetID() interface{} {

	if p.ID != nil {
		return p.ID
	}

	p.ID = atomic.AddUint64(&playerAutoID, 1)

	return p.ID
}

// SendDeamon ...
func (p *Player) SendDeamon() {

	p.send = make(chan *PlayerMsg, 1000)

	go p.sendDeamon()
}

func (p *Player) sendDeamon() {

	for {
		send, ok := <-p.send
		if !ok {
			break
		}

		var err error
		p.WS.SetWriteDeadline(time.Now().Add(30 * time.Second))

		switch t := send.Content.(type) {

		case []byte:
			err = p.WS.WriteMessage(websocket.BinaryMessage, t)

		case *websocket.PreparedMessage:
			err = p.WS.WritePreparedMessage(t)
		}

		if err != nil || send.Stop {
			break
		}
	}

	p.WS.Close()
}
