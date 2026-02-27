package main

import (
	"errors"
	"expvar"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/len4ernova/lets_go_further/internal/data"
	"github.com/len4ernova/lets_go_further/internal/validator"
	"github.com/pascaldekloe/jwt"
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

// authenticate - аутентификация.
// Если в Authorization присутствует действительный токен, то в контекст запроса будут добавлены данные пользователя.
// Если Authorization не представлен, то запишем AnonymousUser.
// Иначе  401 Unauthorized.
func (app *application) authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// добавим заголовок "Vary: Authorization".
		// Сообщает кешу, что ответ может отличаться в зависимости от значения в заголовке Authorization
		w.Header().Add("Very", "Authorization")

		// Получим значение "Authorization" из заголовка запроса.
		// если отсутсвует, вернуть получим ""
		authorizationHeader := r.Header.Get("Authorization")

		// если заголовка не было, добавим анонимного пользователя в контекст запроса.
		// Затем вызовим next обработчик.
		if authorizationHeader == "" {
			r = app.contextSetUser(r, data.AnonymousUser)
			next.ServeHTTP(w, r)
			return
		}

		// иначе ожидаем получить "Bearer <token>".
		// Пытаемся его разбить на части. Если не получилось, вернуть 401.
		headerParts := strings.Split(authorizationHeader, " ")
		if len(headerParts) != 2 || headerParts[0] != "Bearer" {
			app.invalidAuthenticationTokenResponse(w, r)
			return
		}

		// излекаем токен
		token := headerParts[1]

		// валидация токена
		v := validator.New()

		if data.ValidateTokenPlaintext(v, token); !v.Valid() {
			app.invalidCredentialsResponse(w, r)
			return
		}

		// Пробуем получить данные связанные с токеном.
		// если запись не найдена, то вернуть 401.
		// Важно: ScopeAuthentication - первый пар-р
		user, err := app.models.Users.GetForToken(data.ScopeAuthentication, token)
		if err != nil {
			switch {
			case errors.Is(err, data.ErrRecordNotFound):
				app.invalidAuthenticationTokenResponse(w, r)
			default:
				app.serverErrorResponse(w, r, err)
			}
			return
		}

		// добавим инфо о user в контекст запроса
		r = app.contextSetUser(r, user)

		// вызов next handler в цепочке
		next.ServeHTTP(w, r)
	})
}

// authenticateJWT - проверка аутентификации JWT
func (app *application) authenticateJWT(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// добавим заголовок "Vary: Authorization".
		// Сообщает кешу, что ответ может отличаться в зависимости от значения в заголовке Authorization
		w.Header().Add("Very", "Authorization")

		// Получим значение "Authorization" из заголовка запроса.
		// если отсутсвует, вернуть получим ""
		authorizationHeader := r.Header.Get("Authorization")

		// если заголовка не было, добавим анонимного пользователя в контекст запроса.
		// Затем вызовим next обработчик.
		if authorizationHeader == "" {
			r = app.contextSetUser(r, data.AnonymousUser)
			next.ServeHTTP(w, r)
			return
		}

		// иначе ожидаем получить "Bearer <token>".
		// Пытаемся его разбить на части. Если не получилось, вернуть 401.
		headerParts := strings.Split(authorizationHeader, " ")
		if len(headerParts) != 2 || headerParts[0] != "Bearer" {
			app.invalidAuthenticationTokenResponse(w, r)
			return
		}

		// излекаем токен
		token := headerParts[1]

		// парсим токен и излекаем полезную нагрузку
		claims, err := jwt.HMACCheck([]byte(token), []byte(app.config.jwt.secret))
		if err != nil {
			app.invalidAuthenticationTokenResponse(w, r)
			return
		}

		// проверка валидности JWT
		if !claims.Valid(time.Now()) {
			app.invalidCredentialsResponse(w, r)
			return
		}

		// подтвердим этитента
		if claims.Issuer != "greenlight.alexedwards.net" {
			app.invalidAuthenticationTokenResponse(w, r)
			return
		}

		// проверка целевой аудитории
		if !claims.AcceptAudience("greenlight.alexedwards.net") {
			app.invalidAuthenticationTokenResponse(w, r)
			return
		}

		// jwt в порядке, излечем ID пользователя
		userID, err := strconv.ParseInt(claims.Subject, 10, 64)
		if err != nil {
			app.serverErrorResponse(w, r, err)
			return
		}
		// найти запись пользователя в БД
		user, err := app.models.Users.Get(userID)
		if err != nil {
			switch {
			case errors.Is(err, data.ErrRecordNotFound):
				app.invalidAuthenticationTokenResponse(w, r)
			default:
				app.serverErrorResponse(w, r, err)
			}
			return
		}

		// добавим инфо о user в контекст запроса и продолжить работу
		r = app.contextSetUser(r, user)

		// вызов next handler в цепочке
		next.ServeHTTP(w, r)
	})
}

