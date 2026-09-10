package jackgodb

import (
	"database/sql"
	"time"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/mysqldialect"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/dialect/sqlitedialect"
	"github.com/uptrace/bun/driver/pgdriver"
	"github.com/uptrace/bun/driver/sqliteshim"
	"github.com/uptrace/bun/extra/bundebug"
)

const (
	MaxOpenConns = 25
	MaxIdleConns = 25
	MaxLifetime  = 5 * time.Minute
)

// https://bun.uptrace.dev/guide/models.html
// https://bun.uptrace.dev/
// https://bun.uptrace.dev/guide/golang-orm.html

var (
	SQLDebug = false
)

func OpenMySQLDBConnection(dsn string) *DAO {
	sqlDB, err := sql.Open("mysql", dsn)
	if err != nil {
		panic(err)
	}

	sqlDB.SetMaxOpenConns(MaxOpenConns)
	sqlDB.SetMaxIdleConns(MaxIdleConns)
	sqlDB.SetConnMaxLifetime(MaxLifetime)

	db := bun.NewDB(sqlDB, mysqldialect.New())

	if SQLDebug {
		db.AddQueryHook(bundebug.NewQueryHook(
			bundebug.WithVerbose(true),
		))
	}

	dao := NewDAO(db)
	dao.dialect = DialectMySQL

	return dao
}
func OpenSQLiteDBConnection(dsn string) *DAO {
	if dsn == "" {
		dsn = "file::memory:?cache=shared"
	}
	sqlDB, err := sql.Open(sqliteshim.ShimName, dsn)
	if err != nil {
		panic(err)
	}

	db := bun.NewDB(sqlDB, sqlitedialect.New())

	if SQLDebug {
		db.AddQueryHook(bundebug.NewQueryHook(
			bundebug.WithVerbose(true),
		))
	}

	dao := NewDAO(db)
	dao.dialect = DialectSQLite

	return dao
}
func OpenPostgreSQLDBConnection(dsn string) *DAO {
	sqlDB := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(dsn)))

	err := sqlDB.Ping()
	if err != nil {
		sqlDB.Close()
		panic(err)
	}

	db := bun.NewDB(sqlDB, pgdialect.New())

	if SQLDebug {
		db.AddQueryHook(bundebug.NewQueryHook(
			bundebug.WithVerbose(true),
		))
	}

	dao := NewDAO(db)
	dao.dialect = DialectPostgreSQL

	return dao
}
