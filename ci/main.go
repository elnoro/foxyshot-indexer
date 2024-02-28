package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"dagger.io/dagger"
)

func main() {
	if err := build(context.Background()); err != nil {
		fmt.Println(err)
	}
}

func build(ctx context.Context) error {
	client, err := dagger.Connect(ctx, dagger.WithLogOutput(os.Stderr))
	if err != nil {
		return err
	}
	defer func(client *dagger.Client) {
		err := client.Close()
		if err != nil {
			log.Println("dagger client closing error:", err)
		}
	}(client)

	src := client.Host().Directory(".")
	contextDir := client.Host().Directory(".")
	runner := client.Container().
		Build(contextDir, dagger.ContainerBuildOpts{Dockerfile: "dev.Dockerfile"}).
		WithEntrypoint([]string{"go"}).
		WithDirectory(
			"/src",
			src,
			dagger.ContainerWithDirectoryOpts{Exclude: []string{"ci/", "build/", "tmp/"}},
		).
		WithWorkdir("/src")

	// 1. make check/mod
	out, err := runner.
		WithExec([]string{"mod", "tidy"}).
		WithExec([]string{"mod", "verify"}).
		Stderr(ctx)
	if err != nil {
		return fmt.Errorf("running check/mod, %w", err)
	}
	fmt.Println(out)

	// 2. make check/lint
	golangci := client.Container().
		From("golangci/golangci-lint:v1.55.2-alpine").
		WithDirectory(
			"/app",
			src,
			dagger.ContainerWithDirectoryOpts{Exclude: []string{"ci/", "build/", "tmp/"}},
		).
		WithWorkdir("/app")

	out, err = golangci.
		WithExec([]string{"golangci-lint", "run", "-v"}).
		Stderr(ctx)
	if err != nil {
		return fmt.Errorf("running check/lint, %w", err)
	}
	fmt.Println(out)

	// test dependencies - db, migrations, s3 - replacement for docker-compose-dev.yml
	minio := client.Container().
		From("bitnami/minio").
		WithEnvVariable("MINIO_ROOT_USER", "minio-access-key").
		WithEnvVariable("MINIO_ROOT_PASSWORD", "minio-secret-key").
		WithEnvVariable("MINIO_DEFAULT_BUCKETS", "bucket:public").
		WithExposedPort(9000)

	const testDSN = "postgres://user:pass@db/db"
	postgresContextDir := client.Host().Directory("./docker/postgres")
	postgres := client.Container().
		Build(postgresContextDir).
		WithEnvVariable("POSTGRES_PASSWORD", "pass").
		WithEnvVariable("POSTGRES_USER", "user").
		WithEnvVariable("POSTGRES_DB", "db").
		WithEnvVariable("PGUSER", "user").
		WithEnvVariable("PGDATABASE", "db").
		WithExposedPort(5432)

	migrations := client.Host().Directory("./migrations")
	out, err = client.Container().
		WithServiceBinding("db", postgres.AsService()).
		From("migrate/migrate:4").
		WithDirectory("/migrations", migrations).
		WithEnvVariable("CACHEBUSTER", time.Now().String()).
		WithExec([]string{
			"-database", testDSN + "?sslmode=disable", "-path=/migrations", "up",
		}).
		Stdout(ctx)
	if err != nil {
		return fmt.Errorf("running migrations, %w", err)
	}
	fmt.Println(out)

	// make check/test
	out, err = runner.
		// passing postgres container to the test runner
		WithServiceBinding("db", postgres.AsService()).
		WithEnvVariable("TEST_DSN", testDSN).
		// passing minio container to the test runner
		WithServiceBinding("minio", minio.AsService()).
		WithEnvVariable("S3_ENDPOINT", "minio:9000").
		WithEnvVariable("S3_KEY", "minio-access-key").
		WithEnvVariable("S3_SECRET", "minio-secret-key").
		WithEnvVariable("S3_BUCKET", "bucket").
		WithEnvVariable("S3_PUBLIC", "minio").
		// installing test dependencies
		WithExec([]string{"generate", "./..."}).
		WithExec([]string{"test", "-race", "-vet=off", "./..."}).
		Stderr(ctx)
	if err != nil {
		return fmt.Errorf("running check/test, %w", err)
	}
	fmt.Println(out)

	return nil
}
