package docker

import (
	"context"
	"errors"
	"testing"
)

func TestWaitForAttachDoneNilChannel(t *testing.T) {
	t.Parallel()

	if err := waitForAttachDone(context.Background(), nil); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}

func TestWaitForAttachDoneCopyError(t *testing.T) {
	t.Parallel()

	copyErr := errors.New("copy failed")
	attachDone := make(chan error, 1)
	attachDone <- copyErr

	err := waitForAttachDone(context.Background(), attachDone)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, copyErr) {
		t.Fatalf("expected wrapped copy error, got %v", err)
	}
}

func TestWaitForAttachDoneContextDone(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	attachDone := make(chan error)
	err := waitForAttachDone(ctx, attachDone)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context canceled error, got %v", err)
	}
}
