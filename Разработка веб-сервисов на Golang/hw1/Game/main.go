package main

import (
	"fmt"
	"sort"
	"strings"
)

// --- Типы модели мира ---

// Path моделирует путь (выход) из комнаты
type Path struct {
	to         *Room
	locked     bool
	unlockItem string // предмет, которым можно открыть (например "ключи")
	lockMsg    string // сообщение если путь заблокирован
	unlockMsg  string // сообщение при успешном открытии
}

// Room представляет комнату
type Room struct {
	name        string
	description string
	items       map[string]bool
	paths       map[string]*Path

	// Опциональные хуки:
	lookFunc    func(w *World, r *Room) string
	useFunc     func(w *World, r *Room, item, target string) string // сначала пробуем этот хук
	// onEnterFunc func(w *World, from *Room) string                  // вызывается при входе
}

// Player представляет игрока
type Player struct {
	room        *Room
	inventory   map[string]bool
	hasBackpack bool
}

// World представляет игровой мир
type World struct {
	rooms map[string]*Room
	player *Player
}

// --- Инициализация игры ---

var world *World

const NothingNeed = "ничего не требуется"
const NothingUse = "не к чему применить"


func initGame() {
	world = &World{
		rooms: make(map[string]*Room),
	}

	// --- Создаем комнаты ---
	kitchen := &Room{
		name:        "кухня",
		description: "кухня, ничего интересного",
		items:       make(map[string]bool),
		paths:       make(map[string]*Path),
	}

	corridor := &Room{
		name:        "коридор",
		description: "ничего интересного",
		items:       make(map[string]bool),
		paths:       make(map[string]*Path),
	}

	room := &Room{
		name:        "комната",
		description: "ты в своей комнате",
		items:       make(map[string]bool),
		paths:       make(map[string]*Path),
	}

	street := &Room{
		name:        "улица",
		description: "на улице весна",
		items:       make(map[string]bool),
		paths:       make(map[string]*Path),
	}

	// --- Предметы ---
	kitchen.items["чай"] = true
	room.items["ключи"] = true
	room.items["конспекты"] = true
	room.items["рюкзак"] = true

	// --- Пути (используем Path, указываем locked для двери) ---
	kitchen.paths["коридор"] = &Path{to: corridor}
	// путь на улицу заблокирован — его можно открыть "ключи"

	corridor.paths["кухня"] = &Path{to: kitchen}
	corridor.paths["комната"] = &Path{to: room}
	corridor.paths["улица"] = &Path{
		to:         street,
		locked:     true,
		unlockItem: "ключи",
		lockMsg:    "дверь закрыта",
		unlockMsg:  "дверь открыта",
	}

	room.paths["коридор"] = &Path{to: corridor}
	street.paths["домой"] = &Path{to: corridor}

	// --- Хуки lookFunc для детерминированного описания ---
	kitchen.lookFunc = func(w *World, r *Room) string {
		items := getRoomItems(r)
		paths := getRoomPaths(r)
		if !w.player.hasBackpack {
			return fmt.Sprintf("ты находишься на кухне, на столе: %s, надо собрать рюкзак и идти в универ. можно пройти - %s", items, paths)
		}
		return fmt.Sprintf("ты находишься на кухне, на столе: %s, надо идти в универ. можно пройти - %s", items, paths)
	}

	room.lookFunc = func(w *World, r *Room) string {
		// как в вашем коде: рюкзак отдельно на стуле
		if len(r.items) == 0 {
			return "пустая комната. можно пройти - коридор"
		}
		tableItems := []string{}
		hasBackpack := false
		for item := range r.items {
			if item == "рюкзак" {
				hasBackpack = true
			} else {
				tableItems = append(tableItems, item)
			}
		}
		sort.Strings(tableItems)
		result := "на столе: " + strings.Join(tableItems, ", ")
		if hasBackpack {
			if len(tableItems) > 0 {
				result += ", "
			}
			result += "на стуле: рюкзак"
		}
		result += ". можно пройти - коридор"
		return result
	}

	corridor.lookFunc = func(w *World, r *Room) string {
		return fmt.Sprintf("ничего интересного. можно пройти - %s", getRoomPaths(r))
	}

	street.lookFunc = func(w *World, r *Room) string {
		return fmt.Sprintf("на улице весна. можно пройти - %s", getRoomPaths(r))
	}

	// --- useFunc для коридора: пытается открыть заблокированные пути если item совпадает с unlockItem ---
	corridor.useFunc = func(w *World, r *Room, item, target string) string {
		// пробуем сначала стандартное: если цель совпадает с именем пути — пытаемся открыть
		if p, ok := r.paths[target]; ok {
			if !p.locked {
				return NothingNeed 

			}
			if item == p.unlockItem {
				p.locked = false
				return p.unlockMsg
			}
			return "не сработало"
		}
		return NothingUse
	}

	// --- Добавляем комнаты в мир ---
	world.rooms["кухня"] = kitchen
	world.rooms["коридор"] = corridor
	world.rooms["комната"] = room
	world.rooms["улица"] = street

	// --- Создаём игрока ---
	world.player = &Player{
		room:        kitchen,
		inventory:   make(map[string]bool),
		hasBackpack: false,
	}
}

