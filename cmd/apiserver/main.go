package main

import (
	"log/slog"
	"os"

	"github.com/chaewonkong/matchmaker/services/apiserver"
	"github.com/labstack/echo/v4"
)

func main() {
	code := run()
	os.Exit(code)
}

func run() int {
	logger := slog.New(slog.NewJSONHandler(os.Stderr, nil))
	logger.Info("Starting the application...")

	e := echo.New()
	h := apiserver.NewHandler()
	apiserver.RegisterRoutes(e, h)

	err := e.Start(":8080")
	if err != nil {
		logger.Error("Failed to start the server", "error", err)
		return 1
	}

	return 0
}
