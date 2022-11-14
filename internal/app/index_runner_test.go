package app

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/matryer/is"
)

func TestIndexRunner_Start_NoError(t *testing.T) {
	tt := is.New(t)

	li := &listIndexerMock{IndexNewListFunc: func(_ context.Context, _ string) error { return nil }}

	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()
	runner := NewIndexRunner(li, "expected-ext", 100*time.Millisecond)

	err := runner.Start(ctx)

	tt.True(errors.Is(err, context.DeadlineExceeded)) // must end by deadline
	tt.Equal(len(li.IndexNewListCalls()), 2)          // must run 2 times (start immediately + 1 timer)
	tt.Equal(li.IndexNewListCalls()[0].S, "expected-ext")
}

func TestIndexRunner_Start_Error(t *testing.T) {
	tt := is.New(t)

	expectedErr := errors.New("expected-err")
	li := &listIndexerMock{IndexNewListFunc: func(_ context.Context, _ string) error { return expectedErr }}

	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()
	runner := NewIndexRunner(li, "expected-ext", 100*time.Millisecond)

	err := runner.Start(ctx)

	tt.True(errors.Is(err, expectedErr)) // must end after the first call
	tt.Equal(len(li.IndexNewListCalls()), 1)
}
