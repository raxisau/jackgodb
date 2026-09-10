package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"
)

type Column struct {
	Name         string
	GoName       string
	GoType       string
	IsPK         bool
	IsAuto       bool
	IsNullable   bool
	IsUnique     bool
	OriginalLine string
}

type Table struct {
	Name       string
	GoName     string
	Alias      string
	DAOName    string
	Columns    []Column
	PrimaryKey string
}

// go run ./cmd/modelgen/modelgen.go -file infer schema.sql
func main() {
	fileFlag := flag.String("file", "", "stdout, infer, or output filename")
	flag.Parse()

	input, err := readInput(flag.Args())
	if err != nil {
		panic(err)
	}

	table, err := parseSchema(input)
	if err != nil {
		panic(err)
	}

	output := generateModel(table)

	switch strings.ToLower(strings.TrimSpace(*fileFlag)) {
	case "", "stdout":
		fmt.Print(output)

	case "infer":
		filename := inferFileName(table.Name)

		if err := os.WriteFile(filename, []byte(output), 0644); err != nil {
			panic(err)
		}

		fmt.Printf("Generated %s\n", filename)

	default:
		filename := filepath.Clean(*fileFlag)

		if err := os.WriteFile(filename, []byte(output), 0644); err != nil {
			panic(err)
		}

		fmt.Printf("Generated %s\n", filename)
	}
}

func readInput(args []string) (string, error) {
	if len(args) > 0 {
		bytes, err := os.ReadFile(args[0])
		if err != nil {
			return "", err
		}

		return string(bytes), nil
	}

	var sb strings.Builder
	scanner := bufio.NewScanner(os.Stdin)

	for scanner.Scan() {
		sb.WriteString(scanner.Text())
		sb.WriteRune('\n')
	}

	return sb.String(), scanner.Err()
}

func parseSchema(schema string) (Table, error) {
	var table Table

	reTable := regexp.MustCompile("(?i)CREATE\\s+TABLE\\s+`?([a-zA-Z0-9_]+)`?")
	match := reTable.FindStringSubmatch(schema)

	if len(match) < 2 {
		return table, fmt.Errorf("could not find CREATE TABLE name")
	}

	table.Name = match[1]
	table.GoName = tableNameToGoName(table.Name)
	table.Alias = goNameToAlias(table.GoName)
	table.DAOName = table.GoName + "DAO"

	lines := strings.SplitSeq(schema, "\n")

	for rawLine := range lines {
		line := strings.TrimSpace(rawLine)
		line = strings.TrimSuffix(line, ",")

		upperLine := strings.ToUpper(line)

		switch {
		case strings.HasPrefix(line, "`"):
			col := parseColumn(line)
			if col.Name != "" {
				table.Columns = append(table.Columns, col)
			}

		case strings.HasPrefix(upperLine, "PRIMARY KEY"):
			table.PrimaryKey = extractFirstBacktickValue(line)

		case strings.HasPrefix(upperLine, "UNIQUE KEY"):
			uniqueCol := extractLastBacktickValue(line)

			for i := range table.Columns {
				if table.Columns[i].Name == uniqueCol {
					table.Columns[i].IsUnique = true
				}
			}
		}
	}

	for i := range table.Columns {
		if table.Columns[i].Name == table.PrimaryKey {
			table.Columns[i].IsPK = true
		}
	}

	return table, nil
}

func parseColumn(line string) Column {
	var col Column
	col.OriginalLine = line

	re := regexp.MustCompile("^`([^`]+)`\\s+([^\\s]+)(.*)$")
	match := re.FindStringSubmatch(line)

	if len(match) < 4 {
		return col
	}

	col.Name = match[1]

	sqlType := strings.ToLower(match[2])
	rest := strings.ToLower(match[3])

	col.GoName = columnNameToGoName(col.Name)
	col.IsNullable = !strings.Contains(rest, "not null")
	col.IsAuto = strings.Contains(rest, "auto_increment")

	col.GoType = sqlTypeToGoType(sqlType, col.IsNullable)
	if strings.EqualFold(col.Name, "fldLastUpdated") ||
		strings.EqualFold(col.Name, "f_updated_at") {
		col.GoType = "time.Time"
	}

	return col
}

