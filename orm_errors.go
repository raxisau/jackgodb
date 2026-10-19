package jackgodb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
)

// ErrNoChanges is returned by UpdateE when the record has no dirty columns.
var ErrNoChanges = errors.New("jackgodb: no changes to update")

// ErrNotFound is returned by LoadE when no row matches the id.
var ErrNotFound = errors.New("jackgodb: record not found")

// SaveE inserts when the ID is zero, otherwise updates. Unlike Save it returns the error.
func (m *ModelRecord[T]) SaveE() (int64, error) {
	if m.Self.GetID() == 0 {
		return m.InsertE()
	}
	return m.UpdateE()
}

// InsertE inserts the record and returns its new ID. A unique-constraint violation is returned
// as an error. Unlike Insert it does not use INSERT OR IGNORE.
func (m *ModelRecord[T]) InsertE() (int64, error) {
	record := m.Self

	q := m.Dao.Db.NewInsert().Model(m.Self)
	// for column, dirty := range m.dirty {
	// 	if dirty {
	// 		q.Column(column)
	// 	}
	// }
	_, err := q.Exec(context.Background())
	if err != nil {
		slog.Error("1. ModelRecord[T].InsertE", "error", err)
		return 0, fmt.Errorf("insert: %w", err)
	}

	m.Clean()
	return record.GetID(), nil
}

// UpdateE updates the dirty columns and returns the rows affected. It returns ErrNoChanges
// when nothing is dirty, so "no changes" is no longer confused with failure.
func (m *ModelRecord[T]) UpdateE() (int64, error) {
	if m.IsClean() {
		return 0, ErrNoChanges
	}

	q := m.Dao.Db.NewUpdate().Model(m.Self).WherePK()
	for column, dirty := range m.dirty {
		if dirty {
			q.Column(column)
		}
	}

	result, err := q.Exec(context.Background())
	if err != nil {
		slog.Error("2. ModelRecord[T].UpdateE", "error", err)
		return 0, fmt.Errorf("update: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("rows affected: %w", err)
	}

	m.Clean()
	return rows, nil
}

// LoadE loads the record by id. It returns ErrNotFound when no row matches. On any error it
// returns the zero value of T, so callers should check err before using the result.
func (m *ModelRecord[T]) LoadE(id int64) (T, error) {
	record := m.Self
	record.SetID(id)

	err := m.Dao.Db.NewSelect().
		Model(record).
		WherePK().
		Limit(1).
		Scan(context.Background())

	if err != nil {
		var zero T
		if errors.Is(err, sql.ErrNoRows) {
			return zero, ErrNotFound
		}
		slog.Error("1. ModelRecord[T].LoadE", "error", err)
		return zero, fmt.Errorf("load: %w", err)
	}

	return record, nil
}
