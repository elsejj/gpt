package llm

import (
	"log/slog"
	"reflect"
	"strconv"
	"strings"
)

var zeroValue = reflect.Value{}

func MGet[V any](m any, key string, defaultValue V) V {

	fields := strings.FieldsFunc(key, func(r rune) bool { return r == '.' })

	d := reflect.ValueOf(m)

	for _, f := range fields {
		v, ok := mGet(d, f)
		if !ok {
			return defaultValue
		}
		d = v
	}
	if d.IsValid() {
		if d.CanInterface() {
			vv, ok := d.Interface().(V)
			if ok {
				return vv
			}
		}
	}
	return defaultValue
}

func mGet(d reflect.Value, key string) (reflect.Value, bool) {
	if d.CanInterface() && d.Interface() == nil {
		return zeroValue, false
	}

	switch d.Kind() {
	case reflect.Map:
		v := d.MapIndex(reflect.ValueOf(key))
		if !v.IsValid() {
			return zeroValue, false
		}
		if v.Kind() == reflect.Interface {
			if v.IsNil() {
				return zeroValue, false
			}
			v = v.Elem()
		}
		return v, true
	case reflect.Slice:
		idx, err := strconv.Atoi(key)
		if err != nil {
			return zeroValue, false
		}
		if idx < 0 || idx >= d.Len() {
			return zeroValue, false
		}
		v := d.Index(idx)
		if v.IsValid() {
			return v, true
		}
		return zeroValue, false
	case reflect.Struct:
		v := d.FieldByName(key)
		if v.IsValid() {
			return v, true
		}
		return zeroValue, false
	case reflect.Pointer:
		return mGet(d.Elem(), key)
	case reflect.Interface:
		return mGet(reflect.ValueOf(d.Interface()), key)
	default:
		slog.Warn("unsupported kind", "kind", d.Kind(), "key", key)
		return zeroValue, false
	}
}
