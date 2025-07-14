package main

import (
	"log/slog"
	"os"

	"github.com/chaewonkong/matchmaker/schema"
	"github.com/chaewonkong/matchmaker/services/apiserver"
	"github.com/chaewonkong/matchmaker/services/apiserver/usecase"
	"github.com/chaewonkong/matchmaker/services/queue"
	"github.com/labstack/echo/v4"
)

const (
	// ExitCodeSuccess success
	ExitCodeSuccess = 0
	// ExitCodeFailure failure
	ExitCodeFailure = 1
)

func main() {
	code := run()
	os.Exit(code)
}

func run() int {
	queueConfigPath := os.Getenv("QUEUE_CONFIG_PATH")

	queueConfig := schema.NewQueueConfig()
	err := queueConfig.UnmarshalFromYAML(queueConfigPath)
	if err != nil {
		slog.Error("failed to load queue config", "error", err)
		return ExitCodeFailure
	}

	logger := slog.New(slog.NewJSONHandler(os.Stderr, nil))
	logger.Info("starting the application...")

	e := echo.New()
	q := queue.New()
	ts := usecase.NewTicketService(q)
	h := apiserver.NewHandler(ts)
	apiserver.RegisterRoutes(e, h)

	err = e.Start(":8080")
	if err != nil {
		logger.Error("failed to start the server", "error", err)
		return ExitCodeFailure
	}

	return ExitCodeSuccess
}
