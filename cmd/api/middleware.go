package main

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/tomasen/realip"
	"golang.org/x/time/rate"
)

// recoverPanic - обработка паники.
func (app *application) recoverPanic(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// создадим ф-ию котоая всегда выполняется в случае паники
		defer func() {
			// используем встроенную ф-ию восстановления, которая проверяет произощла паника или нет
			if err := recover(); err != nil {
				// в случае паники: установить заголовок "Connection: close"/
				// Это послужит триггером для закрытия соединния после отправки
				w.Header().Set("Connection", "close")
				app.serverErrorResponse(w, r, fmt.Errorf("%s", err))
			}
		}()

		next.ServeHTTP(w, r)
	})
}

// rateLimit - ограничитель запросов. корзина с токенами пополняется со скоростью два токена в секунду.
func (app *application) rateLimit(next http.Handler) http.Handler {

	// клиентская структуру для хранения ограничителя скорости и времени последнего посещения для каждого клиента
	type client struct {
		limiter  *rate.Limiter
		lastSeen time.Time
	}
	// мьютекс и карт для хранения IP-адресов клиентов и ограничителей скорости
	var (
		mu      sync.Mutex
		clients = make(map[string]*client)
	)
	// фоновая горутина, раз в минуту удаляет старые записи с карты клиентов.
	go func() {
		for {
			time.Sleep(time.Minute)
			// Заблокируйте мьютекс, чтобы предотвратить выполнение любых проверок ограничителя скорости во время очистки.
			mu.Lock()
			// Перебираем всех клиентов.
			// Если их не было видно в течение последних трех минут, удаляем соответствующую запись с карты
			for ip, client := range clients {
				if time.Since(client.lastSeen) > 3*time.Minute {
					delete(clients, ip)
				}
			}
			//  разблокировать мьютекс после завершения очистки
			mu.Unlock()
		}
	}()

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// проверка выполняется если включен соотв-ий флаг
		if app.config.limiter.enabled {
			// получить IP-адрес клиента
			ip := realip.FromRequest(r)
			// Заблокируйте мьютекс, чтобы этот код не выполнялся одновременно с другими
			mu.Lock()
			// Проверьте, есть ли этот IP-адрес на карте. Если нет, то
			// инициализируйте новый ограничитель скорости и добавьте IP-адрес и ограничитель на карту
			if _, found := clients[ip]; !found {
				// Создайте и добавьте на карту новую структуру клиента, если она еще не существует.
				// инициализируем новый ограничитель запросов в среднем 2 запроса/сек, макс 4
				clients[ip] = &client{
					// Используйте значения количества запросов в секунду и пиковой нагрузки из конфигурации структуры
					limiter: rate.NewLimiter(rate.Limit(app.config.limiter.rps), app.config.limiter.burst),
				}
			}
			// Обновите время последнего посещения клиента.
			clients[ip].lastSeen = time.Now()

			// метод Allow() для ограничителя скорости текущего IP-адреса.
			// Если запрос не разрешен, разблокируйте мьютекс и отправьте ответ с кодом 429
			if !clients[ip].limiter.Allow() {
				mu.Unlock()
				app.rateLimitExceededResponse(w, r)
				return
			}
			// разблокировать мьютекс перед вызовом следующего обработчика в цепочке.
			mu.Unlock()
		}

		next.ServeHTTP(w, r)
	})
}
