package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/notifique/shared/dto"
	"github.com/notifique/worker/internal/di"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	notificationMsgChan := make(chan dto.NotificationMsg)

	worker, close, err := di.InjectRabbitMQWorker(ctx, nil, notificationMsgChan)

	if err != nil {
		panic(err)
	}

	defer close()

	go worker.Start(ctx)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		slog.Info("Received shutdown signal, shutting down gracefully...")
		cancel() // Cancel context when signal is received
	}()

	<-ctx.Done() // Wait for context cancellation
}
