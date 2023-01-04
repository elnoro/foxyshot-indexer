migrate:
	migrate -database="${DB_DSN}?sslmode=disable" -path=./migrations up
.PHONY: migrate

compose/dev/sh:
	docker compose -f docker-compose-dev.yml exec app /bin/sh
.PHONY: compose/dev/sh

compose/dev/up:
	docker compose -f docker-compose-dev.yml up -d
.PHONY: compose/dev/up

compose/dev/down:
	docker compose -f docker-compose-dev.yml down
.PHONY: compose/dev/down

check/mod:
	go mod tidy
	go mod verify
.PHONY: check/mod