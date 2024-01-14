package embedding

import (
	"github.com/matryer/is"
	"log/slog"
	"testing"
)

func TestClient_CreateEmbeddingForFile(t *testing.T) {
	tt := is.New(t)
	cl := NewClient("http://embeddings:8000", slog.Default())

	vector, err := cl.CreateEmbeddingForFile("https%3A%2F%2Fs.foxyshot.me%2F01d4e576-78d5-4ca0-80a0-775af99829a6.jpg")

	tt.NoErr(err)
	tt.Equal(len(vector), 512) // default model must 512 parameters
}
