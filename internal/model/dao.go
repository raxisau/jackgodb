package model

import (
	"database/sql"
	"strconv"
	"strings"

	"github.com/uptrace/bun"
)

const (
	DialectMySQL      = "mysql"
	DialectSQLite     = "sqlite"
	DialectPostgreSQL = "pgsql"
)

type DAO struct {
	db      *bun.DB
	dialect string
	alias   map[string]string
}

func NewDAO(db *bun.DB) *DAO {
	return &DAO{
		db: db,
	}
}

func (dao *DAO) Close() {
	dao.db.Close()
}

func (dao *DAO) GetDB() *sql.DB {
	return dao.db.DB
}

func (dao *DAO) DbIn(start, count int) string {
	placeholders := make([]string, count)

	switch dao.dialect {
	case DialectPostgreSQL:
		for i := range placeholders {
			placeholders[i] = "$" + strconv.Itoa(start+i)
		}

	default:
		for i := range placeholders {
			placeholders[i] = "?"
		}
	}
	return strings.Join(placeholders, ",")
}

type ModelDAO[T DBModel] struct {
	db        *bun.DB
	alias     map[string]string
	newRecord func() T
}

type ModelDAOAttacher[T DBModel] interface {
	AttachModelDAO(*ModelDAO[T], T)
}

func NewModelDAO[T DBModel](dao *DAO, alias map[string]string, newRecord func() T) *ModelDAO[T] {
	return &ModelDAO[T]{
		db:        dao.db,
		alias:     alias,
		newRecord: newRecord,
	}
}

func (dao *ModelDAO[T]) New() T {
	record := dao.newRecord()
	dao.Attach(record)

	return record
}

func (dao *ModelDAO[T]) Attach(record T) {
	recordAny := any(record)

	attachable, ok := recordAny.(ModelDAOAttacher[T])
	if !ok {
		return
	}

	attachable.AttachModelDAO(dao, record)
}

func (dao *ModelDAO[T]) AttachList(recordList []T) {
	for _, record := range recordList {
		dao.Attach(record)
	}
}
