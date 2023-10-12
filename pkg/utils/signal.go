package utils

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

var ErrSignal = errors.New("signal received")

func WithSignal(parentCtx context.Context) context.Context {
	ctx, cancel := context.WithCancelCause(parentCtx)

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		s := <-sig
		cancel(fmt.Errorf("%w: %s", ErrSignal, s))
	}()

	return ctx
}
