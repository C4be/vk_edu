package objs

// main game object
type GameObj struct {
	// all rooms
	Rooms map[string]*Room
}

func InitGameObj() *GameObj {
	// create the game
	var mainGame GameObj  
	
	// create rooms
	mainGame.Rooms = createRooms() 
	// and add connects
	for _, name := range []string{"кухня", "комната", "улица"} {
		mainGame.Rooms[name].connect(mainGame.Rooms["коридор"])
		mainGame.Rooms["коридор"].connect(mainGame.Rooms[name])
	}

	return &mainGame
}
