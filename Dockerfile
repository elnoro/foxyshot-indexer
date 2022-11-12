FROM golang:1.19-alpine

COPY --from=migrate/migrate:4 /usr/local/bin/migrate /usr/local/bin/migrate
RUN apk update && apk add tesseract-ocr
RUN go install github.com/matryer/moq@latest
RUN go install github.com/go-delve/delve/cmd/dlv@latest
ENV CGO_ENABLED=0

WORKDIR /app

COPY go.mod .
COPY go.sum .
RUN go mod download

ENTRYPOINT ["sleep", "3600"]