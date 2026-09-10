package model

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"
)

var (
	StringType = reflect.TypeFor[string]()
	TimeType   = reflect.TypeFor[time.Time]()
)

func ConvertTo(value any, targetType reflect.Type) (any, error) {

	// nullable string
	kind := targetType.Kind()
	if kind == reflect.Pointer {

		if targetType.Elem().Kind() == reflect.String {
			switch v := value.(type) {
			case string:
				s := v
				return &s, nil
			case *string:
				return value, nil
			default:
				s := fmt.Sprintf("%v", v)
				return &s, nil
			}
		}

		// nullable int64
		if targetType.Elem().Kind() == reflect.Int64 {
			switch v := value.(type) {
			case *int64:
				return v, nil
			case int64:
				return &v, nil
			default:
				n := AnyToInt64(v)
				return &n, nil
			}
		}

		// nullable time.Time
		if targetType.Elem() == TimeType {

			switch v := value.(type) {
			case time.Time:
				return &v, nil
			case *time.Time:
				return v, nil
			case string:
				return ParseUTCDateTime(v)
			}
		}
	}

	// int
	if kind == reflect.Int {
		n := AnyToInt(value)
		return n, nil
	}

	// int64
	if kind == reflect.Int64 {
		n := AnyToInt64(value)
		return n, nil
	}

	// string
	if kind == reflect.String {
		return fmt.Sprintf("%v", value), nil
	}

	// time.Time
	if targetType == TimeType {
		switch v := value.(type) {
		case time.Time:
			return v, nil
		case string:
			return ParseUTCDateTime(v)
		}
	}

	return nil, fmt.Errorf("unsupported type %s", targetType.String())
}
func ParseUTCDateTime(value string) (time.Time, error) {
	formats := []string{
		"2006-01-02 15:04:05",
		"2006-01-02 15:04",
		"2006-01-02",
		time.RFC3339,
		time.RFC1123Z,
		time.RFC3339Nano,
		time.UnixDate,
		time.RubyDate,
		time.RFC1123,
		time.RFC822,
		time.RFC850,
		time.RFC822Z,
		"2006/01/02",
		"2006/01/02 15:04:05",
	}

	for _, format := range formats {
		t, err := time.ParseInLocation(format, value, time.UTC)
		if err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("invalid UTC date/time: %q", value)
}
func AnyToTime(value any) time.Time {
	resultTime, err := ConvertTo(value, TimeType)
	if err != nil {
		return time.Now().UTC()
	}
	return resultTime.(time.Time)
}
func AnyToInt(v any) int {
	val := AnyToInt64(v)
	return int(val)
}

func AnyToInt64(v any) int64 {
	switch t := v.(type) {
	case int:
		return int64(t)
	case int8:
		return int64(t)
	case int16:
		return int64(t)
	case int32:
		return int64(t)
	case int64:
		return t

	case uint:
		return int64(t)
	case uint8:
		return int64(t)
	case uint16:
		return int64(t)
	case uint32:
		return int64(t)
	case uint64:
		return int64(t)

	case float32:
		return int64(t)
	case float64:
		return int64(t)

	case string:
		n, err := strconv.ParseInt(strings.TrimSpace(t), 10, 64)
		if err != nil {
			return 0
		}
		return n

	default:
		return 0
	}
}
func AnyToString(value any) string {
	resultString, err := ConvertTo(value, StringType)
	if err != nil {
		return ""
	}
	return resultString.(string)
}

func StringValue(value *string) string {
	if value == nil {
		return ""
	}

	return *value
}
