package main

import (
	"log/slog"
	"os"
)

func main() {
	code := run()
	os.Exit(code)
}

func run() int {
	logger := slog.New(slog.NewJSONHandler(os.Stderr, nil))
	logger.Info("Starting the application...")

	return 0
}
