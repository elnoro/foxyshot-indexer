package db

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/elnoro/foxyshot-indexer/internal/domain"
	_ "github.com/jackc/pgx/stdlib"
	"github.com/jmoiron/sqlx"
	"github.com/matryer/is"
)

const testFileID = "expected-file-id"

func TestImageRepo(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	ctx := context.Background()
	testDB := newTestDB(t)

	repo := NewImageRepo(testDB)

	t.Run("Upsert replaces description on when the same file id is used", func(t *testing.T) {
		tt := is.New(t)

		err := repo.Upsert(ctx, domain.Image{
			FileID:      testFileID,
			Description: "original-description",
		})
		tt.NoErr(err)

		gotOrig, err := repo.Get(ctx, testFileID)
		tt.NoErr(err)
		tt.Equal(gotOrig.Description, "original-description")

		err = repo.Upsert(context.Background(), domain.Image{
			FileID:      testFileID,
			Description: "updated-description",
		})
		tt.NoErr(err)
		gotUpdated, err := repo.Get(ctx, testFileID)
		tt.NoErr(err)
		tt.Equal(gotUpdated.Description, "updated-description")
	})

	t.Run("Upsert adds file id to an error in case of query failure", func(t *testing.T) {
		tt := is.New(t)

		cancelledContext, cancel := context.WithCancel(ctx)
		cancel() // cancelling immediately to induce error

		err := repo.Upsert(cancelledContext, domain.Image{
			FileID: testFileID,
		})

		tt.True(strings.Contains(err.Error(), testFileID))
		tt.True(errors.Is(err, context.Canceled))
	})

	t.Run("GetLastModified returns zero value if no images were added", func(t *testing.T) {
		tt := is.New(t)

		want := time.Unix(32530952912, 0).UTC()
		img := &domain.Image{FileID: "file-id", LastModified: want}
		err := repo.Upsert(ctx, *img)
		tt.NoErr(err)

		got, err := repo.GetLastModified(ctx)
		tt.NoErr(err)
		tt.Equal(want, got)

		want = want.Add(1 * time.Second)
		img.LastModified = want
		err = repo.Upsert(ctx, *img)
		tt.NoErr(err)

		got, err = repo.GetLastModified(ctx)
		tt.NoErr(err)
		tt.Equal(want, got)
	})

	t.Run("GetLastModified returns the latest last modified from all images", func(t *testing.T) {
		tt := is.New(t)

		want := time.Unix(32530952912, 0).UTC()
		img := &domain.Image{FileID: "file-id", LastModified: want}
		err := repo.Upsert(ctx, *img)
		tt.NoErr(err)

		got, err := repo.GetLastModified(ctx)
		tt.NoErr(err)
		tt.Equal(want, got)

		want = want.Add(1 * time.Second)
		img.LastModified = want
		err = repo.Upsert(ctx, *img)
		tt.NoErr(err)

		got, err = repo.GetLastModified(ctx)
		tt.NoErr(err)
		tt.Equal(want, got)
	})

	t.Run("GetLastModified returns wrapped error if there is an error in the query", func(t *testing.T) {
		tt := is.New(t)

		ctx, cancel := context.WithCancel(ctx)
		cancel() // inducing error

		_, err := repo.GetLastModified(ctx)

		tt.True(errors.Is(err, context.Canceled))
	})

	t.Run("GetLastModified returns start of Unix time if there is nothing to find", func(t *testing.T) {
		tt := is.New(t)

		query := `truncate image_descriptions`
		_, err := testDB.Exec(query)
		tt.NoErr(err)

		want := time.Unix(0, 0).UTC()
		got, err := repo.GetLastModified(ctx)

		tt.NoErr(err)
		tt.Equal(want, got)
	})

	t.Run("Search returns images with descriptions matching search string", func(t *testing.T) {
		tt := is.New(t)

		err := repo.Upsert(ctx, domain.Image{
			FileID:      "expected-found-id",
			Description: "find me",
		})
		tt.NoErr(err)

		err = repo.Upsert(ctx, domain.Image{
			FileID:      "expected-not-found-id",
			Description: "skip me",
		})
		tt.NoErr(err)

		images, err := repo.Search(ctx, "find me", 1, 100)
		tt.NoErr(err)

		tt.Equal(1, len(images))
		tt.Equal("expected-found-id", images[0].FileID)
	})

	t.Run("Search returns error if there is an error in the query", func(t *testing.T) {
		tt := is.New(t)

		ctx, cancel := context.WithCancel(ctx)
		cancel()

		images, err := repo.Search(ctx, "find me", 1, 100)
		tt.Equal(0, len(images))
		tt.True(errors.Is(err, context.Canceled))

	})

	t.Run("Get returns wrapped error if there is an error in the query", func(t *testing.T) {
		tt := is.New(t)

		ctx, cancel := context.WithCancel(ctx)
		cancel() // inducing error

		_, err := repo.Get(ctx, testFileID)

		tt.True(errors.Is(err, context.Canceled))
	})

	t.Run("Get returns not found error if fileID does not exist", func(t *testing.T) {
		tt := is.New(t)

		_, err := repo.Get(ctx, "does-not-exist")

		tt.True(errors.Is(err, ErrRecordNotFound))
	})
}

func newTestDB(t *testing.T) *sqlx.DB {
	t.Helper()

	getenv := os.Getenv("TEST_DSN")
	testDB, err := sqlx.Connect("pgx", getenv)
	if err != nil {
		t.Fatal(err)
	}

	return testDB
}
