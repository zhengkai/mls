package mls

import (
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// IRoom ...
type IRoom interface {
	PlayerAdd(p IPlayer)
	PlayerMsg(p IPlayer, msg interface{})
	PlayerExit(p IPlayer)
	SendMsg(msg []byte, playerID ...interface{})
	GetTickDuration() (d time.Duration)
	Start(tick time.Duration)
	Stop()
	Init()
}

type roomCmdType uint8

// Room ...
type Room struct {
	World     IWorld
	recv      ifRecvChan
	player    map[interface{}]IPlayer
	isStop    bool
	idleCount int
	tick      *time.Ticker
	tickID    int
}

var (
	roomMap = sync.Map{}
)

const (
	_ roomCmdType = iota
	roomCmdMsg
	roomCmdNew
	roomCmdExit
	roomCmdBroadcast
)

type ifRecvChan chan *roomCmd

type roomCmd struct {
	t      roomCmdType
	Player IPlayer
	Msg    interface{}
}

// InitRoom 启动 room
func InitRoom(r IRoom) {

	r.Init()
	go func() {
		r.Start(r.GetTickDuration())
		r.Stop()
	}()
}

// GetRoom ...
func GetRoom(id interface{}, w IWorld, r IRoom) IRoom {

	load, ok := roomMap.Load(id)
	if ok {
		return load.(IRoom)
	}

	r.Init()

	roomMap.Store(id, r)

	go func() {
		r.Start(r.GetTickDuration())
		roomMap.Delete(id)
		r.Stop()
	}()

	return r
}

// Init ...
func (r *Room) Init() {
	r.recv = make(ifRecvChan, 1000)
	r.player = make(map[interface{}]IPlayer)
}

// PlayerAdd ...
func (r *Room) PlayerAdd(p IPlayer) {

	r.recv <- &roomCmd{
		t:      roomCmdNew,
		Player: p,
	}

	p.SetRoom(r)
}

// PlayerMsg ...
func (r *Room) PlayerMsg(p IPlayer, msg interface{}) {
	r.recv <- &roomCmd{
		t:      roomCmdMsg,
		Player: p,
		Msg:    msg,
	}
}

// PlayerExit ...
func (r *Room) PlayerExit(p IPlayer) {
	r.recv <- &roomCmd{
		t:      roomCmdExit,
		Player: p,
	}
}

// Start ...
func (r *Room) Start(tick time.Duration) {

	r.tick = time.NewTicker(tick)

	for {
		ok := r.LoopServe()
		if !ok || r.isStop {
			break
		}
	}
}

// GetTickDuration ...
func (r *Room) GetTickDuration() time.Duration {
	return time.Second
}

// LoopServe ...
func (r *Room) LoopServe() (ok bool) {

	var recv *roomCmd

	select {
	case <-r.tick.C:
		r.tickID++
		return r.World.Tick(r.tickID)
	case recv, ok = <-r.recv:
	}

	if !ok {
		return
	}

	switch recv.t {

	case roomCmdNew:
		r.cmdNew(recv)

	case roomCmdExit:
		r.cmdExit(recv)

	case roomCmdMsg:
		r.cmdMsg(recv)
	}

	return
}

// Stop ...
func (r *Room) Stop() {
	r.isStop = true
}

func (r *Room) cmdNew(c *roomCmd) {

	pid := c.Player.GetID()

	old, oldOK := r.player[pid]
	if oldOK {
		old.Close()
	}

	if r.isStop {
		delete(r.player, pid)
		return
	}

	r.player[pid] = c.Player

	c.Player.SendDeamon()

	if !oldOK {
		r.World.Player(c.Player, true)
	}
}

func (r *Room) cmdExit(c *roomCmd) {

	pid := c.Player.GetID()

	p, ok := r.player[pid]
	if !ok {
		return
	}

	if p != c.Player {
		return
	}

	delete(r.player, pid)
	p.Close()

	r.World.Player(c.Player, false)
}

func (r *Room) cmdMsg(c *roomCmd) {

	if r.isStop {
		return
	}

	r.World.Input(c.Player, c.Msg)
}

// SendMsg 给玩家发信息
func (r *Room) SendMsg(msg []byte, playerID ...interface{}) {

	pm, _ := websocket.NewPreparedMessage(websocket.BinaryMessage, msg)

	// 按指定 id 列表发

	if len(playerID) > 0 {
		for _, pid := range playerID {
			p, ok := r.player[pid]
			if ok {
				p.SendBroadcast(pm, false)
			}
		}
		return
	}

	// 所有人群发

	for _, p := range r.player {
		go p.SendBroadcast(pm, false)
	}
}
