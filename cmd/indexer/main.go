package main

import (
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
	ImgDir string
	Ext    string
	DSN    string `validate:"required"`
	S3     S3Config
}

type S3Config struct {
	Key        string `validate:"required"`
	Secret     string `validate:"required"`
	Endpoint   string `validate:"required"`
	Region     string `validate:"required"`
	Bucket     string `validate:"required"`
	PublicAddr string `validate:"required"`
}

func main() {
	cfg := Config{}

	flag.StringVar(&cfg.ImgDir, "dir", "imgdata", "path to the folder with the images")
	flag.StringVar(&cfg.Ext, "ext", ".jpg", "file extensions to use")
	flag.StringVar(&cfg.DSN, "dsn", os.Getenv("DB_DSN"), "connection string for the database")

	flag.StringVar(&cfg.S3.Key, "s3.key", os.Getenv("S3_KEY"), "s3 key")
	flag.StringVar(&cfg.S3.Secret, "s3.secret", os.Getenv("S3_SECRET"), "s3 secret")
	flag.StringVar(&cfg.S3.Endpoint, "s3.endpoint", os.Getenv("S3_ENDPOINT"), "s3 endpoint")
	flag.StringVar(&cfg.S3.Region, "s3.region", "eu-west1", "s3 region")
	flag.StringVar(&cfg.S3.Bucket, "s3.bucket", os.Getenv("S3_BUCKET"), "s3 bucket")
	flag.StringVar(&cfg.S3.PublicAddr, "s3.public", os.Getenv("S3_PUBLIC"), "public address to which images will be attached")
	flag.Parse()

	err := validateConfig(cfg)
	if err != nil {
		log.Fatal(err)
	}

	storage, err := s3wrapper.NewFromSecrets(cfg.S3.Key, cfg.S3.Secret, cfg.S3.Endpoint, cfg.S3.Region, cfg.S3.Bucket)
	if err != nil {
		log.Fatal(err)
	}

	ocrEngine, err := ocr.Default()
	if err != nil {
		log.Fatal(err)
	}

	db, err := sqlx.Connect("pgx", cfg.DSN)
	if err != nil {
		log.Fatal(err)
	}

	imgRepo := dbadapter.NewImageRepo(db)

	i := indexer.NewIndexer(imgRepo, storage, ocrEngine)

	files, err := storage.ListFiles(time.Unix(0, 0), cfg.Ext)
	if err != nil {
		log.Fatal(err)
	}

	for _, file := range files {
		err := i.Index(file)
		if err != nil {
			log.Println("ERROR: failed to index:", err)
		} else {
			log.Println("added", file)
		}
		return
	}
}

func validateConfig(cfg Config) error {
	validate := validator.New()
	return validate.Struct(cfg)
}
