confirm:
	@echo -n 'Are you sure? [y/N] ' && read ans && [ $${ans:-N} = y ]
.PHONY: confirm

compose/dev/sh:
	docker compose -f docker-compose-dev.yml exec app /bin/sh
.PHONY: compose/dev/sh

compose/dev/rebuild-image:
	docker compose -f docker-compose-dev.yml build app
.PHONY: compose/dev/up

compose/dev/up:
	docker compose -f docker-compose-dev.yml up -d

	@echo "API http://localhost:8080/healthcheck"
	@echo "Minio http://localhost:9001/"
.PHONY: compose/dev/up

compose/dev/down:
	docker compose -f docker-compose-dev.yml down
.PHONY: compose/dev/down

check/mod:
	go mod tidy
	go mod verify
.PHONY: check/mod

check/lint:
	docker run --rm -v `pwd`:/app -w /app golangci/golangci-lint:v1.55.2-alpine golangci-lint run -v
.PHONY: check/lint

check/test:
	docker compose -f docker-compose-dev.yml exec app go generate ./...
	docker compose -f docker-compose-dev.yml exec app go test -race -vet=off ./...
.PHONY: check/test

check/all: check/mod check/lint check/test
.PHONY: check/all

check/dagger:
	dagger run go run ci/main.go
.PHONY: check/dagger

migrate/run: confirm
	docker compose -f docker-compose-dev.yml exec app \
		sh -c 'migrate -database="$${DB_DSN}?sslmode=disable" -path=./migrations up'
.PHONY: migrate/run

publish/docker:
	docker buildx build --push --platform linux/amd64,linux/arm64 -t ghcr.io/elnoro/indexer .
.PHONY: publish/docker
