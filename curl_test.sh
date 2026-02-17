# проверка валидного (невалидного) запроса /movies

BODY='{"title":"Moana","year":2016,"runtime":"107 mins","genres":["animation","adventure"]}'
# BODY='{"title":"","year":1000,"runtime":"-123 mins","genres":["sci-fi","sci-fi"]}'

curl -i -d "$BODY" localhost:4000/v1/movies

# подключение к БД PG(user/18)
psql --host=localhost --dbname=greenlight --username=user

# просмотр пути к конфиг.файлу
sudo -u postgres psql -c 'SHOW config_file;'


# add  $HOME/.profile or  $HOME/.bashrc
export GREENLIGHT_DB_DSN='postgres://user:pass@localhost/greenlight'
#reboot computer OR :
source $HOME/.profile

#так же можно подключиться к БД через переменную окружения
psql $GREENLIGHT_DB_DSN

# изменить настройки пула соединений БД
go run ./cmd/api -db-max-open-conns=50 -db-max-idle-conns=50 -db-max-idle-time=2h30m

#migrate
# create table ..  | DROP table..
 migrate create -seq -ext=.sql -dir=./migrations create_movies_table

#CHECK
 migrate create -seq -ext=.sql -dir=./migrations add_movies_check_constraints

# узнать версию миграции
 migrate -path=./migrations -database=$EXAMPLE_DSN version

# миграция к опред. версии
migrate -path=./migrations -database=$EXAMPLE_DSN goto 1

#  откатить последнюю миграцию
 migrate -path=./migrations -database =$EXAMPLE_DSN down 1

# откат всех миграций
migrate -path=./migrations -database=$EXAMPLE_DSN down

# fix синтаксических ошибок
# найти в чем ошибка. вручную откатиться к стабильной версии
# вручную установить номер версии 
migrate -path=./migrations -database=$EXAMPLE_DSN force 1

# чтение файлов миграции из удаленных источников
# https://github.com/golang-migrate/migrate#migration-sources
migrate -source="s3://<bucket>/<path>" -database=$EXAMPLE_DSN up
migrate -source="github://owner/repo/path#ref" -database=$EXAMPLE_DSN up
migrate -source="github://user:personal-access-token@owner/repo/path#ref" -database=$EXAMPLE_DSN up

# --------------------------------
# данные для БД: фильмы
BODY='{"title":"Moana","year":2016,"runtime":"107 mins","genres":["animation","adventure"]}'
BODY='{"title":"Black Panther","year":2018,"runtime":"134 mins","genres":["action","adventure"]}'
BODY='{"title":"The Breakfast Club","year":1986, "runtime":"96 mins","genres":["drama"]}'
# создать фильм в БД
curl -d "$BODY" -i localhost:4000/v1/movies

# вывести данные о фильме с id = 4 
curl -i localhost:4000/v1/movies/4

# UPDATE
BODY='{"title":"The Breakfast Club","year":1986, "runtime":"100 mins","genres":["drama"]}'
curl -X PUT localhost:4000/v1/movies/3

# DELETE
curl -X DELETE localhost:4000/v1/movies/3

#PATCH
curl -X PATCH -d '{"year": 1986}' localhost:4000/v1/movies/3

# множественные запросы к конечной точке, детектирование состояния гонки
xargs -I % -P8 curl -X PATCH -d '{"runtime": "97 mins"}' "localhost:4000/v1/movies/2" < <(printf '%s\n' {1..8})

# curl запрос с фиксацией времени выполнения запроса
curl -w '\nTime: %{time_total}s \n' localhost:4000/v1/movies/1

#список фильмов (строка парметров)
curl "localhost:4000/v1/movies?title=godfather&genres=crime,drama&page=18&page_size=5&sort=year"


# проверка ввалидности данных переданных в строке запроса
# невалидна строка
curl  "localhost:4000/v1/movies?page=-1&page_size=-1&sort=foo"

# List all movies.
/v1/movies

# add filters
# List movies where the title is a case-insensitive exact match for 'black panther'.  +  это пробел
 curl "localhost:4000/v1/movies?title=black+panther"# List movies where the genres includes 'adventure'.
 curl "localhost:4000/v1/movies?genres=adventure"
 
 curl "localhost:4000/v1/movies?title=moana&genres=animation,adventure"
 
 curl "localhost:4000/v1/movies?genres=western"  # не сущ-ие данные

 # полнотеекстовый поиск 
  curl "localhost:4000/v1/movies?title=panther"

