package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"os"
	"time"

	dbadapter "github.com/elnoro/foxyshot-indexer/internal/db"
	"github.com/elnoro/foxyshot-indexer/internal/indexer"
	"github.com/go-playground/validator/v10"
	_ "github.com/jackc/pgx/stdlib"
	"github.com/jmoiron/sqlx"

	"github.com/elnoro/foxyshot-indexer/internal/ocr"
	"github.com/elnoro/foxyshot-indexer/internal/s3wrapper"
)

type Config struct {
	ScrapeInterval time.Duration `validate:"required"`
	Ext            string        `validate:"required"`
	DSN            string        `validate:"required"`
	S3             S3Config      `validate:"required"`
}

type S3Config struct {
	Key           string `validate:"required"`
	Secret        string `validate:"required"`
	Endpoint      string `validate:"required"`
	Region        string `validate:"required"`
	Bucket        string `validate:"required"`
	Insecure      bool   `validate:"required"`
	RetryAttempts int
	RetryDuration time.Duration
}

func main() {
	cfg := Config{}

	flag.DurationVar(&cfg.ScrapeInterval, "scrape.interval", 15*time.Minute, "how often to scrape s3")
	flag.StringVar(&cfg.Ext, "ext", ".jpg", "file extensions to use")
	flag.StringVar(&cfg.DSN, "dsn", os.Getenv("DB_DSN"), "connection string for the database")

	flag.StringVar(&cfg.S3.Key, "s3.key", os.Getenv("S3_KEY"), "s3 key")
	flag.StringVar(&cfg.S3.Secret, "s3.secret", os.Getenv("S3_SECRET"), "s3 secret")
	flag.StringVar(&cfg.S3.Endpoint, "s3.endpoint", os.Getenv("S3_ENDPOINT"), "s3 endpoint")
	flag.StringVar(&cfg.S3.Region, "s3.region", "eu-west1", "s3 region")
	flag.StringVar(&cfg.S3.Bucket, "s3.bucket", os.Getenv("S3_BUCKET"), "s3 bucket")
	flag.BoolVar(&cfg.S3.Insecure, "s3.insecure", false, "disable ssl. For testing purposes only!")

	flag.IntVar(&cfg.S3.RetryAttempts, "s3.attempts", 0, "how many times to check s3 connectivity during startup")
	flag.DurationVar(&cfg.S3.RetryDuration, "s3.retry", 15*time.Second, "retry duration between attempts")
	flag.Parse()

	err := validateConfig(cfg)
	if err != nil {
		log.Fatal(err)
	}

	storage, err := s3wrapper.NewFromSecrets(
		cfg.S3.Key,
		cfg.S3.Secret,
		cfg.S3.Endpoint,
		cfg.S3.Region,
		cfg.S3.Bucket,
		cfg.S3.Insecure,
	)
	if err != nil {
		log.Fatal(err)
	}
	if cfg.S3.RetryAttempts > 0 {
		err := storage.CheckConnectivity(cfg.S3.RetryAttempts, cfg.S3.RetryDuration)
		if err != nil {
			log.Fatal(err)
		}
	}

	ocrEngine, err := ocr.Default()
	if err != nil {
		log.Fatal(err)
	}

	db, err := sqlx.Connect("pgx", cfg.DSN)
	if err != nil {
		log.Fatal("sqlx error", err)
	}

	imgRepo := dbadapter.NewImageRepo(db)

	i := indexer.NewIndexer(imgRepo, storage, ocrEngine)

	ctx := context.Background()
	for {
		lastModified, err := imgRepo.GetLastModified(context.Background())
		if err != nil {
			log.Fatal(err) // if there is something wrong with the db we fail the app and let a supervisor restart it
		}
		files, err := storage.ListFiles(lastModified, cfg.Ext)
		if err != nil {
			log.Fatal(err)
		}

		for _, file := range files {
			_, err := imgRepo.Get(ctx, file.Key)
			if err != nil && !errors.Is(err, dbadapter.ErrRecordNotFound) {
				log.Println("ERROR: failed to check:", err)
				continue
			}
			if nil == err {
				log.Printf("INFO: %s already processed, skipping\n", file.Key)
				continue
			}

			err = i.Index(file)
			if err != nil {
				log.Println("ERROR: failed to index:", err)
			} else {
				log.Println("added", file)
			}
		}

		time.Sleep(cfg.ScrapeInterval)
	}
}

func validateConfig(cfg Config) error {
	validate := validator.New()
	return validate.Struct(cfg)
}
