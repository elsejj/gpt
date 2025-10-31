package utils

import "reflect"

func isEmpty(v any) bool {
	switch val := v.(type) {
	case nil:
		return true
	case string:
		return val == ""
	case []string:
		return len(val) == 0
	case []byte:
		return len(val) == 0
	case int, int8, int16, int32, int64:
		return val == 0
	case uint, uint8, uint16, uint32, uint64:
		return val == 0
	case float32, float64:
		return val == 0.0
	case bool:
		return val == false
	case []int:
		return len(val) == 0
	case []float64:
		return len(val) == 0
	case []bool:
		return len(val) == 0
	default:
		t := reflect.TypeOf(v)
		switch t.Kind() {
		case reflect.Slice, reflect.Array, reflect.Map:
			rv := reflect.ValueOf(v)
			return rv.Len() == 0
		case reflect.Ptr:
			val := reflect.ValueOf(v)
			if val.IsNil() {
				return true
			}
		}
		return v == nil
	}
}

func Or[T any](candidates ...T) T {
	for _, candidate := range candidates {
		if !isEmpty(candidate) {
			return candidate
		}
	}
	var zero T
	return zero
}
