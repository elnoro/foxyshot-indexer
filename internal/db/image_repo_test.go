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

		err := repo.Upsert(context.Background(), domain.Image{
			FileID:      testFileID,
			Description: "original-description",
		})
		tt.NoErr(err)

		var desc string
		query := `select description from image_descriptions where file_id = $1`
		args := []any{testFileID}

		err = testDB.QueryRow(query, args...).Scan(&desc)
		tt.NoErr(err)
		tt.Equal(desc, "original-description")

		err = repo.Upsert(context.Background(), domain.Image{
			FileID:      testFileID,
			Description: "updated-description",
		})
		tt.NoErr(err)
		err = testDB.QueryRow(query, args...).Scan(&desc)
		tt.NoErr(err)
		tt.Equal(desc, "updated-description")
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

	t.Run("Upsert replaces description on when the same file id is used", func(t *testing.T) {
		tt := is.New(t)

		query := `truncate image_descriptions`
		_, err := testDB.Exec(query)
		tt.NoErr(err)

		want := time.Unix(0, 0).UTC()
		got, err := repo.GetLastModified(ctx)

		tt.NoErr(err)
		tt.Equal(want, got)
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
