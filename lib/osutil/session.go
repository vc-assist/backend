package osutil

import (
	"context"
	"os"
	"os/signal"
	"syscall"
)

// Returns a context that will live until Ctrl+C is pressed
func SignalContext() context.Context {
	ctx, cancel := context.WithCancel(context.Background())

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigs
		cancel()
	}()

	return ctx
}
