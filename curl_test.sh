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

migrate create -seq -ext .sql -dir ./migrations create_tokens_table

BODY='{"name": "Bob Jones", "email": "bob@example.com", "password": "pa55word"}'
curl -w '\nTime: %{time_total}\n'-i -d "$BODY" localhost:4000/v1/users
