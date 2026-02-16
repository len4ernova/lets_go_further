# Create the new confirm target.
confirm:
	@echo -n 'Are you sure? (y/n) ' && read ans && [ $${ans:-N} = y ]
run/api:
	go run ./cmd/api

db/sql:
	psql ${GREENLIGHT_DB_DSN}

db/migrations/new:
	@echo 'Creating migration files for ${name}'
	migrate create -seq -ext=.sql -dir=./migrations ${name}

db/migrations/up: confirm
	@echo '@Running up migratioons ...'
	migrate -path=./migrations -database=${GREENLIGHT_DB_DSN} up
