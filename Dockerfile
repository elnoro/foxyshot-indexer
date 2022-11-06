FROM golang:1.19-alpine AS build

WORKDIR /go/src/indexer
COPY . .
RUN CGO_ENABLED=0 go build -o /app/indexer

FROM ubuntu:22.10

RUN apt update && apt install -y tesseract-ocr
COPY --from=build /app /

ENTRYPOINT "/indexer -dsn=$DSN -s3.key=$S3_KEY -s3.secret=$S3_SECRET -s3.bucket=$BUCKET -s3.endpoint=$S3_ENDPOINT -s3.public=$S3_PUBLIC"