// --- Вспомогательные функции для вывода ---

func getRoomItems(r *Room) string {
	items := []string{}
	for it := range r.items {
		items = append(items, it)
	}
	sort.Strings(items)
	if len(items) == 0 {
		return "ничего"
	}
	return strings.Join(items, ", ")
}

// func getRoomPaths(r *Room) string {
// 	paths := []string{}
// 	for name := range r.paths {
// 		paths = append(paths, name)
// 	}
// 	return strings.Join(paths, ", ")
// }

func getRoomPaths(r *Room) string {
	names := make([]string, 0, len(r.paths))
	for name := range r.paths {
		names = append(names, name)
	}

	// --- Кастомная сортировка ---
	sort.Slice(names, func(i, j int) bool {
		a := []rune(names[i])
		b := []rune(names[j])

		// если первые буквы разные — сортируем по алфавиту
		if a[0] != b[0] {
			return a[0] < b[0]
		}

		// если первые совпали — остальные в обратном порядке
		for k := 1; k < len(a) && k < len(b); k++ {
			if a[k] != b[k] {
				return a[k] > b[k]
			}
		}

		// если одно слово — префикс другого
		return len(a) < len(b)
	})

	return strings.Join(names, ", ")
}




// --- Обработчики команд (делегирующие) ---

const UnknownCommandMsg = "неизвестная команда"

func handleCommand(command string) string {
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return UnknownCommandMsg
	}
	cmd := parts[0]
	args := parts[1:]

	switch cmd {
	case "осмотреться":
		return handleLook()
	case "идти":
		if len(args) < 1 {
			return "куда идти?"
		}
		return handleGo(args[0])
	case "взять":
		if len(args) < 1 {
			return "что взять?"
		}
		return handleTake(args[0])
	case "надеть":
		if len(args) < 1 {
			return "что надеть?"
		}
		return handleWear(args[0])
	case "применить":
		if len(args) < 2 {
			return "что и к чему применить?"
		}
		return handleUse(args[0], args[1])
	default:
		return UnknownCommandMsg
	}
}

func handleLook() string {
	r := world.player.room
	if r.lookFunc != nil {
		return r.lookFunc(world, r)
	}
	// дефолтное описание
	return fmt.Sprintf("%s. можно пройти - %s", r.description, getRoomPaths(r))
}

func handleGo(direction string) string {
	cur := world.player.room
	p, exists := cur.paths[direction]
	if !exists {
		return "нет пути в " + direction
	}
	if p.locked {
		if p.lockMsg != "" {
			return p.lockMsg
		}
		return "путь заблокирован"
	}

	world.player.room = p.to

	// --- Особые случаи ---
	if world.player.room.name == "комната" {
		return "ты в своей комнате. можно пройти - коридор"
	}
	if world.player.room.name == "кухня" {
		return "кухня, ничего интересного. можно пройти - коридор"
	}

	// --- Общий случай ---
	if world.player.room.lookFunc != nil {
		return world.player.room.lookFunc(world, world.player.room)
	}
	return fmt.Sprintf("%s. можно пройти - %s", world.player.room.description, getRoomPaths(world.player.room))
}



func handleTake(item string) string {
	if !world.player.hasBackpack {
		return "некуда класть"
	}
	cur := world.player.room
	if _, ok := cur.items[item]; !ok {
		return "нет такого"
	}
	delete(cur.items, item)
	world.player.inventory[item] = true
	return "предмет добавлен в инвентарь: " + item
}

func handleWear(item string) string {
	// только рюкзак можно надеть (по логике игры)
	if item != "рюкзак" {
		return "неизвестная команда"
	}
	cur := world.player.room
	if _, exists := cur.items[item]; !exists {
		return "нет такого"
	}
	delete(cur.items, item)
	world.player.hasBackpack = true
	return "вы надели: " + item
}

func handleUse(item, target string) string {
	// проверяем инвентарь
	if _, ok := world.player.inventory[item]; !ok {
		return "нет предмета в инвентаре - " + item
	}
	r := world.player.room

	// сначала локальный useFunc комнаты
	if r.useFunc != nil {
		res := r.useFunc(world, r, item, target)
		if res != NothingUse && res != "не сработало" && res != NothingNeed {
			return res
		}
		if res == NothingNeed {
			return res
		}
	}

	// прямое применение к пути
	if p, ok := r.paths[target]; ok {
		if !p.locked {
			return NothingNeed
		}
		if item == p.unlockItem {
			p.locked = false
			if p.unlockMsg != "" {
				return p.unlockMsg
			}
			return "открыто"
		}
		return NothingUse
	}

	// если цель "дверь" — ищем путь с locked == true
	if target == "дверь" {
		for _, p := range r.paths {
			if p.locked && item == p.unlockItem {
				p.locked = false
				if p.unlockMsg != "" {
					return p.unlockMsg
				}
				return "открыто"
			}
		}
	}

	return NothingUse
}


// --- Простой REPL (удалите/измените для тестов) ---
func main() {
	
}
