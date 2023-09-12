.PHONY: confirm
confirm:
	@echo -n 'Are you sure? [y/N] ' && read ans && [ $${ans:-N} = y ]

compose/dev/sh:
	docker compose -f docker-compose-dev.yml exec app /bin/sh
.PHONY: compose/dev/sh

compose/dev/rebuild-image:
	docker compose -f docker-compose-dev.yml build app
.PHONY: compose/dev/up

compose/dev/up:
	docker compose -f docker-compose-dev.yml up -d

	@echo "Open http://localhost:8080/healthcheck"
.PHONY: compose/dev/up

compose/dev/down:
	docker compose -f docker-compose-dev.yml down
.PHONY: compose/dev/down

check/mod:
	go mod tidy
	go mod verify
.PHONY: check/mod

check/lint:
	docker run --rm -v `pwd`:/app -w /app golangci/golangci-lint:v1.50.1 golangci-lint run -v
.PHONY: check/lint

check/test:
	docker compose -f docker-compose-dev.yml exec app go generate ./...
	docker compose -f docker-compose-dev.yml exec app go test -race -vet=off ./...
.PHONY: check/test

check/all: check/mod check/lint check/test

check/dagger:
	dagger run go run ci/main.go

migrate/run: confirm
	docker compose -f docker-compose-dev.yml exec app \
		sh -c 'migrate -database="$${DB_DSN}?sslmode=disable" -path=./migrations up'
.PHONY: migrate/run

publish/docker:
	docker buildx build --push --platform linux/amd64,linux/arm64 -t ghcr.io/elnoro/indexer .