// requireActivatedUser - функция оборачивает обработчики для проверки аутентификации и активации.
func (app *application) requireActivatedUser(next http.HandlerFunc) http.HandlerFunc {
	fn := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// получим инфо о пользователе из контекста запроса
		user := app.ContextGetUser(r)

		// если учетная запись на активирована, то проинформируем клиента 401
		if !user.Activated {
			app.inactiveAccountREsponse(w, r)
			return
		}
		// вызываем next handler
		next.ServeHTTP(w, r)
	})

	// оборачиваем промежуточным ПО
	return app.requireAuthenticatedUser(fn)
}

// проверяет что user не anonymous
func (app *application) requireAuthenticatedUser(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := app.ContextGetUser(r)
		if user.IsAnonymous() {
			app.authenticationRequiredResponse(w, r)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// requirePermission - проверяет что у пользователя есть необходимые права.
func (app *application) requirePermission(code string, next http.HandlerFunc) http.HandlerFunc {
	fn := func(w http.ResponseWriter, r *http.Request) {
		// получить пользователя из контекста запроса
		user := app.ContextGetUser(r)

		// получить слайс разрешениий доступа для user
		permissions, err := app.models.Permissions.GetAllForUser(user.ID)
		if err != nil {
			app.serverErrorResponse(w, r, err)
			return
		}

		// проверка содержит ли необходимые разрешения
		if !permissions.Include(code) {
			app.notPermittedResponse(w, r)
			return
		}
		// если разрешения есть, вызываем обработчик
		next.ServeHTTP(w, r)
	}
	// обернуть requireActivatedUser
	return app.requireActivatedUser(fn)
}

// enableCORS - метод устанавливает заголовок "Access-Control-Allow-Origin" CORS.
// Предварительно прояверяется является ли источник доверенным.
func (app *application) enableCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// добавить заголовки, чтобы сообщить браузеру что ответ может отличаться в зависимости от наличия этих заголовков.
		w.Header().Add("Vary", "Origin")
		w.Header().Add("Vary", "Access-Control-Request-Method")

		// зпросим значения Origin из заголовка
		origin := r.Header.Get("Origin")

		// выполняется если Origin присутствует
		if origin != "" {
			// проходим циклом по доверенныым источникам и ищем сопадение.
			// в случае успеха установим "Access-Control-Allow-Origin"
			for i := range app.config.cors.trustedOrigins {
				if origin == app.config.cors.trustedOrigins[i] {
					w.Header().Set("Access-Control-Allow-Origin", origin)

					// проверим не содержит ли запрос мутод OPTIONS, "Access-Control-Request-Method" header
					// если содержит, то мы считаем это предварительным запросом
					if r.Method == http.MethodOptions && r.Header.Get("Access-Control-Request-Method") != "" {
						// установить необходимые заголовки предварительного ответа
						w.Header().Set("Access-Control-Allow-Methods", "OPTIONS, PUT, PATCH, DELETE")
						w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
						// ! важно не устанавливать "Access-Control-Allow-Origin: *" и проверять из доверенных источников,
						// т.к. подвержены атаке brute-force
						// записать заголоки со статусом 200 и вернуться из мидлеваре без предварительных действий
						w.WriteHeader(http.StatusOK)
						return
					}
					break
				}
			}
		}

		next.ServeHTTP(w, r)
	})
}

