package awips

import (
	"context"
	"log/slog"
	"os/signal"
	"syscall"

	"github.com/metdatasystem/us/services/parse/awips/internal/server"
)

func Go() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	server, err := server.New(ctx)
	if err != nil {
		slog.Error("failed to create server instance", "error", err.Error())
		return
	}

	server.Run()

	<-ctx.Done()
	server.Stop()

}
