package objs

// data about rooms
type Room struct {
	Name string
	routes []*Room
}

// allocate new room
func NewRoom(name string) *Room {
	return &Room{Name: name}
}

// global types of rooms
var RoomsNames = []string {
	"кухня",
	"коридор",
	"комната",
	"улица",
}

// create rooms map
func createRooms() map[string]*Room {
	rooms := make(map[string]*Room)
	for _, name := range RoomsNames {
		rooms[name] = NewRoom(name)
	}
	return rooms
}

// connect Room A and Room B
func (from *Room) connect(to *Room) {
	from.routes = append(from.routes, to)
}
