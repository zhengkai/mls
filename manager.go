package rome

// Manager ...
func Manager(playerChan chan IPlayerConn, getWorld func(id interface{}) IWorld) {

	for {
		p := <-playerChan

		w := getWorld(p.GetWorldID())

		go w.PlayerConn(p)
	}
}
