package rome

// IWorld ...
type IWorld interface {
	Player(p IPlayerConn, status bool)
	Input(p IPlayerConn, msg interface{})
	PlayerConn(p IPlayerConn)
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

// PlayerConn ...
func (w *World) PlayerConn(p IPlayerConn) {

	w.Room.PlayerConn(p)
}

// Player 状态更新
func (w *World) Player(p IPlayerConn, status bool) {
}

// Input 玩家输入
func (w *World) Input(p IPlayerConn, msg interface{}) {
}

// Tick ...
func (w *World) Tick(i int) (ok bool) {
	return true
}