// metrics - запись пользовательских метрик на уровне запросов.
func (app *application) metrics(next http.Handler) http.Handler {
	// инициализируем новые переменный expvar
	var (
		totalRequestsReceived           = expvar.NewInt("total_requests_received")
		totalResponsesSent              = expvar.NewInt("total_responses_sent")
		totalProcessingTimeMicroseconds = expvar.NewInt("total_processing_time_ms")
		totalActiveResponses            = expvar.NewInt("total_active_responses")

		totalResponsesSentByStatus = expvar.NewMap("total_responses_sent_by_status") // хранения количества ответов для каждого HTTP-статуса код
	)

	// выполняется при каждом запросе
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// формируем время начала обработки запроса
		start := time.Now()
		// увеличить кол-во запросов

		totalRequestsReceived.Add(1)

		//обернуть исходное значение http.ResponseWriter, полученное промежуточным ПО для сбора метрик
		mw := newMetricsResponseWriter(w)

		next.ServeHTTP(mw, r)

		// // вызвать след. handler в цепочке
		// next.ServeHTTP(w, r)

		// при возврате вверх по мидлваре увеличим кол-во ответов
		totalResponsesSent.Add(1)
		// код состояния ответа должен быть сохранен в поле
		// mw.statusCode. Обратите внимание, что ключ в карте expvar — это строка,
		// поэтому нам нужно использовать функцию strconv.Itoa(),
		// чтобы преобразовать код состояния (целое число) в строку.
		// Затем мы используем метод Add() на нашей новой карте totalResponsesSentByStatus,
		// чтобы увеличить счетчик для данного кода состояния на 1.
		totalResponsesSentByStatus.Add(strconv.Itoa(mw.statusCode), 1)

		//посчитать кол-во милесукунд затраченных на обработку запроса и добавим к общему
		duration := time.Since(start).Milliseconds()
		totalProcessingTimeMicroseconds.Add(duration)

		// кол-во активных запросов
		activeResponses := totalRequestsReceived.Value() - totalResponsesSent.Value()
		fmt.Println(totalRequestsReceived.Value(), totalResponsesSent.Value())
		totalActiveResponses.Set(activeResponses)

	})
}

// оборачиваем ResponseWriter, поле для записи кода состояния, логическое поле - записан или нет код состояния
type metricsResponseWriter struct {
	wrapped       http.ResponseWriter
	statusCode    int
	headerWritten bool
}

// newMetricsResponseWriter - функция оборачивает исхлдный ResponseWriter
// и записывает код 200 (по ум. возвр-ся в ответе)
func newMetricsResponseWriter(w http.ResponseWriter) *metricsResponseWriter {
	return &metricsResponseWriter{
		wrapped:    w,
		statusCode: http.StatusOK,
	}
}

// Header - метод выполняет "сквозной переход" к методу Header()
func (mw *metricsResponseWriter) Header() http.Header {
	return mw.wrapped.Header()
}

// WriteHeader() выполняет "сквозной переход" к методу WriteHeader()
// обернутого http.ResponseWriter. Но после того, как это значение вернется,
// мы также запишем код состояния ответа (если он еще не был записан)
// и установим для поля headerWritten значение true, чтобы указать, что заголовки HTTP-ответа уже записаны
func (mw *metricsResponseWriter) WriteHeader(statusCode int) {
	mw.wrapped.WriteHeader(statusCode)
	if !mw.headerWritten {
		mw.statusCode = statusCode
		mw.headerWritten = true
	}
}

// Write() «передает» вызов методу Write() объекта
//
//	обернутого http.ResponseWriter. При вызове этого метода автоматически записываются все
//	заголовки ответа, поэтому мы устанавливаем для поля headerWritten значение true
func (mw *metricsResponseWriter) Write(b []byte) (int, error) {
	mw.headerWritten = true
	return mw.wrapped.Write(b)
}

// Unwrap() возвращает существующий обернутый объект http.ResponseWriter
func (mw *metricsResponseWriter) Unwrap() http.ResponseWriter {
	return mw.wrapped
}
