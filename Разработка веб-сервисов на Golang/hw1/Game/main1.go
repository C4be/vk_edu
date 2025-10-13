 package main

// import (
// 	"fmt"
// 	"sort"
// 	"strings"
// )

// // Player представляет игрока
// type Player struct {
// 	room        *Room
// 	inventory   map[string]bool
// 	hasBackpack bool
// }

// // Room представляет комнату
// type Room struct {
// 	name        string
// 	description string
// 	items       map[string]bool
// 	paths       map[string]*Room
// 	lookFunc    func() string
// 	useFunc     func(string, string) string // Изменена сигнатура для поддержки цели
// }

// // World представляет игровой мир
// type World struct {
// 	rooms      map[string]*Room
// 	player     *Player
// 	doorLocked bool
// }

// var world *World

// func initGame() {
// 	world = &World{
// 		rooms:      make(map[string]*Room),
// 		doorLocked: true,
// 	}

// 	// Создаем комнаты
// 	kitchen := &Room{
// 		name:        "кухня",
// 		description: "кухня, ничего интересного",
// 		items:       make(map[string]bool),
// 		paths:       make(map[string]*Room),
// 	}

// 	corridor := &Room{
// 		name:        "коридор",
// 		description: "ничего интересного",
// 		items:       make(map[string]bool),
// 		paths:       make(map[string]*Room),
// 	}

// 	room := &Room{
// 		name:        "комната",
// 		description: "ты в своей комнате",
// 		items:       make(map[string]bool),
// 		paths:       make(map[string]*Room),
// 	}

// 	street := &Room{
// 		name:        "улица",
// 		description: "на улице весна",
// 		items:       make(map[string]bool),
// 		paths:       make(map[string]*Room),
// 	}

// 	// Добавляем предметы в комнаты
// 	kitchen.items["чай"] = true
// 	room.items["ключи"] = true
// 	room.items["конспекты"] = true
// 	room.items["рюкзак"] = true

// 	// Настраиваем пути
// 	kitchen.paths["коридор"] = corridor

// 	corridor.paths["кухня"] = kitchen
// 	corridor.paths["комната"] = room
// 	corridor.paths["улица"] = street

// 	room.paths["коридор"] = corridor

// 	street.paths["домой"] = corridor

// 	// Специальные функции для комнат
// 	kitchen.lookFunc = func() string {
// 		items := getRoomItems(kitchen)
// 		paths := getRoomPaths(kitchen)

// 		if !world.player.hasBackpack {
// 			return fmt.Sprintf("ты находишься на кухне, на столе: %s, надо собрать рюкзак и идти в универ. можно пройти - %s", items, paths)
// 		}
// 		return fmt.Sprintf("ты находишься на кухне, на столе: %s, надо идти в универ. можно пройти - %s", items, paths)
// 	}

// 	room.lookFunc = func() string {
// 		if len(room.items) == 0 {
// 			return "пустая комната. можно пройти - коридор"
// 		}

// 		tableItems := []string{}
// 		hasBackpack := false

// 		for item := range room.items {
// 			if item == "рюкзак" {
// 				hasBackpack = true
// 			} else {
// 				tableItems = append(tableItems, item)
// 			}
// 		}

// 		// ИСПРАВЛЕНИЕ: Добавлена сортировка предметов на столе для консистентного вывода
// 		sort.Strings(tableItems)

// 		result := "на столе: " + strings.Join(tableItems, ", ")
// 		if hasBackpack {
// 			if len(tableItems) > 0 {
// 				result += ", "
// 			}
// 			result += "на стуле: рюкзак"
// 		}
// 		result += ". можно пройти - коридор"

// 		return result
// 	}

// 	corridor.lookFunc = func() string {
// 		return fmt.Sprintf("ничего интересного. можно пройти - %s", getRoomPaths(corridor))
// 	}

// 	street.lookFunc = func() string {
// 		return fmt.Sprintf("на улице весна. можно пройти - %s", getRoomPaths(street))
// 	}

// 	// ИСПРАВЛЕНИЕ: Функция теперь принимает и обрабатывает цель (target)
// 	corridor.useFunc = func(item, target string) string {
// 		if item == "ключи" && target == "дверь" && world.doorLocked {
// 			world.doorLocked = false
// 			return "дверь открыта"
// 		}
// 		return "не к чему применить"
// 	}

// 	// Добавляем комнаты в мир
// 	world.rooms["кухня"] = kitchen
// 	world.rooms["коридор"] = corridor
// 	world.rooms["комната"] = room
// 	world.rooms["улица"] = street

