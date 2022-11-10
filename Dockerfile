FROM golang:1.19-alpine

RUN apk update && apk add tesseract-ocr
RUN go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@v4.15.2

WORKDIR /go/src/indexer
COPY . .
RUN CGO_ENABLED=0 go build -o /app/indexer
WORKDIR /app

ENTRYPOINT ["sleep", "600"]