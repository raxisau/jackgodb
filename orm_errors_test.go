package jackgodb_test

import (
	"context"
	"errors"
	"testing"

	"github.com/raxisau/jackgodb"
	"github.com/raxisau/jackgodb/internal/demo"
)

func setupErrDAO(t *testing.T) (*jackgodb.DAO, *demo.StatusesDAO) {
	t.Helper()
	dao := jackgodb.OpenSQLiteDBConnection("")
	t.Cleanup(func() { dao.Close() })
	if _, err := dao.Db.NewCreateTable().Model((*demo.Statuses)(nil)).IfNotExists().Exec(context.Background()); err != nil {
		t.Fatalf("create table: %v", err)
	}
	return dao, demo.NewStatusesDAO(dao)
}

func TestInsertE_InsertsAndReturnsID(t *testing.T) {
	_, sDao := setupErrDAO(t)

	rec := sDao.NewStatuses()
	rec.SetValue("statusCode", "ACTIVE")
	id, err := rec.InsertE()
	if err != nil {
		t.Fatalf("InsertE returned error: %v", err)
	}
	if id == 0 {
		t.Fatal("expected non-zero id")
	}
}

func TestInsertE_DuplicateUniqueReturnsError(t *testing.T) {
	_, sDao := setupErrDAO(t)

	first := sDao.NewStatuses()
	first.SetValue("statusCode", "DUP")
	if _, err := first.InsertE(); err != nil {
		t.Fatalf("first insert failed: %v", err)
	}

	dup := sDao.NewStatuses()
	dup.SetValue("statusCode", "DUP")
	id, err := dup.InsertE()
	if err == nil {
		t.Fatalf("expected error on duplicate unique value, got id %d", id)
	}
}

func TestUpdateE_CleanRecordReturnsErrNoChanges(t *testing.T) {
	_, sDao := setupErrDAO(t)

	rec := sDao.NewStatuses()
	rec.SetID(1)
	rows, err := rec.UpdateE()
	if !errors.Is(err, jackgodb.ErrNoChanges) {
		t.Fatalf("expected ErrNoChanges, got %v", err)
	}
	if rows != 0 {
		t.Errorf("expected 0 rows, got %d", rows)
	}
}

func TestUpdateE_PersistsDirtyColumns(t *testing.T) {
	dao, sDao := setupErrDAO(t)

	rec := sDao.NewStatuses()
	rec.SetValue("statusCode", "BEFORE")
	id, err := rec.InsertE()
	if err != nil {
		t.Fatalf("insert: %v", err)
	}

	rec.SetValue("statusCode", "AFTER")
	rows, err := rec.UpdateE()
	if err != nil {
		t.Fatalf("UpdateE returned error: %v", err)
	}
	if rows != 1 {
		t.Errorf("expected 1 row, got %d", rows)
	}

	var code string
	if err := dao.Db.NewRaw("SELECT f_status_code FROM reg_statuses WHERE id=?", id).Scan(context.Background(), &code); err != nil {
		t.Fatalf("read back: %v", err)
	}
	if code != "AFTER" {
		t.Errorf("expected AFTER in db, got %s", code)
	}
}

func TestLoadE_MissingIDReturnsErrNotFound(t *testing.T) {
	_, sDao := setupErrDAO(t)

	rec := sDao.NewStatuses()
	_, err := rec.LoadE(999)
	if !errors.Is(err, jackgodb.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestLoadE_FoundReturnsRecord(t *testing.T) {
	_, sDao := setupErrDAO(t)

	rec := sDao.NewStatuses()
	rec.SetValue("statusCode", "FOUND")
	id, err := rec.InsertE()
	if err != nil {
		t.Fatalf("insert: %v", err)
	}

	got, err := sDao.NewStatuses().LoadE(id)
	if err != nil {
		t.Fatalf("LoadE returned error: %v", err)
	}
	if got.StatusCode != "FOUND" {
		t.Errorf("expected FOUND, got %q", got.StatusCode)
	}
}

func TestSaveE_RoutesInsertAndUpdate(t *testing.T) {
	_, sDao := setupErrDAO(t)

	rec := sDao.NewStatuses()
	rec.SetValue("statusCode", "NEW")
	id, err := rec.SaveE()
	if err != nil || id == 0 {
		t.Fatalf("SaveE insert: id=%d err=%v", id, err)
	}

	rec.SetValue("statusCode", "CHANGED")
	if _, err := rec.SaveE(); err != nil {
		t.Fatalf("SaveE update: %v", err)
	}
}

func TestInsert_DuplicateReturnsZeroNotSilentIgnore(t *testing.T) {
	_, sDao := setupErrDAO(t)

	first := sDao.NewStatuses()
	first.SetValue("statusCode", "DUP")
	if id := first.Insert(); id == 0 {
		t.Fatal("first insert returned 0")
	}

	dup := sDao.NewStatuses()
	dup.SetValue("statusCode", "DUP")
	if id := dup.Insert(); id != 0 {
		t.Fatalf("duplicate Insert should return 0, got %d", id)
	}
}

func TestInsertIgnore_DuplicateLeavesExistingRow(t *testing.T) {
	dao, sDao := setupErrDAO(t)

	first := sDao.NewStatuses()
	first.SetValue("statusCode", "KEEP")
	first.Insert()

	dup := sDao.NewStatuses()
	dup.SetValue("statusCode", "KEEP")
	dup.InsertIgnore()

	var n int
	if err := dao.Db.NewRaw("SELECT COUNT(*) FROM reg_statuses WHERE f_status_code='KEEP'").Scan(context.Background(), &n); err != nil {
		t.Fatalf("count: %v", err)
	}
	if n != 1 {
		t.Errorf("expected 1 row after InsertIgnore, got %d", n)
	}
}
