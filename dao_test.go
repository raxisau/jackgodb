package jackgodb_test

import (
	"testing"

	"github.com/raxisau/jackgodb"
	"github.com/raxisau/jackgodb/internal/demo"
)

func TestDbInAndAttach(t *testing.T) {
	dao := jackgodb.OpenSQLiteDBConnection("")
	defer dao.Close()

	// DbIn for sqlite should use "?" placeholders
	in := dao.DbIn(1, 3)
	if in != "?,?,?" {
		t.Fatalf("expected '?,?,?' got %q", in)
	}

	statusDao := demo.NewStatusesDAO(dao)

	// Create an unattached record and attach it using Attach
	var raw demo.Statuses
	// Ensure it's not attached until we call Attach
	statusDao.Attach(&raw)

	// Now the attached record should respond to SetValue
	ok := raw.SetValue("statusCode", "ATTACH_TEST")
	if !ok {
		t.Fatalf("expected SetValue to succeed after Attach")
	}

	// AttachList should not panic and should attach each record
	var list [](*demo.Statuses)
	r1 := &demo.Statuses{}
	r2 := &demo.Statuses{}
	list = append(list, r1, r2)
	statusDao.AttachList(list)

	if !r1.SetValue("statusCode", "L1") {
		t.Fatalf("expected SetValue on attached list item r1")
	}
	if !r2.SetValue("statusCode", "L2") {
		t.Fatalf("expected SetValue on attached list item r2")
	}
}
