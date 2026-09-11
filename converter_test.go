package jackgodb_test

import (
	"reflect"
	"testing"
	"time"

	"github.com/raxisau/jackgodb"
)

func TestConvertToAndHelpers(t *testing.T) {
	// nullable string
	t1 := "hello"
	v, err := jackgodb.ConvertTo("hello", reflect.TypeFor[*string]())
	if err != nil {
		t.Fatalf("ConvertTo(*string) error: %v", err)
	}
	ps, ok := v.(*string)
	if !ok || ps == nil || *ps != "hello" {
		t.Fatalf("ConvertTo(*string) expected *\"hello\" got %#v", v)
	}

	// nullable int64
	v, err = jackgodb.ConvertTo(42, reflect.TypeFor[*int64]())
	if err != nil {
		t.Fatalf("ConvertTo(*int64) error: %v", err)
	}
	pi, ok := v.(*int64)
	if !ok || pi == nil || *pi != 42 {
		t.Fatalf("ConvertTo(*int64) expected 42 got %#v", v)
	}

	// nullable time.Time from string
	now := time.Now().UTC()
	s := now.Format(time.RFC3339)
	v, err = jackgodb.ConvertTo(s, reflect.TypeFor[*time.Time]())
	if err != nil {
		t.Fatalf("ConvertTo(*time.Time) error: %v", err)
	}
	pt, ok := v.(*time.Time)
	if !ok || pt == nil {
		t.Fatalf("ConvertTo(*time.Time) expected *time.Time got %#v", v)
	}

	// int
	v, err = jackgodb.ConvertTo("123", reflect.TypeFor[int]())
	if err != nil {
		t.Fatalf("ConvertTo(int) error: %v", err)
	}
	if vi, ok := v.(int); !ok || vi != 123 {
		t.Fatalf("ConvertTo(int) expected 123 got %#v", v)
	}

	v, err = jackgodb.ConvertTo(123, reflect.TypeFor[int]())
	if err != nil {
		t.Fatalf("ConvertTo(int) error: %v", err)
	}
	if vi, ok := v.(int); !ok || vi != 123 {
		t.Fatalf("ConvertTo(int) expected 123 got %#v", v)
	}

	// int64
	v, err = jackgodb.ConvertTo("321", reflect.TypeFor[int64]())
	if err != nil {
		t.Fatalf("ConvertTo(int64) error: %v", err)
	}
	if vi, ok := v.(int64); !ok || vi != 321 {
		t.Fatalf("ConvertTo(int64) expected 321 got %#v", v)
	}

	// string
	v, err = jackgodb.ConvertTo(3.14, reflect.TypeFor[string]())
	if err != nil {
		t.Fatalf("ConvertTo(string) error: %v", err)
	}
	if vs, ok := v.(string); !ok || vs == "" {
		t.Fatalf("ConvertTo(string) expected non-empty got %#v", v)
	}

	// time.Time target
	v, err = jackgodb.ConvertTo(s, reflect.TypeFor[time.Time]())
	if err != nil {
		t.Fatalf("ConvertTo(time.Time) error: %v", err)
	}
	if vt, ok := v.(time.Time); !ok || vt.IsZero() {
		t.Fatalf("ConvertTo(time.Time) expected parsed time got %#v", v)
	}

	// unsupported type
	_, err = jackgodb.ConvertTo("x", reflect.TypeOf(struct{ X int }{}))
	if err == nil {
		t.Fatalf("ConvertTo expected error for unsupported type")
	}

	// AnyToInt64
	if jackgodb.AnyToInt64("42") != 42 {
		t.Fatalf("AnyToInt64(\"42\") expected 42")
	}
	if jackgodb.AnyToInt64(7) != 7 {
		t.Fatalf("AnyToInt64(7) expected 7")
	}

	// AnyToString
	if s := jackgodb.AnyToString(99); s == "" {
		t.Fatalf("AnyToString(99) expected non-empty")
	}

	// StringValue
	var pnil *string
	if jackgodb.StringValue(pnil) != "" {
		t.Fatalf("StringValue(nil) expected empty string")
	}
	if jackgodb.StringValue(&t1) != "hello" {
		t.Fatalf("StringValue pointer expected 'hello'")
	}

	// AnyToTime fallback: invalid input returns time near now
	before := time.Now().UTC()
	at := jackgodb.AnyToTime("not-a-date")
	after := time.Now().UTC()
	if at.Before(before.Add(-time.Second*1)) || at.After(after.Add(time.Second*5)) {
		t.Fatalf("AnyToTime fallback out of expected range: %v (between %v and %v)", at, before, after)
	}
}