// 	// Создаем игрока
// 	world.player = &Player{
// 		room:        kitchen,
// 		inventory:   make(map[string]bool),
// 		hasBackpack: false,
// 	}
// }

// // Вспомогательные функции
// func getRoomItems(room *Room) string {
// 	items := []string{}
// 	for item := range room.items {
// 		items = append(items, item)
// 	}
// 	sort.Strings(items)
// 	return strings.Join(items, ", ")
// }

// func getRoomPaths(room *Room) string {
// 	paths := []string{}
// 	for path := range room.paths {
// 		paths = append(paths, path)
// 	}
// 	return strings.Join(paths, ", ")
// }

// const UnknownCommandMsg = "неизвестная команда"

// func handleCommand(command string) string {
// 	parts := strings.Fields(command) // Используем Fields для лучшей обработки пробелов
// 	if len(parts) == 0 {
// 		return UnknownCommandMsg
// 	}

// 	cmd := parts[0]

// 	switch cmd {
// 	case "осмотреться":
// 		return handleLook()
// 	case "идти":
// 		if len(parts) < 2 {
// 			return "куда идти?"
// 		}
// 		return handleGo(parts[1])
// 	case "взять":
// 		if len(parts) < 2 {
// 			return "что взять?"
// 		}
// 		return handleTake(parts[1])
// 	case "надеть":
// 		if len(parts) < 2 {
// 			return "что надеть?"
// 		}
// 		return handleWear(parts[1])
// 	case "применить":
// 		if len(parts) < 3 {
// 			return "что и к чему применить?" // Уточнено сообщение
// 		}
// 		return handleUse(parts[1], parts[2])
// 	default:
// 		return "неизвестная команда"
// 	}
// }

// func handleLook() string {
// 	room := world.player.room

// 	if room.lookFunc != nil {
// 		return room.lookFunc()
// 	}

// 	return fmt.Sprintf("%s. можно пройти - %s", room.description, getRoomPaths(room))
// }

// func handleGo(direction string) string {
// 	targetRoom, exists := world.player.room.paths[direction]
// 	if !exists {
// 		return "нет пути в " + direction
// 	}

// 	// Проверка на заблокированную дверь
// 	if world.player.room == world.rooms["коридор"] && direction == "улица" && world.doorLocked {
// 		return "дверь закрыта"
// 	}

// 	world.player.room = targetRoom

// 	// Для комнаты при переходе показываем базовое описание
// 	if targetRoom.name == "комната" {
// 		return "ты в своей комнате. можно пройти - коридор"
// 	}

// 	// Для кухни при переходе показываем базовое описание
// 	if targetRoom.name == "кухня" {
// 		return "кухня, ничего интересного. можно пройти - коридор"
// 	}

// 	// Вызываем lookFunc, чтобы получить полное описание с правильной сортировкой
// 	if targetRoom.lookFunc != nil {
// 		return targetRoom.lookFunc()
// 	}

// 	return fmt.Sprintf("%s. можно пройти - %s", targetRoom.description, getRoomPaths(targetRoom))
// }

// func handleTake(item string) string {
// 	if !world.player.hasBackpack {
// 		return "некуда класть"
// 	}

// 	_, exists := world.player.room.items[item]
// 	if !exists {
// 		return "нет такого"
// 	}

// 	delete(world.player.room.items, item)
// 	world.player.inventory[item] = true
// 	return "предмет добавлен в инвентарь: " + item
// }

// func handleWear(item string) string {
// 	if item != "рюкзак" {
// 		return "неизвестная команда"
// 	}

// 	_, exists := world.player.room.items[item]
// 	if !exists {
// 		return "нет такого"
// 	}

// 	delete(world.player.room.items, item)
// 	world.player.hasBackpack = true
// 	return "вы надели: " + item
// }

// func handleUse(item, target string) string {
// 	_, hasItem := world.player.inventory[item]
// 	if !hasItem {
// 		return "нет предмета в инвентаре - " + item
// 	}
// 	// ИСПРАВЛЕНИЕ: Передаем в useFunc и предмет, и цель
// 	if world.player.room.useFunc != nil {
// 		return world.player.room.useFunc(item, target)
// 	}

// 	return "не к чему применить"
// }

// func main() {
// 	/*
// 		в этой функции можно ничего не писать,
// 		но тогда у вас не будет работать через go run main.go
// 		очень круто будет сделать построчный ввод команд тут, хотя это и не требуется по заданию
// 	*/
// }
