package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/elnoro/foxyshot-indexer/internal/captioning"
	"github.com/elnoro/foxyshot-indexer/internal/captioning/ollama"
	"github.com/elnoro/foxyshot-indexer/internal/embedding"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/elnoro/foxyshot-indexer/internal/app"
	dbadapter "github.com/elnoro/foxyshot-indexer/internal/db"
	"github.com/elnoro/foxyshot-indexer/internal/indexer"
	"github.com/elnoro/foxyshot-indexer/internal/monitoring"
	"github.com/go-playground/validator/v10"
	_ "github.com/jackc/pgx/stdlib"
	"github.com/jmoiron/sqlx"

	"github.com/elnoro/foxyshot-indexer/internal/ocr"
	"github.com/elnoro/foxyshot-indexer/internal/s3wrapper"
)

type Config struct {
	Port           int           `validate:"required"`
	ScrapeInterval time.Duration `validate:"required"`
	Ext            string        `validate:"required"`
	DSN            string        `validate:"required"`
	S3             S3Config      `validate:"required"`
	Caption        Caption       `validate:"required"`
	EmbeddingsURL  string        ``
}

type S3Config struct {
	Key           string `validate:"required"`
	Secret        string `validate:"required"`
	Endpoint      string `validate:"required"`
	Region        string `validate:"required"`
	Bucket        string `validate:"required"`
	Insecure      bool
	RetryAttempts int
	RetryDuration time.Duration
}

type Caption struct {
	OllamaURL string
}

var version = "development"

func main() {
	cfg := Config{}

	flag.IntVar(&cfg.Port, "web.port", 8080, "API server port")

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

	flag.StringVar(&cfg.Caption.OllamaURL, "caption.ollamaUrl", os.Getenv("OLLAMA_URL"), "url pointing to ollama API")
	flag.StringVar(&cfg.EmbeddingsURL, "embeddings.url", os.Getenv("EMBEDDINGS_URL"), "url pointing to embeddings API")

	flag.Parse()

	err := validateConfig(cfg)
	if err != nil {
		log.Fatal(err)
	}

	logger := slog.Default()

	storage, err := s3wrapper.NewFromSecrets(
		cfg.S3.Key,
		cfg.S3.Secret,
		cfg.S3.Endpoint,
		cfg.S3.Region,
		cfg.S3.Bucket,
		cfg.S3.Insecure,
		logger,
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

	captionSmith := buildCaptionSmith(cfg, logger)

	embeddingsClient := buildEmbeddingsClient(cfg, logger)

	tracker := monitoring.NewTracker()
	err = tracker.Register()
	if err != nil {
		log.Fatal(err)
	}

	db, err := sqlx.Connect("pgx", cfg.DSN)
	if err != nil {
		log.Fatal("sqlx error", err)
	}
	defer func(db *sqlx.DB) {
		err := db.Close()
		if err != nil {
			log.Println("database closing error,", err)
		}
	}(db)

	imgRepo := dbadapter.NewImageRepo(db)

	idxr := indexer.NewIndexer(imgRepo, storage, ocrEngine, captionSmith, embeddingsClient, logger, tracker)
	runner := app.NewIndexRunner(idxr, cfg.Ext, cfg.ScrapeInterval, logger)

	ctx, cancel := context.WithCancel(context.Background())

	wg := sync.WaitGroup{}

	web := &webApp{
		config: cfg,
		log:    log.Default(),

		imageDescriptions: imgRepo,
		embeddings:        embeddingsClient,
		fileStorage:       storage,
		tracker:           tracker,
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		err := web.serve(ctx)
		if err != nil {
			log.Println("web server error:", err)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		err := runner.Start(ctx)
		if err != nil {
			log.Println("index runner error:", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	s := <-quit

	fmt.Printf("Received %s signal\n", s.String())
	cancel()
	wg.Wait()
}

func validateConfig(cfg Config) error {
	validate := validator.New()
	return validate.Struct(cfg)
}

func buildCaptionSmith(cfg Config, l *slog.Logger) indexer.CaptionSmith {
	if cfg.Caption.OllamaURL == "" {
		l.Info("Caption service URL is not provided, running without captions...")
		return captioning.NewPlaceholder("")
	}

	ollamaClient, err := ollama.NewClient(cfg.Caption.OllamaURL, l)
	if err != nil {
		log.Fatal(err)
	}

	return ollamaClient
}

func buildEmbeddingsClient(cfg Config, l *slog.Logger) indexer.ImageEmbeddingClient {
	if cfg.EmbeddingsURL != "" {
		return embedding.NewClient(cfg.EmbeddingsURL, l)
	}

	l.Info("Embeddings service URL is not provided, running without embeddings...")

	return &embedding.NullClient{}
}
