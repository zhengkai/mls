package mls

// Manager ...
func Manager(playerChan chan IPlayer, getWorld func(id interface{}) IWorld) {

	for {
		p := <-playerChan

		w := getWorld(p.GetWorldID())

		go w.PlayerAdd(p)
	}
}
