FROM golang:1.20-alpine

COPY --from=migrate/migrate:4 /usr/local/bin/migrate /usr/local/bin/migrate
RUN apk update && apk add tesseract-ocr tesseract-ocr-data-eng gcc musl-dev
RUN go install github.com/matryer/moq@latest
RUN go install github.com/cosmtrek/air@latest

WORKDIR /app

COPY go.mod .
COPY go.sum .
RUN go mod download

ENTRYPOINT ["air", "--", "-s3.insecure", "-scrape.interval=1m", "-s3.attempts=3" ]