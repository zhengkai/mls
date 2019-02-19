package rome

import (
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

var (
	playerAutoID uint64
)

// IPlayerConn 用户
type IPlayerConn interface {
	LoginInput() (b []byte, err error)
	Login(b []byte) (login interface{}, ok bool)
	SendManager(login interface{}, pchan chan IPlayerConn, np IPlayerConn) (ok bool)
	SetRoom(r IRoom)
	GetWorldID() interface{}
	GetID() interface{}
	SendDeamon()
	ReadDeamon(cp IPlayerConn)
	ParseMessage(b []byte) (msg interface{}, ok bool)
	Send(b []byte)
	SendStop(b []byte)
	SendBroadcast(pm *websocket.PreparedMessage, stop bool)
	Close()
}

// PlayerConn ...
type PlayerConn struct {
	ID      interface{}
	WS      *websocket.Conn
	room    IRoom
	confirm chan bool
	isClose bool
	send    chan *PlayerConnMsg

	CloseCode int
	CloseText string
}

// PlayerConnMsg ...
type PlayerConnMsg struct {
	Content interface{}
	Stop    bool
}

// LoginInput ...
func (p *PlayerConn) LoginInput() (b []byte, err error) {
	p.WS.SetReadDeadline(time.Now().Add(21 * time.Second))
	_, b, err = p.WS.ReadMessage()
	return
}

// LoginDecode ...
func (p *PlayerConn) Login(b []byte) (login interface{}, ok bool) {

	// p.CloseCode = 1
	// p.CloseText = `login parse error`

	return nil, true
}

// SendManager ...
func (p *PlayerConn) SendManager(login interface{}, pchan chan IPlayerConn, np IPlayerConn) (ok bool) {

	p.confirm = make(chan bool)

	pchan <- np

	ok = <-p.confirm

	close(p.confirm)

	return
}

// SetRoom ...
func (p *PlayerConn) SetRoom(r IRoom) {
	ok := r != nil
	p.room = r
	p.confirm <- ok
}

// Close ...
func (p *PlayerConn) Close() {

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

// ParsePlayerConn 处理通过 ws 连接过来的用户
func ParsePlayerConn(p IPlayerConn, pchan chan IPlayerConn) {
	b, err := p.LoginInput()
	if err != nil {
		return
	}

	closeReason := parsePlayerConn(p, b, pchan)

	if closeReason {
		p.Close()
	}
}

func parsePlayerConn(p IPlayerConn, b []byte, pchan chan IPlayerConn) (closeReason bool) {

	closeReason = true

	login, ok := p.Login(b)
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
func (p *PlayerConn) Send(b []byte) {
	p.send <- &PlayerConnMsg{
		Content: b,
	}
}

// SendStop 给客户端发消息，并关闭连接
func (p *PlayerConn) SendStop(b []byte) {
	p.send <- &PlayerConnMsg{
		Content: b,
		Stop:    true,
	}
}

// SendBroadcast 群发消息专用，
func (p *PlayerConn) SendBroadcast(pm *websocket.PreparedMessage, stop bool) {
	p.send <- &PlayerConnMsg{
		Content: pm,
		Stop:    stop,
	}
}

// ReadDeamon ...
func (p *PlayerConn) ReadDeamon(cp IPlayerConn) {

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
func (p *PlayerConn) ParseMessage(b []byte) (msg interface{}, ok bool) {
	return string(b), true
}

// GetWorldID ...
func (p *PlayerConn) GetWorldID() interface{} {
	return 1
}

// GetID ...
func (p *PlayerConn) GetID() interface{} {

	if p.ID != nil {
		return p.ID
	}

	p.ID = atomic.AddUint64(&playerAutoID, 1)

	return p.ID
}

// SendDeamon ...
func (p *PlayerConn) SendDeamon() {

	p.send = make(chan *PlayerConnMsg, 1000)

	go p.sendDeamon()
}

func (p *PlayerConn) sendDeamon() {

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
