migrate:
	migrate -database="${DB_DSN}?sslmode=disable" -path=./migrations up
.PHONY: migrate