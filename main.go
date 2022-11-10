package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
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

	client := NewS3Client(cfg.S3)

	fp := &FilesProvider{
		client:     client,
		publicAddr: cfg.S3.PublicAddr,
		suffix:     cfg.Ext,
		bucket:     cfg.S3.Bucket,
	}

	db, err := sqlx.Connect("pgx", cfg.DSN)
	if err != nil {
		log.Fatal(err)
	}

	i := Indexer{db: db}

	files, err := fp.GetFiles(time.Unix(0, 0))
	if err != nil {
		log.Fatal(err)
	}

	for _, file := range files {
		err := i.Index(file)
		if err != nil {
			log.Println("failed to index:", err)
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

type FilesProvider struct {
	client     *s3.S3
	publicAddr string
	suffix     string
	bucket     string
}

func (f *FilesProvider) GetFiles(start time.Time) ([]string, error) {
	listObjsResponse, err := f.client.ListObjectsV2(&s3.ListObjectsV2Input{Bucket: aws.String(f.bucket)})
	if err != nil {
		return nil, fmt.Errorf("listing objects from bucket %s, %w", f.bucket, err)
	}

	var files []string
	for _, object := range listObjsResponse.Contents {
		if object.LastModified.Before(start) {
			continue
		}
		if !strings.HasSuffix(*object.Key, f.suffix) {
			continue
		}

		files = append(files, f.publicAddr+"/"+*object.Key)
	}

	return files, nil
}

type Indexer struct {
	db *sqlx.DB
}

func (i *Indexer) Index(file string) error {
	ocr, err := RunOCR(file)
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

func NewS3Client(config S3Config) *s3.S3 {
	s3Config := &aws.Config{
		S3ForcePathStyle: aws.Bool(true),
		Credentials:      credentials.NewStaticCredentials(config.Key, config.Secret, ""),
		Endpoint:         aws.String(config.Endpoint),
		Region:           aws.String(config.Region),
	}

	newSession, err := session.NewSession(s3Config)
	if err != nil {
		log.Fatalf("Cannot connect to storage, got %v", err)
	}
	s3Client := s3.New(newSession)

	return s3Client
}
