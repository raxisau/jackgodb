package jackgodb

import (
	"context"
	"testing"

	"github.com/uptrace/bun"
)

type dirtyProbe struct {
	bun.BaseModel `bun:"table:dirty_probe,alias:DP"`

	ModelRecord[*dirtyProbe] `bun:"-" json:"-"`

	ID    int64  `bun:"id,pk,autoincrement" json:"id,omitempty"`
	Label string `bun:"f_label" json:"f_label,omitempty"`
}

func TestDirtyAll_MarksEveryNonPKColumn(t *testing.T) {
	dao := OpenSQLiteDBConnection("")
	defer dao.Close()
	if _, err := dao.Db.NewCreateTable().Model((*dirtyProbe)(nil)).IfNotExists().Exec(context.Background()); err != nil {
		t.Fatalf("create table: %v", err)
	}
	dpDao := NewModelDAO(dao, map[string]string{"label": "f_label"}, func() *dirtyProbe { return &dirtyProbe{} })
	rec := dpDao.New()
	rec.SetID(7)
	rec.Clean()

	rec.DirtyAll()

	if !rec.IsDirty() {
		t.Fatal("expected record dirty after DirtyAll")
	}
	if !rec.dirty["f_label"] {
		t.Error("expected f_label dirty")
	}
	if rec.dirty["id"] {
		t.Error("primary key must not be marked dirty")
	}
}
