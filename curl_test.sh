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