package main

import (
	"bytes"
	"context"
	"fmt"
	"github.com/elnoro/foxyshot-indexer/internal/monitoring"
	"github.com/matryer/is"
	"log"
	"net"
	"net/http"
	"sync"
	"testing"
)

func newTestApp(repo *imageRepoMock, fs fileStorage) *webApp {
	if fs == nil {
		fs = &fileStorageMock{}
	}
	if repo == nil {
		repo = &imageRepoMock{}
	}

	app := &webApp{
		config:            Config{},
		log:               log.Default(),
		imageDescriptions: repo,
		fileStorage:       fs,
		tracker:           monitoring.NewTracker(),
	}
	return app
}

func Test_webApp_serve(t *testing.T) {
	wg := sync.WaitGroup{}
	defer wg.Wait()

	tt := is.New(t)

	port, err := findUnusedPort()
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	app := newTestApp(nil, nil)
	app.config.Port = port

	wg.Add(1)
	go func() {
		defer wg.Done()
		_ = app.serve(ctx)
	}()

	testURL := fmt.Sprintf("http://localhost:%d", port)

	testCases := []struct {
		endpoint     string
		expectedCode int
	}{
		{"/metrics", http.StatusOK},
		{"/debug/vars", http.StatusOK},
		{"/healthcheck", http.StatusOK},
		{"/not-found", http.StatusNotFound},
	}

	for _, tc := range testCases {
		resp, err := http.DefaultClient.Get(testURL + tc.endpoint)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		tt.NoErr(err)
		tt.Equal(resp.StatusCode, tc.expectedCode)
	}

	resp, err := http.DefaultClient.Post(testURL+"/healthcheck", "", bytes.NewBufferString(""))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	tt.NoErr(err)
	tt.Equal(resp.StatusCode, http.StatusMethodNotAllowed)

}

func findUnusedPort() (int, error) {
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		return 0, err
	}

	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}
