package mls

// IWorld ...
type IWorld interface {
	Player(p IPlayer, status bool)
	Input(p IPlayer, msg interface{})
	PlayerAdd(p IPlayer)
	Tick(i int) (ok bool)
}

// World ...
type World struct {
	Room IRoom
}

// GetWorld 仅做演示用途
func GetWorld(id interface{}) IWorld {

	nw := &World{}
	r := &Room{}
	r.World = nw

	nw.Room = r
	InitRoom(r)

	return nw
}

// PlayerAdd ...
func (w *World) PlayerAdd(p IPlayer) {
	w.Room.PlayerAdd(p)
}

// Player 状态更新
func (w *World) Player(p IPlayer, status bool) {
}

// Input 玩家输入
func (w *World) Input(p IPlayer, msg interface{}) {
}

// Tick ...
func (w *World) Tick(i int) (ok bool) {
	return true
}
