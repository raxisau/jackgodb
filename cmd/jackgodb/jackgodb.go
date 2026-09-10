package main

import (
	"log/slog"
	"os"
)

func main() {
	setUpLogging()

	slog.Info("server stopped")
}

func setUpLogging() *slog.Logger {
	// https://pkg.go.dev/log/slog#example-SetLogLoggerLevel-Log
	opts := &slog.HandlerOptions{Level: slog.LevelDebug}

	log := slog.New(slog.NewJSONHandler(os.Stdout, opts))
	slog.SetDefault(log)
	slog.Info("Starting System")

	return log
}