# запросить список фильмов, отсортированных по убыванию названия, времени
 curl "localhost:4000/v1/movies?sort=-title"
  curl "localhost:4000/v1/movies?sort=-runtime"

# paginating list
#    Return the 5 records on page 1 (records 1-5 in the dataset)
/v1/movies?page=1&page_size=5
#  Return the next 5 records on page 2 (records 6-10 in the dataset)
/v1/movies?page=2&page_size=5
#  Return the next 5 records on page 3 (records 11-15 in the dataset)
/v1/movies?page=3&page_size=5
 curl "localhost:4000/v1/movies?page_size=2" # первые две записи
  curl "localhost:4000/v1/movies?page_size=2&page=2" #вторая страница

# метаданные
curl "localhost:4000/v1/movies?page_size=2&page=2"

# проверка ограничителя запросов (глобального)
 for i in {1..6}; do curl http://localhost:4000/v1/healthcheck; done
# проверка ограничителя запросов (глобального) + флаги запуска
  go run ./cmd/api/ -limiter-burst=2
  go run ./cmd/api/ -limiter-enabled=false

# корректное завершение работы программы
# посмотреть pid запущенного процесса
# Intercepting Shutdown Signals
 pgrep -l api
 pkill -SIGKILL api
 pkill -SIGTERM api
 pkill -SIGQUIT api

#проверить перехват сигналов и корректное завершение
#  Add a 4 second delay штещ : cmd/api/healthcheck.go. 
time.Sleep(4 * time.Second)
curl localhost:4000/v1/healthcheck & pkill -SIGTERM api #придет резудьтат и после завершитсч работа


# 
migrate create -seq -ext=.sql -dir=./migrations create_users_table
migrate -path=./migrations -database=$GREENLIGHT_DB_DSN up

go get golang.org/x/crypto/bcrypt@latest

# Регистрация нового пользователя: POST /v1/users registerUserHandler 
#  I. Создание структуры и вспомогательных методов.
#  II. Добавляем валидацию данных.
#  III. Создаем модель в БД.
#  IV. Создать хендлер и подключить в роутинг.
BODY='{"name": "Alice Smith", "email": "alice@example.com", "password": "pa55word"}'
curl -i -d "$BODY" localhost:4000/v1/users
BODY='{"name": "", "email": "bob@invalid.", "password": "pass"}'  
curl -i -d "$BODY" localhost:4000/v1/users

#  работаем с SMTP сервером - отправка прветственного сообщения(13)
mkdir -p internal/mailer/templates
touch internal/mailer/templates/user_welcome.tmpl
# "subject" - тема письма
# "plainBody" - текстовая часть 
# "htmlBody" - html -письма

go get github.com/wneessen/go-mail@latest


BODY='{"name": "Bob Jones", "email": "bob@example.com", "password": "pa55word"}'
curl -w '\nTime: %{time_total}\n' -i -d "$BODY" localhost:4000/v1/users


# активация user используя токен
migrate create -seq -ext .sql -dir ./migrations create_tokens_table
curl -X PUT -d '{"token":"invalid"}' localhost:4000/v1/users/activated

# создание пользователя, получение токена с сохранением состояния
BODY='{"name": "alex", "email": "alix@mail.com", "password": "pa55word"}'
curl -i -d "$BODY" localhost:4000/v1/users
BODY='{"email":"alix@mail.com", "password":"pa55word"}'
curl -i -d "$BODY" localhost:4000/v1/tokens/authentication

# запросить ресурс без токена аутентификации
curl localhost:4000/v1/healthcheck
# запросить токен по email и паролю
curl -w '\nTime: %{time_total}\n' -d '{"email":"alix@mail.com", "password":"pa55word"}' localhost:4000/v1/tokens/authentication
# запросить ресурс по токену  пользоватля
curl -w '\nTime: %{time_total}\n' -H "Authorization: Bearer <token>" localhost:4000/v1/healthcheck

# после ввода middleware для /v1/movies/* проверятся аутентифиция и активация пользователя

#разграничение доступа
migrate create -seq -ext .sql -dir ./migrations add_permissions


