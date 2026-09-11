package jackgodb

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"reflect"
	"regexp"
	"slices"
	"strings"
	"sync"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/schema"
)

type DBModel interface {
	GetID() int64
	SetID(int64)
}

type ModelMetadata struct {
	TableName     string         `json:"TableName,omitempty"`
	PKField       string         `json:"PKField,omitempty"`
	PKColumn      string         `json:"PKColumn,omitempty"`
	Columns       []string       `json:"Columns,omitempty"`
	FieldByColumn map[string]int `json:"FieldByColumn,omitempty"`
}

type ModelRecord[T DBModel] struct {
	Dao      *ModelDAO[T]
	meta     *ModelMetadata
	dirty    map[string]bool
	attached bool
	Self     T
}

func (m *ModelRecord[T]) AttachModelDAO(dao *ModelDAO[T], self T) {
	if m.attached {
		return
	}
	m.Dao = dao
	m.Self = self
	m.attached = true
	m.meta = m.GetModelMetadata()
	m.Clean()
}
func (m *ModelRecord[T]) IsClean() bool {
	if m.dirty == nil {
		m.dirty = make(map[string]bool)
	}

	for _, column := range m.meta.Columns {
		if m.dirty[column] {
			return false
		}
	}

	return true
}
func (m *ModelRecord[T]) IsDirty() bool {
	return !m.IsClean()
}

func (m *ModelRecord[T]) Clean() {
	if m.dirty == nil {
		m.dirty = make(map[string]bool)
	}

	for _, column := range m.meta.Columns {
		m.dirty[column] = false
	}
}

func (m *ModelRecord[T]) Dirty(column string) {
	correctColumnName, hasAlias := m.Dao.alias[column]
	if hasAlias {
		column = correctColumnName
	}

	val, hasCol := m.dirty[column]
	if !hasCol || val == true {
		return
	}
	m.dirty[column] = true
}

func (m *ModelRecord[T]) Save() int64 {
	record := m.Self

	if record.GetID() == 0 {
		return m.Insert()
	} else {
		return m.Update()
	}
}
func (m *ModelRecord[T]) InsertIgnore() int64 {
	record := m.Self

	_, err := m.Dao.Db.NewInsert().
		Ignore().
		Model(record).
		Exec(context.Background())

	if err != nil {
		slog.Error("1. ModelRecord[T].InsertIgnore", "error", err)
		return 0
	}

	return record.GetID()
}
func (m *ModelRecord[T]) Insert() int64 {
	record := m.Self

	_, err := m.Dao.Db.NewInsert().
		Ignore().
		Model(record).
		Exec(context.Background())

	if err != nil {
		slog.Error("1. ModelRecord[T].InsertIgnore", "error", err)
		return 0
	}

	return record.GetID()
}
func (m *ModelRecord[T]) Update() int64 {
	record := m.Self

	if m.IsClean() {
		slog.Error("2. ModelRecord[T].Update", "error", "nothing to update")
		return 0
	}

	updateQuery := m.Dao.Db.NewUpdate().
		Model(record).
		WherePK()

	// If there is no columns changed then it will just updated all
	for column, dirty := range m.dirty {
		if dirty {
			updateQuery.Column(column)
		}
	}

	result, err := updateQuery.Exec(context.Background())

	if err != nil {
		slog.Error("3. ModelRecord[T].Update", "error", err)
		return 0
	}

	rowCnt, err := result.RowsAffected()
	if err != nil {
		slog.Error("4. ModelRecord[T].Update", "error", err)
		return 0
	}

	m.Clean()
	return rowCnt
}

func (m *ModelRecord[T]) Delete() int64 {
	record := m.Self

	result, err := m.Dao.Db.NewDelete().
		Model(record).
		WherePK().
		Exec(context.Background())

	if err != nil {
		slog.Error("1. ModelDAO[T].Delete", "error", err)
		return 0
	}
	rowCnt, err := result.RowsAffected()
	if err != nil {
		slog.Error("4. ModelDAO[T].Delete", "error", err)
		return 0
	}
	return rowCnt
}

func (m *ModelRecord[T]) GetID() int64 {
	value := m.GetValue(m.meta.PKColumn)
	if value == nil {
		return 0
	}
	return AnyToInt64(value)
}

