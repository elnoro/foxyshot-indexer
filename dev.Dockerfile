FROM golang:1.20-alpine as builder

COPY --from=migrate/migrate:4 /usr/local/bin/migrate /usr/local/bin/migrate
RUN apk update && apk add tesseract-ocr
RUN go install github.com/matryer/moq@latest
RUN go install github.com/go-delve/delve/cmd/dlv@latest
RUN go install github.com/cosmtrek/air@latest

ENV CGO_ENABLED=0

WORKDIR /app

COPY go.mod .
COPY go.sum .
RUN go mod download

ENTRYPOINT ["air", "--", "-s3.insecure", "-scrape.interval=1m", "-s3.attempts=3" ]