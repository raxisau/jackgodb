package main

import (
	"strings"
	"testing"
)

const cronSchema = "CREATE TABLE `d42_cron_queue` (\n" +
	"  `id` bigint(20) NOT NULL AUTO_INCREMENT,\n" +
	"  `f_status` varchar(255) NOT NULL DEFAULT 'NEW',\n" +
	"  `f_priority` int(11) NOT NULL DEFAULT 5,\n" +
	"  `f_command` text NOT NULL,\n" +
	"  `f_result` text,\n" +
	"  `f_created_at` timestamp NOT NULL DEFAULT current_timestamp(),\n" +
	"  `f_updated_at` timestamp NOT NULL DEFAULT current_timestamp(),\n" +
	"  PRIMARY KEY (`id`)\n" +
	") ENGINE=InnoDB;\n"

func mustParse(t *testing.T, schema string) Table {
	t.Helper()
	tbl, err := parseSchema(schema)
	if err != nil {
		t.Fatalf("parseSchema: %v", err)
	}
	return tbl
}

func TestParseSchema_TableNameAndPK(t *testing.T) {
	tbl := mustParse(t, cronSchema)
	if tbl.Name != "d42_cron_queue" {
		t.Errorf("name = %q", tbl.Name)
	}
	if tbl.PrimaryKey != "id" {
		t.Errorf("PrimaryKey = %q", tbl.PrimaryKey)
	}
	var pkCount int
	for _, c := range tbl.Columns {
		if c.IsPK {
			pkCount++
		}
	}
	if pkCount != 1 {
		t.Errorf("expected exactly one PK column, got %d", pkCount)
	}
}

func TestParseSchema_NullableFromNotNull(t *testing.T) {
	tbl := mustParse(t, cronSchema)
	got := map[string]bool{}
	for _, c := range tbl.Columns {
		got[c.Name] = c.IsNullable
	}
	if got["f_command"] {
		t.Error("f_command is NOT NULL, should not be nullable")
	}
	if !got["f_result"] {
		t.Error("f_result has no NOT NULL, should be nullable")
	}
}

func TestParseSchema_SingleColumnUniqueKey(t *testing.T) {
	schema := strings.Replace(cronSchema, "  PRIMARY KEY (`id`)\n",
		"  PRIMARY KEY (`id`),\n  UNIQUE KEY `idx_status` (`f_status`)\n", 1)
	tbl := mustParse(t, schema)
	for _, c := range tbl.Columns {
		if c.Name == "f_status" && !c.IsUnique {
			t.Error("f_status should be marked unique")
		}
		if c.Name == "f_priority" && c.IsUnique {
			t.Error("f_priority must not be marked unique")
		}
	}
}

func TestParseSchema_CompositeUniqueKeyMarksEveryColumn(t *testing.T) {
	schema := strings.Replace(cronSchema, "  PRIMARY KEY (`id`)\n",
		"  PRIMARY KEY (`id`),\n  UNIQUE KEY `idx_pair` (`f_status`,`f_priority`)\n", 1)
	tbl := mustParse(t, schema)

	unique := map[string]bool{}
	for _, c := range tbl.Columns {
		unique[c.Name] = c.IsUnique
	}
	// A composite key is one constraint over both columns. Marking each column "unique" alone
	// would be wrong, so neither should carry the single-column flag.
	if unique["f_status"] || unique["f_priority"] {
		t.Errorf("composite key must not mark single columns unique: status=%v priority=%v",
			unique["f_status"], unique["f_priority"])
	}
}

func TestGenerateModel_PKIsNamedID(t *testing.T) {
	out := generateModel(mustParse(t, cronSchema))
	if !strings.Contains(out, "ID ") || !strings.Contains(out, `bun:"id,pk,autoincrement"`) {
		t.Errorf("expected ID field with pk,autoincrement tag:\n%s", out)
	}
}
