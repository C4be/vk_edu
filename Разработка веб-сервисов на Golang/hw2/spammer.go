package main

import (
	"fmt"
	"sort"
	"sync"
)

// RunPipeline запускает конвейер из cmd-функций.
// Каждая функция выполняется в отдельной горутине.
// STDOUT (out chan) одной функции становится STDIN (in chan) для следующей.
func RunPipeline(cmds ...cmd) {
	// Используем WaitGroup для ожидания завершения всех стадий конвейера
	wg := new(sync.WaitGroup)

	// Создаем начальный (входной) канал.
	// Так как первая cmd (обычно) только пишет, но не читает,
	// мы сразу закрываем этот канал, чтобы range по нему (если он есть) сразу завершился.
	in := make(chan interface{})
	close(in)

	for _, c := range cmds {
		wg.Add(1)
		// Создаем выходной канал для *текущей* cmd
		out := make(chan interface{})

		// Запускаем cmd в отдельной горутине
		go func(in, out chan interface{}, c cmd) {
			defer wg.Done()
			// Когда cmd завершает работу, она *обязана* закрыть свой выходной канал.
			// Это сигнализирует следующей стадии конвейера, что данных больше не будет.
			defer close(out)
			c(in, out)
		}(in, out, c)

		// Выходной канал текущей cmd становится входным для *следующей*
		in = out
	}

	// Ждем, пока все горутины (стадии конвейера) не завершатся.
	// Последний канал `in` (он же `out` последней cmd)
	// будет закрыт, но из него никто не будет читать (кроме тестов).
	wg.Wait()
}

// SelectUsers получает email'ы, вызывает GetUser параллельно и отдает *уникальных* пользователей.
func SelectUsers(in, out chan interface{}) {
	wg := new(sync.WaitGroup)
	// Карта для отслеживания уникальных ID пользователей, чтобы избежать дубликатов из-за алиасов.
	seen := make(map[uint64]bool)
	// Мутекс для безопасной работы с картой `seen` из разных горутин.
	mu := new(sync.Mutex)

	for emailRaw := range in {
		email := emailRaw.(string)
		wg.Add(1)

		go func(email string) {
			defer wg.Done()
			user := GetUser(email)

			// Блокируем мутекс для проверки и записи в карту
			mu.Lock()
			// GetUser возвращает каноничного юзера, поэтому проверяем по ID
			if !seen[user.ID] {
				seen[user.ID] = true
				// Разблокируем *до* отправки в канал,
				// чтобы не блокировать другие горутины, пока эта ждет отправки.
				mu.Unlock()
				out <- user
			} else {
				// Если юзер уже был, просто разблокируем
				mu.Unlock()
			}
		}(email)
	}

	// Ждем, пока *все* вызовы GetUser не завершатся,
	// и только потом завершаем функцию (чтобы RunPipeline закрыл out канал).
	wg.Wait()
}

// SelectMessages получает пользователей, вызывает GetMessages параллельно,
// используя *оптимальные* батчи.
func SelectMessages(in, out chan interface{}) {
	wg := new(sync.WaitGroup)
	// Слайс для накопления батча пользователей
	batch := make([]User, 0, GetMessagesMaxUsersBatch)

	// Хелпер-функция для обработки батча в отдельной горутине
	processBatch := func(usersBatch []User) {
		defer wg.Done()
		// Копируем батч, чтобы избежать проблем с гонкой,
		// так как `batch` в цикле будет переиспользован.
		// (Хотя в данном коде мы создаем новый слайс `batch = make(...)`,
		// но это хорошая практика).
		// В данном случае, так как мы создаем новый слайс `batch = make...`
		// после запуска горутины, копирование не обязательно,
		// но для надежности можно было бы сделать.
		// В этой реализации мы просто передаем слайс, который больше не модифицируется.
		msgs, err := GetMessages(usersBatch...)
		if err == nil {
			for _, msg := range msgs {
				out <- msg
			}
		}
	}

	for userRaw := range in {
		user := userRaw.(User)
		batch = append(batch, user)

		// Как только батч наполнен, запускаем обработку
		if len(batch) == GetMessagesMaxUsersBatch {
			wg.Add(1)
			go processBatch(batch)
			// Создаем новый слайс для следующего батча
			batch = make([]User, 0, GetMessagesMaxUsersBatch)
		}
	}

	// После завершения цикла `range in` мог остаться неполный батч.
	// Его тоже нужно обработать.
	if len(batch) > 0 {
		wg.Add(1)
		go processBatch(batch)
	}

	// Ждем, пока *все* батчи не обработаются.
	wg.Wait()
}

// CheckSpam получает MsgID, вызывает HasSpam параллельно,
// но с ограничением на 5 одновременных запросов.
func CheckSpam(in, out chan interface{}) {
	wg := new(sync.WaitGroup)
	// Используем буферизированный канал как семафор для ограничения
	// одновременных вызовов HasSpam.
	sem := make(chan struct{}, HasSpamMaxAsyncRequests)

	for msgIDRaw := range in {
		msgID := msgIDRaw.(MsgID)
		wg.Add(1)
		// "Захватываем" слот в семафоре.
		// Если семафор полон (5 горутин уже работают), эта строка заблокируется.
		sem <- struct{}{}

		go func(id MsgID) {
			defer wg.Done()
			// "Освобождаем" слот в семафоре по завершении горутины.
			defer func() { <-sem }()

			hasSpam, err := HasSpam(id)
			// Мы не обрабатываем ошибку `too many requests` явно,
			// так как наш семафор *гарантирует*, что ее не будет.
			// Другие ошибки (если бы они были) тоже можно было бы проигнорировать.
			// В задании сказано, что при ошибке мы не получим данных,
			// но наш код предотвращает саму ошибку.
			if err == nil {
				out <- MsgData{ID: id, HasSpam: hasSpam}
			}
			// Если бы HasSpam вернула ошибку, мы бы просто не послали MsgData дальше.
		}(msgID)
	}

	// Ждем, пока *все* проверки на спам не завершатся.
	wg.Wait()
}

// CombineResults получает *все* MsgData, сортирует их и выдает
// в виде отформатированных строк.
func CombineResults(in, out chan interface{}) {
	// Это единственная функция-агрегатор.
	// Сначала собираем все результаты.
	results := make([]MsgData, 0)
	for msgDataRaw := range in {
		results = append(results, msgDataRaw.(MsgData))
	}

	// Сортируем по условию:
	// 1. Сначала `HasSpam = true`, потом `HasSpam = false`
	// 2. При одинаковом `HasSpam` - по возрастанию `MsgID`
	sort.Slice(results, func(i, j int) bool {
		// Сравниваем по HasSpam
		if results[i].HasSpam != results[j].HasSpam {
			// Мы хотим `true` (спам) *раньше*, чем `false` (не спам).
			// `sort.Slice` использует `less` функцию.
			// `return results[i].HasSpam` вернет `true`, если `i` - спам, а `j` - нет.
			// Это значит `i` "меньше" `j`, т.е. `i` будет раньше в списке.
			return results[i].HasSpam
		}
		// Если HasSpam одинаковый, сортируем по ID (по возрастанию)
		return results[i].ID < results[j].ID
	})

	// Отправляем отсортированные и отформатированные результаты
	for _, res := range results {
		out <- fmt.Sprintf("%t %d", res.HasSpam, res.ID)
	}
}