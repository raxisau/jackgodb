package jackgodb_test

import (
	"context"
	"testing"
	"time"

	"github.com/raxisau/jackgodb"
	"github.com/raxisau/jackgodb/internal/demo"
)

func TestCRUDAndConverters(t *testing.T) {
	dao := jackgodb.OpenSQLiteDBConnection("")
	defer dao.Close()

	statusDao := demo.NewStatusesDAO(dao)

	ctx := context.Background()
	_, err := dao.Db.NewCreateTable().Model((*demo.Statuses)(nil)).IfNotExists().Exec(ctx)
	if err != nil {
		t.Fatalf("create table failed: %v", err)
	}

	// Insert
	rec := statusDao.NewStatuses()
	if !rec.SetValue("statusCode", "ACTIVE") {
		t.Fatalf("SetValue failed on insert")
	}
	id := rec.Save()
	if id == 0 {
		t.Fatalf("expected non-zero id after insert")
	}

	// Load
	loaded := statusDao.NewStatuses()
	loaded.Load(id)
	if loaded.StatusCode != "ACTIVE" {
		t.Fatalf("expected ACTIVE got %q", loaded.StatusCode)
	}

	// Update using SetValue
	if !loaded.SetValue("statusCode", "RETIRED") {
		t.Fatalf("SetValue failed on update")
	}
	rows := loaded.Save()
	if rows == 0 {
		t.Fatalf("expected rows affected > 0 on update")
	}

	// ToAPI
	m := loaded.ToAPI()
	if m["f_status_code"] != "RETIRED" {
		t.Fatalf("ToAPI expected RETIRED got %v", m["f_status_code"])
	}

	// FromAPI (update existing)
	payload := map[string]any{
		"id":            id,
		"f_status_code": "FROMAPI",
	}
	loaded.FromAPI(payload)
	if loaded.StatusCode != "FROMAPI" {
		t.Fatalf("FromAPI did not update statusCode")
	}

	// Delete
	deleted := loaded.Delete()
	if deleted == 0 {
		t.Fatalf("expected deleted row count > 0")
	}

	// Converter tests
	if s := jackgodb.AnyToString(123); s == "" {
		t.Fatalf("AnyToString failed on int->string conversion")
	}

	if i := jackgodb.AnyToInt64("42"); i != 42 {
		t.Fatalf("AnyToInt64 failed expected 42 got %d", i)
	}

	tm := time.Now().UTC().Format(time.RFC3339)
	if parsed, err := jackgodb.ParseUTCDateTime(tm); err != nil || parsed.IsZero() {
		t.Fatalf("ParseUTCDateTime failed: %v", err)
	}
}