#для проверки доступности пользователю ресурса
#необходимоЖ пользователь аутентифицирован, активирован.
#подредактирем БД
# -- Set the activated field for alice@example.com to true.
UPDATE users SET activated = true WHERE email = 'alice@example.com';
# -- Give all users the 'movies:read' permission
INSERT INTO users_permissions
SELECT id, (SELECT id FROM permissions WHERE code = 'movies:read') FROM users;
# -- Give faith@example.com the 'movies:write' permission
INSERT INTO users_permissions
VALUES (
(SELECT id FROM users WHERE email = 'faith@example.com'),
(SELECT id FROM permissions WHERE code = 'movies:write')
);
# -- List all activated users and their permissions.
SELECT email, array_agg(permissions.code) as permissions
FROM permissions
INNER JOIN users_permissions ON users_permissions.permission_id = permissions.id
INNER JOIN users ON users_permissions.user_id = users.id
WHERE users.activated = true
GROUP BY email;
# запросить токен
curl -d "$BODY" localhost:4000/v1/tokens/authentication
{
"authentication_token": {
"token": "<token>",
"expiry": "2021-04-17T20:49:39.963768416+02:00"
}
#получить доступ
curl -H "Authorization: Bearer <token>" localhost:4000/v1/movies/1

curl -X DELETE -H "Authorization: Bearer <token>" localhost:4000/v1/movies/1

# даем пользователю права при создании
# выборка из БД
SELECT email, code FROM users
INNER JOIN users_permissions ON users.id = users_permissions.user_id
INNER JOIN permissions ON users_permissions.permission_id = permissions.id
WHERE users.email = 'grace@example.com';


# supporting multiple dynamic origins CORS - междоменные запросы
# add flag
go run ./cmd/api -cors-trusted-origins="https://www.example.com https://staging.example.com"

# 9000, 9001 - доверенные источники
go run ./cmd/api -cors-trusted-origins="http://localhost:9000 http://localhost:9001"

# Responding to preflight requests - реагирование на предварительный запрос.
# Предварительный запрос должен иметь 3 компонента: 
# the HTTP method OPTIONS, an Origin header, and an Access-Control-Request-Method header.
#  После определения отправить 200 и спец.заголовк для полуения реального запроса:
#     Access-Control-Allow-Origin: <reflected trusted origin>
#     Access-Control-Allow-Methods: OPTIONS, PUT, PATCH, DELETE
#     Access-Control-Allow-Headers: Authorization, Content-Type

# CORS-safe methods HEAD, GET or POST in the Access-Control-Allow-Methods.
# 4 CORS-safe headers:
#     Accept
#     Accept-Language
#     Content-Language
#     Content-Type
# The value for the Content-Type header (if set) is one of:
#     application/x-www-form-urlencoded
#     multipart/form-data
#     text/plain

#  Можно установить время кеширования предварительных ответов (preflight responses) Access-Control-Max-Age: 60
# отключить кеширование : -1


# Metrics app
# "cmdline" - праметры запуска приложения.
# "memstats" - снимок памяти в данный момент (https://pkg.go.dev/runtime#MemStats)
# TotalAlloc — общее количество байт, выделенных в куче (не уменьшается).
#  HeapAlloc — текущее количество байт в куче. 
#  HeapObjects — текущее количество объектов в куче. 
#  Sys — общее количество байт памяти, полученных от ОС 
#  (то есть общее количество памяти, зарезервированноые средой выполнения Go для кучи, стеков и других внутренних структур данных). 
#  NumGC — количество завершенных циклов сборки мусора. 
#  NextGC — целевой размер кучи для следующего цикла сборки мусора (Go стремится поддерживать HeapAlloc ≤ NextGC).

go run ./cmd/api -limiter-enabled=false -db-max-open-conns=50 -db-max-idle-conns=50 -db-max-idle-time=20s -port=4000
BODY='{"email":"max@example.com", "password":"pa55word"}'
#нагрузочное тестирование
hey -d "$BODY" -m "POST" http://localhost:4000/v1/tokens/authentication  
# вызовем конечную точку /debug/vars



# запустите API без ограничения скорости
go run ./cmd/api -limiter-enabled=false


# для проверки карты запросов. В этом случае, приложение возвратит /debug/vars "total_responses_sent_by_status": {"201": 4, "429": 196}
go run ./cmd/api
hey -d "$BODY" -m "POST" http://localhost:4000/v1/tokens/authentication
#TODO https://prometheus.io/


# Makefile
make
make up
make migration name=create_example_table
#make special variables https://www.gnu.org/software/make/manual/html_node/Special-Variables.html
make help
# phony target https://www.gnu.org/software/make/manual/html_node/Phony-Targets.html#Phony-Targets
# https://www.gnu.org/software/make/manual/html_node/Special-Targets.html#Special-Targets
make run/api
#direnv https://direnv.net/ или include
echo '.envrc' >> .gitignore

# контроль качества
`go mod tidy -diff` проверяет что в go.mod and go.sum содержатся все зависимости
`go mod verify ` проверяет, что зависимости в $GOPATH/pkg/mod соответсвуют конт.суммам в go.sum
`go mod vendor` скопирует исх.код из кеша модуля в директорию vendor проекта

`go vet ./...` выполняет стат анализ (https://golang.org/cmd/vet/)
staticcheck (https://staticcheck.io/)  https://staticcheck.io/docs/checks
`go test -race -vet=off ./...` запуск тестов с детектором гонки и отключенным vet.
`go mod tidy` удаление/добавление зависимостей
`go fmt ./...` форматирование кода в соотв-ии со стандартом

go get -tool honnef.co/go/tools/cmd/staticcheck@latest   
go tool staticcheck --version

go tool (https://www.alexedwards.net/blog/how-to-manage-tool-dependencies-in-go-1.24-plus)
staticcheck для статического анализа кода
govulncheck сканирования уязвимостей `go get -tool golang.org/x/vuln/cmd/govulncheck` (https://pkg.go.dev/golang.org/x/vuln/cmd/govulncheck)
air для перезагрузки приложений в режиме реального времени (https://github.com/air-verse/air)
`go get -tool golang.org/x/tools/cmd/stringer`

# TODO tests
# - Creating an end-to-end test for the GET /v1/healthcheck endpoint to verify that the
# headers and response body are what you expect.
# - Creating a unit-test for the rateLimit() middleware to confirm that it sends a
# 429 Too Many Requests response after a certain number of requests.
# - Creating an end-to-end integration test, using a test database instance, which confirms
# that the authenticate() and requirePermission() middleware work together correctly
# to allow or disallow access to specific endpoints.



go env
# GOPROXY - содержит зеркало для скачивания пакета
# преопределить:
# export GOPROXY=https://goproxy.io,https://proxy.golang.org,direct
# возможные зеркала:
#  https://goproxy.io
#  https://athens.azurefd.net

go mod vendor  добавление зависимостей в проект

`go clean -modcache` - удалит все из локального кеша зависимостей

# уменьшить размер исполняемого файла можно удалив symbol tables and DWARF debugging information
# c момощью флага ldflaf (!затруднит отладку с пом.  Delve or gdb)
go build -ldflags='-s' -o=./bin/api ./cmd/api
# удаление только отладочной таблицы -ldflags='-s -w=0'


# список поддерживаемых архитектур:
go tool dist list
#  задать ОС и архитектуру
GOOS=linux GOARCH=amd64 go build {args}
echo 'bin/' >> .gitignore

# go env GOCACHE где находится кеш сборки
go build -a -o=/bin/foo ./cmd/foo # Force all packages to be rebuilt
go clean -cache # Remove everything from the build cache


# version
 make build/api
 go version -m ./bin/api покажет инфо о бин файле
# vsc инфо о системе контроля версий
# mod ... v0.0.0.... псевдоверсия автом-ки сгененрированная Go https://go.dev/ref/mod#pseudo-versions
#  debug.BuildInfo - содержит инфо что и go version -m (https://pkg.go.dev/runtime/debug#BuildInfo)
# версия не встраивается при запуске go run и go test без -buildvcs=true. Version: (devel)
 git tag v1.0.0
 # про коммиты и версии https://go.dev/ref/mod#pseudo-versions


  ssh-keygen -t rsa -b 4096 -C "greenlight@greenlight.alexedwards.net" -f $HOME/.ssh/id_rsa_greenlight
   ssh-add -l # отобразить ключ
    ssh-add $HOME/.ssh/id_rsa_greenlight   # добавить ключ в ssh-agent
    
# запросить токен для сброса пароля
 curl -X POST -d '{"email": "alice@example.com"}' localhost:4000/v1/tokens/password-reset

# инициировать смену пароля, отправив токен сброса из письма 
BODY='{"password": "your new password", "token": "Y7QC"}'
curl -X PUT -d "$BODY" localhost:4000/v1/users/password
