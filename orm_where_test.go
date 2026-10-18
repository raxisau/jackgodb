package jackgodb_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/raxisau/jackgodb"
	"github.com/raxisau/jackgodb/internal/demo"
	"github.com/uptrace/bun"
)

func setupWhereDAO(t *testing.T) (*jackgodb.DAO, *demo.StatusesDAO) {
	t.Helper()
	dao := jackgodb.OpenSQLiteDBConnection("")
	t.Cleanup(func() { dao.Close() })
	if _, err := dao.Db.NewCreateTable().Model((*demo.Statuses)(nil)).IfNotExists().Exec(context.Background()); err != nil {
		t.Fatalf("create table: %v", err)
	}
	return dao, demo.NewStatusesDAO(dao)
}

func selectFor(dao *jackgodb.DAO) *bun.SelectQuery {
	return dao.Db.NewSelect().Model((*demo.Statuses)(nil))
}

func TestToWhereE_AliasKeyResolves(t *testing.T) {
	dao, sDao := setupWhereDAO(t)
	rec := sDao.NewStatuses()
	q, err := rec.ToWhereE(selectFor(dao), map[string]any{"where": map[string]any{"statusCode": "ACTIVE"}})
	if err != nil {
		t.Fatalf("alias key rejected: %v", err)
	}
	if !strings.Contains(q.String(), "f_status_code = 'ACTIVE'") {
		t.Errorf("expected alias resolved to f_status_code, got %s", q.String())
	}
}

func TestToWhereE_RawColumnKeyAccepted(t *testing.T) {
	dao, sDao := setupWhereDAO(t)
	rec := sDao.NewStatuses()
	if _, err := rec.ToWhereE(selectFor(dao), map[string]any{"where": map[string]any{"f_status_code": "X"}}); err != nil {
		t.Fatalf("real column key rejected: %v", err)
	}
}

func TestToWhereE_TypoKeyRejected(t *testing.T) {
	dao, sDao := setupWhereDAO(t)
	rec := sDao.NewStatuses()
	_, err := rec.ToWhereE(selectFor(dao), map[string]any{"where": map[string]any{"statuscod": "X"}})
	if !errors.Is(err, jackgodb.ErrUnknownKey) {
		t.Fatalf("expected ErrUnknownKey for typo, got %v", err)
	}
}

func TestToWhereE_InjectionKeyRejected(t *testing.T) {
	dao, sDao := setupWhereDAO(t)
	rec := sDao.NewStatuses()
	q, err := rec.ToWhereE(selectFor(dao), map[string]any{"where": map[string]any{"statusCode = 1 OR 1=1 --": "X"}})
	if !errors.Is(err, jackgodb.ErrUnknownKey) {
		t.Fatalf("expected ErrUnknownKey for injection key, got %v", err)
	}
	if strings.Contains(q.String(), "OR 1=1") {
		t.Errorf("injected SQL reached the query: %s", q.String())
	}
}

func TestToWhereE_OrderValidated(t *testing.T) {
	dao, sDao := setupWhereDAO(t)
	rec := sDao.NewStatuses()
	if _, err := rec.ToWhereE(selectFor(dao), map[string]any{"order": map[string]any{"nope": "asc"}}); !errors.Is(err, jackgodb.ErrUnknownKey) {
		t.Fatalf("expected ErrUnknownKey on bad order key, got %v", err)
	}
	q, err := rec.ToWhereE(selectFor(dao), map[string]any{"order": map[string]any{"createdAt": "desc"}})
	if err != nil {
		t.Fatalf("valid order rejected: %v", err)
	}
	if !strings.Contains(q.String(), "ORDER BY \"f_created_at\" DESC") {
		t.Errorf("expected ORDER BY f_created_at DESC, got %s", q.String())
	}
}

func TestToWhereE_LimitsListAndScalar(t *testing.T) {
	dao, sDao := setupWhereDAO(t)
	rec := sDao.NewStatuses()
	q, err := rec.ToWhereE(selectFor(dao), map[string]any{"limits": []any{int64(10), int64(5)}})
	if err != nil {
		t.Fatalf("limits list rejected: %v", err)
	}
	if !strings.Contains(q.String(), "LIMIT 5") || !strings.Contains(q.String(), "OFFSET 10") {
		t.Errorf("expected LIMIT 5 OFFSET 10, got %s", q.String())
	}
	q, _ = rec.ToWhereE(selectFor(dao), map[string]any{"limits": int64(3)})
	if !strings.Contains(q.String(), "LIMIT 3") {
		t.Errorf("expected LIMIT 3, got %s", q.String())
	}
}

func TestToWhereE_LimitsBadShapeRejected(t *testing.T) {
	dao, sDao := setupWhereDAO(t)
	rec := sDao.NewStatuses()
	if _, err := rec.ToWhereE(selectFor(dao), map[string]any{"limits": []any{int64(1)}}); !errors.Is(err, jackgodb.ErrBadArgs) {
		t.Fatalf("expected ErrBadArgs, got %v", err)
	}
}

func TestToWhereE_WhereMustBeMap(t *testing.T) {
	dao, sDao := setupWhereDAO(t)
	rec := sDao.NewStatuses()
	if _, err := rec.ToWhereE(selectFor(dao), map[string]any{"where": "statusCode"}); !errors.Is(err, jackgodb.ErrBadArgs) {
		t.Fatalf("expected ErrBadArgs for non-map where, got %v", err)
	}
}

func TestToWhere_WrapperReturnsQueryOnError(t *testing.T) {
	dao, sDao := setupWhereDAO(t)
	rec := sDao.NewStatuses()
	q := rec.ToWhere(selectFor(dao), map[string]any{"where": map[string]any{"bogus": "x"}})
	if q == nil {
		t.Fatal("ToWhere returned nil query on error")
	}
}
