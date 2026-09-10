package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/raxisau/jackgodb"
	"github.com/raxisau/jackgodb/internal/demo"
)

func main() {
	setUpLogging()

	dao := jackgodb.OpenSQLiteDBConnection("")
	statusDao := demo.NewStatusesDAO(dao)
	rec := statusDao.NewStatuses()

	_, err := rec.Dao.Db.NewCreateTable().Model((*demo.Statuses)(nil)).IfNotExists().Exec(context.Background())
	if err != nil {
		panic(err)
	}

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
