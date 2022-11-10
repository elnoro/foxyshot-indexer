package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"time"

	"foxyshot-indexer/internal/s3wrapper"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	_ "github.com/jackc/pgx/stdlib"
	"github.com/jmoiron/sqlx"
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

type ImageDescription struct {
	FileID      string `db:"file_id"`
	Description string `db:"description"`
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

	db, err := sqlx.Connect("pgx", cfg.DSN)
	if err != nil {
		log.Fatal(err)
	}

	i := Indexer{db: db, storage: storage}

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

type Indexer struct {
	db      *sqlx.DB
	storage *s3wrapper.BucketClient
}

func (i *Indexer) Index(file string) error {
	f, err := i.storage.Download(file)
	if f != nil {
		defer func(name string) {
			err := os.Remove(name)
			if err != nil {
				log.Println("ERROR: removing temp file,", err)
			}
		}(f.Name())
	}
	if err != nil {
		return fmt.Errorf("cannot download file, %w")
	}
	ocr, err := RunOCR(f.Name())
	if err != nil {
		return fmt.Errorf("running ocr, %w", err)
	}

	id := &ImageDescription{
		FileID:      file,
		Description: ocr,
	}

	_, err = i.db.NamedExec(
		"INSERT INTO image_descriptions (file_id, description) VALUES (:file_id, :description)", id,
	)
	if err != nil {
		return fmt.Errorf("db insert %w", err)
	}

	return nil
}

func RunOCR(file string) (string, error) {
	descFileID := uuid.New().String()
	descFile := descFileID + ".txt"
	defer func(name string) {
		err := os.Remove(name)
		if err != nil {
			log.Println(err)
		}
	}(descFile)
	cmd := exec.Command("tesseract", file, descFileID)

	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Println(string(out))
		return "", fmt.Errorf("running tesseract, %w", err)
	}

	contents, err := os.ReadFile(descFile)
	if err != nil {
		return "", fmt.Errorf("reading tesseract output, %w", err)
	}

	return string(contents), nil
}
