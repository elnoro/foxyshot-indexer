package s3wrapper

import (
	"errors"
	"log/slog"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/matryer/is"
)

func TestNewClientFromSecrets_SessionError(t *testing.T) {
	tt := is.New(t)
	// this might break after an upgrade - doing this only to achieve 100% for the module
	// realistically this test is not necessary
	const stsEndpoint = "AWS_STS_REGIONAL_ENDPOINTS"
	oldval := os.Getenv(stsEndpoint)
	defer os.Setenv(stsEndpoint, oldval)

	err := os.Setenv(stsEndpoint, "invalid-sts-value")
	tt.NoErr(err)

	v, err := NewFromSecrets("", "", "", "", "", false, slog.Default())

	tt.True(v == nil)
	tt.True(strings.Contains(err.Error(), "invalid-sts-value")) // must return an error because of invalid env var
}

func TestBucketClient_CheckConnectivity(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	tt := is.New(t)
	cl := validClient(t)

	t.Run("successful connection", func(t *testing.T) {
		err := cl.CheckConnectivity(1, 0)

		tt.NoErr(err)
	})

	t.Run("zero attempts", func(t *testing.T) {
		err := cl.CheckConnectivity(0, 0)

		tt.True(errors.Is(err, ErrNoAttempts)) // when passing 0 attempts, return special error
	})

	t.Run("unsuccessful connection (invalid key)", func(t *testing.T) {
		cl, err := NewFromSecrets(
			"invalid-key",
			os.Getenv("S3_SECRET"),
			os.Getenv("S3_ENDPOINT"),
			"eu-west-1",
			os.Getenv("S3_BUCKET"),
			true,
			slog.Default(),
		)
		tt.NoErr(err)

		err = cl.CheckConnectivity(2, 1*time.Millisecond)

		tt.True(strings.Contains(err.Error(), "failed to initialize bucket client")) // must return init error
	})

}

func validClient(t *testing.T) *BucketClient {
	t.Helper()

	cl, err := NewFromSecrets(
		os.Getenv("S3_KEY"),
		os.Getenv("S3_SECRET"),
		os.Getenv("S3_ENDPOINT"),
		"eu-west-1",
		os.Getenv("S3_BUCKET"),
		true,
		slog.Default(),
	)

	if err != nil {
		t.Errorf("failed to create valid client for test. check env!")
	}

	return cl
}
