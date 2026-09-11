# jackgodb

A generic ORM + DAO layer for Go, built on top of [bun](https://bun.uptrace.dev/). It adds typed
generic data-access objects, per-record dirty tracking, map-based dynamic query building, and a
schema-to-model code generator — so your models expose CRUD without writing SQL for the common paths.

Supported backends: **MySQL**, **SQLite**, and **PostgreSQL** (single code path, dialect-aware).

- Module: `github.com/raxisau/jackgodb`
- Requires: Go 1.26+
- License: The Unlicense (public domain) — see [LICENSE](LICENSE)

## Installation

```bash
go get github.com/raxisau/jackgodb@latest
```

## Core concepts

| Type | Purpose |
|---|---|
| `DAO` | Wraps a `*bun.DB` plus the active dialect (`mysql` / `sqlite` / `pgsql`). Created by the `Open*DBConnection` helpers. |
| `ModelDAO[T]` | A typed, generic DAO for one model. Knows the column alias map and how to construct new attached records. |
| `ModelRecord[T]` | Embedded in your model struct. Provides dirty tracking, CRUD, reflection-based get/set, and the dynamic query builder. |

Your model embeds both a `bun.BaseModel` (table name/alias) and `ModelRecord[*T]`:

```go
package demo

import (
	"context"
	"log/slog"
	"time"

	"github.com/raxisau/jackgodb"
	"github.com/uptrace/bun"
)

type StatusesDAO struct {
	*jackgodb.ModelDAO[*Statuses]
}

type Statuses struct {
	bun.BaseModel `bun:"table:reg_statuses,alias:RSTAT"`

	jackgodb.ModelRecord[*Statuses] `bun:"-" json:"-"`

	ID                  int64     `bun:"id,pk,autoincrement" json:"id,omitempty"`
	StatusCode          string    `bun:"f_status_code,unique" json:"f_status_code,omitempty"`
	StatusDescription   *string   `bun:"f_status_description" json:"f_status_description,omitempty"`
	StatusReasonDefault *string   `bun:"f_status_reason_default" json:"f_status_reason_default,omitempty"`
	StatusURL           *string   `bun:"f_status_url" json:"f_status_url,omitempty"`
	CreatedAt           time.Time `bun:"f_created_at" json:"f_created_at,omitzero"`
	Comments            *string   `bun:"f_comments" json:"f_comments,omitempty"`
}

var aliasStatuses = map[string]string{
	"id":                  "id",
	"statusCode":          "f_status_code",
	"statusDescription":   "f_status_description",
	"statusReasonDefault": "f_status_reason_default",
	"statusURL":           "f_status_url",
	"createdAt":           "f_created_at",
	"comments":            "f_comments",
}

func NewStatusesDAO(dao *jackgodb.DAO) *StatusesDAO {
	return &StatusesDAO{
		ModelDAO: jackgodb.NewModelDAO(
			dao,
			aliasStatuses,
			func() *Statuses {
				return &Statuses{}
			}),
	}
}

func (dao *StatusesDAO) NewStatuses() *Statuses {
	record := dao.New()
	return record
}

func (m *Statuses) LoadByName(statusCode string) *Statuses {

	record := m.Self
	record.SetID(0)

	if statusCode == "" {
		return nil
	}

	err := m.Dao.Db.NewSelect().
		Model(record).
		Where("f_status_code=?", statusCode).
		Limit(1).
		Scan(context.Background())

	if err != nil {
		slog.Error("1. Statuses.LoadByName", "error", err)
		return nil
	}

	return m
}
```

The alias map translates the keys you pass to `SetValue` / `GetValue` / `Dirty` / `ToWhere` into
real column names, so application code never has to spell out `f_`-prefixed column names.

A complete working reference model lives in [`internal/demo/statuses.go`](internal/demo/statuses.go).

## Quickstart

```go
package demo

import (
	"context"
	"log/slog"
	"time"

	"github.com/raxisau/jackgodb"
	"github.com/uptrace/bun"
)

type StatusesDAO struct {
	*jackgodb.ModelDAO[*Statuses]
}

type Statuses struct {
	bun.BaseModel `bun:"table:reg_statuses,alias:RSTAT"`

	jackgodb.ModelRecord[*Statuses] `bun:"-" json:"-"`

	ID                  int64     `bun:"id,pk,autoincrement" json:"id,omitempty"`
	StatusCode          string    `bun:"f_status_code,unique" json:"f_status_code,omitempty"`
	StatusDescription   *string   `bun:"f_status_description" json:"f_status_description,omitempty"`
	StatusReasonDefault *string   `bun:"f_status_reason_default" json:"f_status_reason_default,omitempty"`
	StatusURL           *string   `bun:"f_status_url" json:"f_status_url,omitempty"`
	CreatedAt           time.Time `bun:"f_created_at" json:"f_created_at,omitzero"`
	Comments            *string   `bun:"f_comments" json:"f_comments,omitempty"`
}

var aliasStatuses = map[string]string{
	"id":                  "id",
	"statusCode":          "f_status_code",
	"statusDescription":   "f_status_description",
	"statusReasonDefault": "f_status_reason_default",
	"statusURL":           "f_status_url",
	"createdAt":           "f_created_at",
	"comments":            "f_comments",
}

func NewStatusesDAO(dao *jackgodb.DAO) *StatusesDAO {
	return &StatusesDAO{
		ModelDAO: jackgodb.NewModelDAO(
			dao,
			aliasStatuses,
			func() *Statuses {
				return &Statuses{}
			}),
	}
}

func (dao *StatusesDAO) NewStatuses() *Statuses {
	record := dao.New()
	return record
}

func (m *Statuses) LoadByName(statusCode string) *Statuses {

	record := m.Self
	record.SetID(0)

	if statusCode == "" {
		return nil
	}

	err := m.Dao.Db.NewSelect().
		Model(record).
		Where("f_status_code=?", statusCode).
		Limit(1).
		Scan(context.Background())

	if err != nil {
		slog.Error("1. Statuses.LoadByName", "error", err)
		return nil
	}

	return m
}
```

## Connections

```go
dao := jackgodb.OpenMySQLDBConnection("user:pass@tcp(127.0.0.1:3306)/mydb")
dao := jackgodb.OpenSQLiteDBConnection("")                                  // in-memory (shared cache)
dao := jackgodb.OpenSQLiteDBConnection("file:data.db?cache=shared&mode=rwc")
dao := jackgodb.OpenPostgreSQLDBConnection("postgres://user:pass@127.0.0.1:5432/mydb")
```

- Pool sizing uses the package constants `MaxOpenConns` (25), `MaxIdleConns` (25) and
  `ConnMaxLifetime` (`MaxLifetime`, 5 minutes).
- Set `jackgodb.SQLDebug = true` before opening a connection to attach bun's debug query hook
  (`bundebug`, verbose) and log every statement.
- `dao.Close()` closes the underlying pool. A `DAO` also exposes `Db` (the raw `*bun.DB`) when you
  need bun directly, and `DbIn(start, count)` which renders a placeholder list for `IN (...)`
  clauses (`?,?` everywhere; `$3,$4,$5`-style on PostgreSQL).

## CRUD reference

All methods below are promoted from the embedded `ModelRecord[T]`. They return row counts / ids
(`int64`) and **log errors via `log/slog`** instead of returning them — see
[Known limitations](#known-limitations).

| Method | Behaviour |
|---|---|
| `Save() int64` | `Insert()` when the primary key is zero, otherwise `Update()`. Returns the new id or rows affected. |
| `Insert() int64` | Inserts **only the dirty columns** (all columns when none are dirty). Returns the new primary key (via `LastInsertId` on MySQL; via `RETURNING` on SQLite and PostgreSQL). |
| `InsertIgnore() int64` | `INSERT IGNORE` / `ON CONFLICT DO NOTHING` equivalent. Existing rows are left untouched. |
| `Update() int64` | Updates only dirty columns by primary key. No-ops (logs an error) when the record is clean. |
| `Delete() int64` | Deletes by primary key. Returns rows affected. |
| `Load(id int64) T` | `SELECT ... WHERE pk = id LIMIT 1`, scanning into the record. On failure the id is reset to 0. |

Dirty tracking:

| Method | Behaviour |
|---|---|
| `Dirty(column string)` | Mark a column (by alias or column name) as modified. |
| `IsClean() / IsDirty() bool` | Whether any column is dirty. |
| `Clean()` | Reset all columns to clean (done automatically after attach and update). |

Direct struct field assignment is **not** tracked. To update a record, change values via
`SetValue` (which marks the column dirty) or call `Dirty` yourself.

## Value API

```go
ok := rec.SetValue("statusCode", "ACTIVE") // alias-resolved, type-coerced, marks dirty
val := rec.GetValue("statusCode")          // any

id := rec.GetID()   // reads the pk column as int64
rec.SetID(id)       // writes the pk column
```

`SetValue` coerces the incoming value to the field's type via `ConvertTo` and only marks the
column dirty when the value actually changed. Supported targets: `string`, `int`, `int64`,
`float64`, `time.Time`, and their nullable (`*`) forms; anything else falls back through
`AnyToString` / `AnyToInt64` helpers.

## API bridging

```go
// Struct -> map[string]any, keyed by json tags.
asMap := rec.ToAPI()

// map[string]any -> struct. If the payload contains the pk column the record is loaded
// from the database first, so it works as an upsert-style PATCH endpoint.
rec.FromAPI(map[string]any{
	"id":         3,
	"statusCode": "ACTIVE",
})
```

## Dynamic queries with `ToWhere`

`ToWhere(query, args)` builds bun `SELECT` clauses from plain maps/lists — useful when the
request shape comes from JSON:

```go
var rows []*Statuses
query := statusDao.Db.NewSelect().Model(&rows)
rec.ToWhere(query, map[string]any{
	"where": map[string]any{
		"statusCode":           "ACTIVE",               // parameterized: WHERE f_status_code = ?
		"statusUrl != ?":       "https://spam.example", // keys with "?" bind values as params
		"comments IS NOT NULL": nil,                    // value-less keys run as raw SQL
	},
	"order": map[string]any{"createdAt": "DESC"},
	"limits": []any{0, 50},             // offset, limit
})
err := query.Scan(context.Background())
statusDao.AttachList(rows) // attach loaded rows so they get full record behaviour
```

| Key | Accepted shapes |
|---|---|
| `limits` | `int` (limit only) · `[]any{offset, limit}` · `map[string]any{"<offset>": limit}` |
| `order` | `"column"` · `map[string]any{"column": "ASC"/"DESC"}` (anything but `DESC` = `ASC`) · `[]any` of columns |
| `where` | `"column"` (raw) · `map[string]any{"column": value}` or `{"sql with ?": value}` · `[]any` of raw clauses |

Keys are resolved through the alias map and converted to real column names; **values** are bound
as query parameters, but **string/list keys** are interpolated as SQL — never pass untrusted
input there (see [Known limitations](#known-limitations)).

## Code generation with `modelgen`

`cmd/modelgen` turns a MySQL-flavoured `CREATE TABLE` statement into a ready-to-use model file
(bun tags, alias map, DAO constructor):

```bash
# From a file, print to stdout
go run ./cmd/modelgen -file stdout schema.sql

# Or pipe it in
cat schema.sql | go run ./cmd/modelgen -file infer   # writes <snake_case_name>.go
go run ./cmd/modelgen -file model/statuses.go schema.sql
```

Conventions the generator understands:

- Table names prefixed `tbl` and column names prefixed `fld` / `f_` are stripped before
  Pascal-casing (`tblUser` -> `User`, `f_status_code` -> `StatusCode`).
- Types: `int*` -> `int64`, `tinyint(1)` -> `bool`, `decimal/float/double` -> `float64`,
  `date/datetime/timestamp` -> `time.Time`, text/varchar/json -> `string`; nullable columns
  become pointer types.
- `PRIMARY KEY` and `UNIQUE KEY` lines are detected; `auto_increment` produces the
  `autoincrement` bun tag.
- A column named `f_updated_at` (or `fldLastUpdated`) is forced to `time.Time` and tagged
  `nullzero,notnull,default:current_timestamp`.

## Model metadata rules

`ModelRecord` caches metadata per concrete type (table name, primary key, column list,
column -> field index) by reading your `bun` tags:

- Fields need a `bun` tag to participate; `bun:"-"` fields are ignored (as embedded
  `ModelRecord` is).
- The model **must** declare a table (`table:...` on any field's bun tag, normally the
  `bun.BaseModel`) and a column tagged `pk`. If either is missing, metadata is nil and value/
  CRUD calls log "model metadata is nil".
- Tag options: `pk` marks the primary key (kept out of dirty tracking); `scanonly` columns are
  readable but never written.
- A column named `fldLastUpdated` tagged `default:current_timestamp` is treated as database-managed
  and excluded from inserts/updates.

## Package API summary

```go
// db.go
func OpenMySQLDBConnection(dsn string) *DAO
func OpenSQLiteDBConnection(dsn string) *DAO
func OpenPostgreSQLDBConnection(dsn string) *DAO
const MaxOpenConns, MaxIdleConns = 25, 25
const MaxLifetime = 5 * time.Minute
var SQLDebug bool

// dao.go
func NewDAO(db *bun.DB) *DAO
func (dao *DAO) Close()
func (dao *DAO) DbIn(start, count int) string
func NewModelDAO[T DBModel](dao *DAO, alias map[string]string, newRecord func() T) *ModelDAO[T]
func (dao *ModelDAO[T]) New() T
func (dao *ModelDAO[T]) Attach(record T)
func (dao *ModelDAO[T]) AttachList(records []T)

// orm.go — promoted onto your model via ModelRecord[T]
func (m) Save() / Insert() / InsertIgnore() / Update() / Delete() int64
func (m) Load(id int64) T
func (m) SetValue(column string, value any) bool
func (m) GetValue(column string) any
func (m) GetID() int64 / SetID(id int64)
func (m) Dirty(column string) / IsClean() / IsDirty() bool / Clean()
func (m) ToAPI() map[string]any
func (m) ToJSON() string
func (m) ToJSONIndent() string
func (m) FromAPI(data map[string]any) T
func (m) ToWhere(query *bun.SelectQuery, args map[string]any) *bun.SelectQuery

// converter.go
func ConvertTo(value any, targetType reflect.Type) (any, error)
func ParseUTCDateTime(value string) (time.Time, error)
func AnyToTime(value any) time.Time
func AnyToInt(value any) int
func AnyToInt64(value any) int64
func AnyToString(value any) string
func StringValue(*string) string
```

## Known limitations

- **Errors are logged, not returned.** CRUD methods write failures to `log/slog` and return `0` /
  empty records; callers cannot distinguish "not found" from a connection failure. Treat this as
  the library's biggest rough edge.
- **Raw SQL surface in `ToWhere`.** String/list `where` and `order` keys are interpolated as SQL.
  Parameterize everything user-controlled or validate against the alias map first.
- **No context parameter.** Queries run on `context.Background()`; there is no cancellation or
  timeout plumbing yet.
- **`Update()` no-ops on clean records** (logging an error), so field writes must go through
  `SetValue`/`Dirty`.
- **`modelgen` targets MySQL-style DDL** (backtick-quoted identifiers, `auto_increment`, inline
  `PRIMARY KEY`/`UNIQUE KEY` lines).

## Project layout

```
cmd/jackgodb/    Demo binary (in-memory SQLite; creates the demo table)
cmd/modelgen/    CREATE TABLE -> Go model code generator
internal/demo/   Reference model (Statuses) showing the expected shape
db.go            Connection openers for MySQL / SQLite / PostgreSQL
dao.go           DAO and generic ModelDAO[T]
orm.go           ModelRecord[T]: metadata, dirty tracking, CRUD, ToWhere
converter.go     Type coercion and date parsing helpers
```

## Development

```bash
make jgdb   # Run the demo
make test   # Run the tests
make cover  # check the coverage from tests
```