func (m *ModelRecord[T]) SetID(id int64) {
	m.SetValue(m.meta.PKColumn, id)
}

func (m *ModelRecord[T]) ToAPI() map[string]any {
	jsonModel := m.ToJSON()
	var out map[string]any
	if err := json.Unmarshal([]byte(jsonModel), &out); err != nil {
		slog.Error("2. ModelRecord[T].ToAPI", "error", err)
		return nil
	}
	return out
}
func (m *ModelRecord[T]) ToJSONIndent() string {

	v := reflect.ValueOf(m.Self)
	if !v.IsValid() {
		return ""
	}

	for v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return ""
		}
		v = v.Elem()
	}

	jsonBytes, err := json.MarshalIndent(v.Interface(), "", "  ")
	if err != nil {
		slog.Error("1. ModelRecord[T].ToJSONIndent", "error", err)
		return ""
	}

	return string(jsonBytes)
}
func (m *ModelRecord[T]) ToJSON() string {

	v := reflect.ValueOf(m.Self)
	if !v.IsValid() {
		return ""
	}

	for v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return ""
		}
		v = v.Elem()
	}

	jsonBytes, err := json.Marshal(v.Interface())
	if err != nil {
		slog.Error("1. ModelRecord[T].ToJSON", "error", err)
		return ""
	}

	return string(jsonBytes)
}

func (m *ModelRecord[T]) ToWhere(query *bun.SelectQuery, args map[string]any) *bun.SelectQuery {

	limits, hasLimits := args["limits"]
	if hasLimits {
		switch v := limits.(type) {
		case map[string]any:
			for key, value := range v {
				query.Offset(AnyToInt(key))
				query.Limit(AnyToInt(value))
				break
			}

		case []any:
			query.Offset(AnyToInt(v[0]))
			query.Limit(AnyToInt(v[1]))

		default:
			query.Limit(AnyToInt(v))
		}
	}

	order, hasOrder := args["order"]
	if hasOrder {
		switch v := order.(type) {
		case map[string]any:
			for key, value := range v {
				query.OrderBy(m.getRealColumn(key), toOrder(value))
			}

		case []any:
			// Handle arrays/lists
			for _, item := range v {
				query.OrderExpr(m.getRealColumn(item))
			}

		default:
			query.OrderExpr(m.getRealColumn(v))
		}
	}

	where, hasWhere := args["where"]
	if hasWhere {
		switch v := where.(type) {
		case map[string]any:
			// Handle dictionaries
			for key, value := range v {
				if strings.Contains(key, " ") || strings.Contains(key, "?") || strings.Contains(key, "=") {
					key = m.getRealColumn(key)
				} else {
					key = fmt.Sprintf("%s = ?", m.getRealColumn(key))
				}
				if strings.Contains(key, "?") {
					query.Where(key, value)
				} else {
					query.Where(key)
				}
			}

		case []any:
			// Handle arrays/lists
			for _, item := range v {
				query.Where(m.getRealColumn(item))
			}

		default:
			query.Where(m.getRealColumn(v))
		}
	}

	return query
}

func (m *ModelRecord[T]) Load(id int64) T {
	record := m.Self
	record.SetID(id)

	err := m.Dao.Db.NewSelect().
		Model(record).
		WherePK().
		Limit(1).
		Scan(context.Background())

	if err != nil {
		slog.Error("1. ModelRecord[T].Load", "error", err)
		record.SetID(0)
		return record
	}

	return record
}

func (m *ModelRecord[T]) FromAPI(apiData map[string]any) T {
	metaData := m.meta

	pKeyRaw, hasID := apiData[metaData.PKColumn]
	if hasID {
		pKey := AnyToInt64(pKeyRaw)
		m.Load(pKey)
	}

	changedSomething := false
	for _, colName := range metaData.Columns {

		value, hasCol := apiData[colName]
		if !hasCol {
			continue
		}

		if m.SetValue(colName, value) {
			changedSomething = true
		}
	}
	if changedSomething {
		m.Save()
	}

	return m.Self
}
func toOrder(value any) schema.Order {
	if strings.EqualFold(fmt.Sprintf("%v", value), "DESC") {
		return schema.OrderDesc
	} else {
		return schema.OrderAsc
	}
}

