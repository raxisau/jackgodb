package jackgodb

import (
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"sort"

	"github.com/uptrace/bun"
)

// ErrUnknownKey is returned by ToWhereE when a where or order key is not an alias or column.
var ErrUnknownKey = errors.New("jackgodb: unknown column key")

// ErrBadArgs is returned by ToWhereE when an argument has the wrong shape.
var ErrBadArgs = errors.New("jackgodb: bad query argument")

// identRe is a plain column identifier: letters, digits, underscore, not starting with a digit.
var identRe = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// ToWhereE applies limits, order and where from args to query. Keys in "where" and "order" must
// be an alias or a real column name. Anything else returns ErrUnknownKey, so no raw SQL is
// accepted from keys.
//
// Supported args:
//
//	"limits": int, or [offset, limit]
//	"order":  {"column": "asc"|"desc"}, or ["column", ...]
//	"where":  {"column": value}  (equality), or ["column", ...] for a column-only key
//	          (not supported: use query.Where directly for expressions)
func (m *ModelRecord[T]) ToWhereE(query *bun.SelectQuery, args map[string]any) (*bun.SelectQuery, error) {
	if v, ok := args["limits"]; ok {
		switch lv := v.(type) {
		case []any:
			if len(lv) != 2 {
				return query, fmt.Errorf("%w: limits list must be [offset, limit]", ErrBadArgs)
			}
			query.Offset(AnyToInt64(lv[0])).Limit(AnyToInt64(lv[1]))
		default:
			query.Limit(AnyToInt64(lv))
		}
	}

	if v, ok := args["order"]; ok {
		switch ov := v.(type) {
		case map[string]any:
			for _, key := range sortedKeys(ov) {
				col, err := m.resolveColumn(key)
				if err != nil {
					return query, err
				}
				query.OrderBy(col, toOrder(ov[key]))
			}
		case []any:
			for _, item := range ov {
				col, err := m.resolveColumn(fmt.Sprint(item))
				if err != nil {
					return query, err
				}
				query.OrderExpr(col)
			}
		default:
			col, err := m.resolveColumn(fmt.Sprint(ov))
			if err != nil {
				return query, err
			}
			query.OrderExpr(col)
		}
	}

	if v, ok := args["where"]; ok {
		wv, isMap := v.(map[string]any)
		if !isMap {
			return query, fmt.Errorf("%w: where must be a map of column to value", ErrBadArgs)
		}
		for _, key := range sortedKeys(wv) {
			col, err := m.resolveColumn(key)
			if err != nil {
				return query, err
			}
			query.Where(col+" = ?", wv[key])
		}
	}

	return query, nil
}

// ToWhere is the logging wrapper around ToWhereE. On error it logs and returns the query with
// the error's clauses skipped, so callers that ignore errors do not get an unfiltered query.
// Use ToWhereE to see the error.
func (m *ModelRecord[T]) ToWhere(query *bun.SelectQuery, args map[string]any) *bun.SelectQuery {
	q, err := m.ToWhereE(query, args)
	if err != nil {
		slog.Error("ModelRecord[T].ToWhere", "error", err)
	}
	return q
}

// resolveColumn maps an alias or column name to a real column. It rejects anything that is not
// a plain identifier, so no SQL can come through a key.
func (m *ModelRecord[T]) resolveColumn(key string) (string, error) {
	if real, ok := m.Dao.alias[key]; ok {
		return real, nil
	}
	if identRe.MatchString(key) && m.hasColumn(key) {
		return key, nil
	}
	return "", fmt.Errorf("%w: %q", ErrUnknownKey, key)
}

func (m *ModelRecord[T]) hasColumn(col string) bool {
	for _, c := range m.meta.Columns {
		if c == col {
			return true
		}
	}
	return false
}

func sortedKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
