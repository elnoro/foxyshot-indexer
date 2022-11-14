FROM golang:1.19-alpine as builder

ENV CGO_ENABLED=0

WORKDIR /app

COPY go.mod .
COPY go.sum .
RUN go mod download

COPY --from=migrate/migrate:4 /usr/local/bin/migrate /service/migrate

COPY . .
RUN go build -o /service/indexer ./cmd/indexer/main.go
RUN cp -R migrations /service

FROM alpine:3.16

RUN apk update && apk add tesseract-ocr
COPY --from=builder /service /service
WORKDIR /service

ENTRYPOINT ["/service/indexer"]