func (m *ModelRecord[T]) getRealColumn(rawColName any) string {
	var re = regexp.MustCompile(`(?i)^([a-z]+)(.*)$`)
	parts := re.FindStringSubmatch(fmt.Sprintf("%v", rawColName))

	correctColumnName, hasAlias := m.Dao.alias[parts[1]]
	if hasAlias {
		parts[1] = correctColumnName
	}
	return strings.Join(parts[1:], "")
}

func (m *ModelRecord[T]) SetValue(columnName string, value any) bool {
	if m.meta == nil {
		slog.Error("model.SetValue", "error", "model metadata is nil")
		return false
	}

	if m.Dao == nil {
		slog.Error("model.SetValue", "error", "model dao is nil")
		return false
	}

	columnName = m.getRealColumn(columnName)

	fieldIndex, hasCol := m.meta.FieldByColumn[columnName]
	if !hasCol {
		return false
	}

	v := reflect.ValueOf(m.Self)

	if v.Kind() != reflect.Pointer || v.IsNil() {
		return false
	}

	v = v.Elem()

	field := v.Field(fieldIndex)
	if !field.CanSet() {
		return false
	}

	oldValue := field.Interface()

	newValue, err := ConvertTo(value, field.Type())
	if err != nil {
		slog.Error("model.SetValue", "error", err)
		return false
	}

	if reflect.DeepEqual(oldValue, newValue) {
		return false
	}

	field.Set(reflect.ValueOf(newValue))
	m.Dirty(columnName)

	return true
}

func (m *ModelRecord[T]) GetValue(columnName string) any {
	if m.meta == nil {
		slog.Error("model.GetValue", "error", "model metadata is nil")
		return nil
	}

	columnName = m.getRealColumn(columnName)

	fieldIndex, hasCol := m.meta.FieldByColumn[columnName]
	if !hasCol {
		return nil
	}

	v := reflect.ValueOf(m.Self)

	if v.Kind() != reflect.Pointer || v.IsNil() {
		return nil
	}

	v = v.Elem()

	field := v.Field(fieldIndex)

	if field.Kind() == reflect.Pointer {
		if field.IsNil() {
			return nil
		}

		return field.Elem().Interface()
	}

	return field.Interface()
}

var modelMetadataCache sync.Map

func (m *ModelRecord[T]) GetModelMetadata() *ModelMetadata {
	v := reflect.ValueOf(m.Self)

	if !v.IsValid() || (v.Kind() == reflect.Pointer && v.IsNil()) {
		return nil
	}

	t := v.Type()

	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}

	cached, ok := modelMetadataCache.Load(t)
	if ok {
		return cached.(*ModelMetadata)
	}

	meta := &ModelMetadata{
		FieldByColumn: make(map[string]int),
	}

	foundTableName := false
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		bunTag := field.Tag.Get("bun")
		if bunTag == "" || bunTag == "-" {
			continue
		}

		parts := strings.Split(bunTag, ",")

		// See if this is the table name
		if !foundTableName {
			for _, part := range parts {
				part = strings.TrimSpace(part)

				if tableName, ok := strings.CutPrefix(part, "table:"); ok {
					meta.TableName = tableName
					foundTableName = true
					break
				}
			}
			if foundTableName {
				continue
			}
		}

		columnName := strings.TrimSpace(parts[0])
		if columnName == "" || strings.Contains(columnName, ":") {
			continue
		}

		meta.FieldByColumn[columnName] = i

		// Skip - Do not save, save primary key details
		if slices.Contains(parts[1:], "pk") {
			meta.PKField = field.Name
			meta.PKColumn = columnName
			continue
		}

		// Skip - Do not save
		if slices.Contains(parts[1:], "scanonly") {
			continue
		}

		// Skip - Do not save
		if strings.EqualFold(columnName, "fldLastUpdated") && slices.Contains(parts[1:], "default:current_timestamp") {
			continue
		}

		meta.Columns = append(meta.Columns, columnName)
	}

	if meta.TableName == "" || meta.PKColumn == "" {
		return nil
	}

	modelMetadataCache.Store(t, meta)

	return meta
}
