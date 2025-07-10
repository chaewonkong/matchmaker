package main

import (
	"log/slog"
	"os"

	"github.com/chaewonkong/matchmaker/schema"
	"github.com/chaewonkong/matchmaker/services/apiserver"
	"github.com/labstack/echo/v4"
	"gopkg.in/yaml.v3"
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
	f, err := os.Open(queueConfigPath)
	if err != nil {
		slog.Error("failed to open queue config file", "error", err)
		return ExitCodeFailure
	}

	queueConfig := schema.QueueConfig{}
	err = yaml.NewDecoder(f).Decode(&queueConfig)
	if err != nil {
		slog.Error("failed to decode queue config file", "error", err)
		return ExitCodeFailure
	}

	logger := slog.New(slog.NewJSONHandler(os.Stderr, nil))
	logger.Info("starting the application...")

	e := echo.New()
	h := apiserver.NewHandler()
	apiserver.RegisterRoutes(e, h)

	err = e.Start(":8080")
	if err != nil {
		logger.Error("failed to start the server", "error", err)
		return ExitCodeFailure
	}

	return ExitCodeSuccess
}
