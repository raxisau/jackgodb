package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/raxisau/jackgodb"
	"github.com/raxisau/jackgodb/internal/demo"
)

func main() {
	setUpLogging()

	// Empty DSN = in-memory SQLite. See "Connections" for MySQL/PostgreSQL.
	dao := jackgodb.OpenSQLiteDBConnection("")
	defer dao.Close()

	statusDao := demo.NewStatusesDAO(dao)

	// Create the table (bun DDL; in production use migrations).
	ctx := context.Background()
	_, err := dao.Db.NewCreateTable().Model((*demo.Statuses)(nil)).IfNotExists().Exec(ctx)
	if err != nil {
		panic(err)
	}

	// Insert
	rec := statusDao.NewStatuses()
	rec.SetValue("statusCode", "ACTIVE")
	rec.SetValue("statusDescription", "rec.SetValue statusDescription")
	rec.SetValue("statusReasonDefault", "rec.SetValue statusReasonDefault")
	rec.SetValue("statusURL", "http://rec.SetValuestatusURL")
	rec.SetValue("comments", "rec.SetValue comments")
	id := rec.Save() // returns the new primary key

	// Load
	loaded := statusDao.NewStatuses()
	loaded.Load(id)
	fmt.Println(loaded.ToJSONIndent())

	// Update — use SetValue, which marks the column dirty
	loaded.SetValue("statusCode", "RETIRED")
	loaded.Save()

	loaded.Load(id)
	fmt.Println(loaded.ToJSONIndent())

	// Delete
	loaded.Delete()
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