func generateModel(table Table) string {
	var sb strings.Builder

	needsTime := false
	for _, col := range table.Columns {
		if strings.Contains(col.GoType, "time.Time") {
			needsTime = true
			break
		}
	}

	sb.WriteString("package model\n\n")

	if needsTime {
		sb.WriteString("import (\n")
		sb.WriteString("\t\"time\"\n\n")
		sb.WriteString("\t\"github.com/uptrace/bun\"\n")
		sb.WriteString(")\n\n")
	} else {
		sb.WriteString("import (\n")
		sb.WriteString("\t\"github.com/uptrace/bun\"\n")
		sb.WriteString(")\n\n")
	}

	fmt.Fprintf(&sb, "type %s struct {\n", table.DAOName)
	fmt.Fprintf(&sb, "\t*ModelDAO[*%s]\n", table.GoName)
	sb.WriteString("}\n\n")

	fmt.Fprintf(&sb, "type %s struct {\n", table.GoName)
	fmt.Fprintf(&sb, "\tbun.BaseModel `bun:\"table:%s,alias:%s\"`\n\n", table.Name, table.Alias)
	fmt.Fprintf(&sb, "\tModelRecord[*%s] `bun:\"-\" json:\"-\"`\n\n", table.GoName)

	pk := primaryKeyColumn(table)

	for _, col := range table.Columns {
		tag := buildBunTag(col)

		goCol := col.GoName
		if pk.GoName == col.GoName {
			goCol = "ID"
		}

		jsonOmit := "omitempty"
		if col.GoType == "time.Time" {
			jsonOmit = "omitzero"
		}
		fmt.Fprintf(&sb, "\t%-22s %-12s `bun:\"%s\" json:\"%s,%s\"`\n", goCol, col.GoType, tag, col.Name, jsonOmit)
	}

	sb.WriteString("}\n\n")
	fmt.Fprintf(&sb, "var alias%s = map[string]string{\n", table.GoName)
	for _, col := range table.Columns {
		goCol := col.GoName
		if pk.GoName == col.GoName {
			goCol = "ID"
		}
		alias := lowerFirstOrAll(goCol)
		fmt.Fprintf(&sb, "\t\"%s\": \"%s\",\n", alias, col.Name)
	}
	fmt.Fprintf(&sb, "}\n")

	fmt.Fprintf(&sb, "func (dao *DAO) New%sDAO() *%s {\n", table.GoName, table.DAOName)

	fmt.Fprintf(&sb, "\treturn &%s{\n", table.DAOName)
	fmt.Fprintf(&sb, "\t\tModelDAO: NewModelDAO(\n")
	fmt.Fprintf(&sb, "\t\t\tdao,\n")
	fmt.Fprintf(&sb, "\t\t\talias%s,\n", table.GoName)
	fmt.Fprintf(&sb, "\t\t\tfunc() *%s {\n", table.GoName)
	fmt.Fprintf(&sb, "\t\t\t\treturn &%s{}\n", table.GoName)
	sb.WriteString("\t\t\t}),\n")
	sb.WriteString("\t}\n")
	sb.WriteString("}\n\n")

	fmt.Fprintf(&sb, "func (dao *%s) New%s() *%s {\n", table.DAOName, table.GoName, table.GoName)
	sb.WriteString("\trecord := dao.New()\n")
	sb.WriteString("\treturn record\n")
	sb.WriteString("}\n")

	return sb.String()
}
func lowerFirstOrAll(s string) string {
	if s == "" {
		return s
	}

	// If entire string is uppercase, return all lowercase
	if s == strings.ToUpper(s) {
		return strings.ToLower(s)
	}

	// Lowercase only the first rune
	runes := []rune(s)
	runes[0] = unicode.ToLower(runes[0])

	return string(runes)
}
func buildBunTag(col Column) string {
	parts := []string{col.Name}

	if col.IsPK {
		parts = append(parts, "pk")
	}

	if col.IsAuto {
		parts = append(parts, "autoincrement")
	}

	if col.IsUnique {
		parts = append(parts, "unique")
	}

	if strings.EqualFold(col.Name, "fldLastUpdated") ||
		strings.EqualFold(col.Name, "f_updated_at") {
		parts = append(parts, "nullzero")
		parts = append(parts, "notnull")
		parts = append(parts, "default:current_timestamp")
	}

	return strings.Join(parts, ",")
}

func primaryKeyColumn(table Table) *Column {
	for i := range table.Columns {
		if table.Columns[i].IsPK {
			return &table.Columns[i]
		}
	}

	return nil
}

func sqlTypeToGoType(sqlType string, nullable bool) string {
	switch {
	case strings.Contains(sqlType, "tinyint(1)"):
		if nullable {
			return "*bool"
		}
		return "bool"

	case strings.Contains(sqlType, "int"):
		if nullable {
			return "*int64"
		}
		return "int64"

	case strings.Contains(sqlType, "decimal"),
		strings.Contains(sqlType, "float"),
		strings.Contains(sqlType, "double"):
		if nullable {
			return "*float64"
		}
		return "float64"

	case strings.Contains(sqlType, "date"),
		strings.Contains(sqlType, "timestamp"),
		strings.Contains(sqlType, "datetime"):
		if nullable {
			return "*time.Time"
		}
		return "time.Time"

	case strings.Contains(sqlType, "text"),
		strings.Contains(sqlType, "char"),
		strings.Contains(sqlType, "varchar"),
		strings.Contains(sqlType, "mediumtext"),
		strings.Contains(sqlType, "longtext"),
		strings.Contains(sqlType, "json"):
		if nullable {
			return "*string"
		}
		return "string"

	default:
		if nullable {
			return "*string"
		}
		return "string"
	}
}

func tableNameToGoName(name string) string {
	name = strings.TrimPrefix(name, "tbl")
	name = strings.TrimPrefix(name, "SD")

	return toPascalCase(name)
}
func goNameToAlias(name string) string {
	var result []rune

	for _, r := range name {
		if unicode.IsUpper(r) {
			result = append(result, r)
		}
	}

	if len(result) == 0 {
		return name
	}

	return string(result)
}

func columnNameToGoName(name string) string {
	name = strings.TrimPrefix(name, "fld")
	name = strings.TrimPrefix(name, "f_")

	return toPascalCase(name)
}

func toPascalCase(s string) string {
	parts := regexp.MustCompile(`[^a-zA-Z0-9]+`).Split(s, -1)

	var out strings.Builder

	for _, part := range parts {
		if part == "" {
			continue
		}

		runes := []rune(part)

		for i, r := range runes {
			if i == 0 {
				out.WriteRune(unicode.ToUpper(r))
			} else {
				out.WriteRune(r)
			}
		}
	}

	result := out.String()

	replacements := map[string]string{
		"Id":   "ID",
		"Acn":  "ACN",
		"Abn":  "ABN",
		"Url":  "URL",
		"Json": "JSON",
		"Html": "HTML",
		"Asic": "ASIC",
	}

	for old, newValue := range replacements {
		result = strings.ReplaceAll(result, old, newValue)
	}

	return result
}

func inferFileName(tableName string) string {
	goName := tableNameToGoName(tableName)

	var result []rune

	for i, r := range goName {
		if i > 0 && unicode.IsUpper(r) {
			result = append(result, '_')
		}

		result = append(result, unicode.ToLower(r))
	}

	return string(result) + ".go"
}

func extractFirstBacktickValue(s string) string {
	re := regexp.MustCompile("`([^`]+)`")
	match := re.FindStringSubmatch(s)

	if len(match) < 2 {
		return ""
	}

	return match[1]
}

func extractLastBacktickValue(s string) string {
	re := regexp.MustCompile("`([^`]+)`")
	matches := re.FindAllStringSubmatch(s, -1)

	if len(matches) == 0 {
		return ""
	}

	return matches[len(matches)-1][1]
}
