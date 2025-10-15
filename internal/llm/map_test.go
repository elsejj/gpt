package llm

import "testing"

func TestMGetReturnsValueFromMap(t *testing.T) {
	data := map[string]any{
		"foo": "bar",
		"iface": any(map[string]any{
			"inner": "value",
		}),
	}

	if got := MGet(data, "foo", ""); got != "bar" {
		t.Fatalf("expected %q, got %q", "bar", got)
	}

	if got := MGet(data, "iface.inner", ""); got != "value" {
		t.Fatalf("expected %q, got %q", "value", got)
	}

	if got := MGet(data, "foo", 42); got != 42 {
		t.Fatalf("expected default %d for type mismatch, got %d", 42, got)
	}
}

func TestMGetTraversesStructAndPointer(t *testing.T) {
	type Address struct {
		City string
	}

	type User struct {
		Name string
		Addr Address
	}

	data := map[string]any{
		"user": User{
			Name: "alice",
			Addr: Address{City: "Paris"},
		},
		"ptr": &Address{City: "Berlin"},
	}

	if got := MGet(data, "user.Addr.City", ""); got != "Paris" {
		t.Fatalf("expected %q, got %q", "Paris", got)
	}

	if got := MGet(data, "ptr.City", ""); got != "Berlin" {
		t.Fatalf("expected %q, got %q", "Berlin", got)
	}
}

func TestMGetHandlesSliceIndex(t *testing.T) {
	data := map[string]any{
		"list": []any{
			map[string]any{"value": 7},
			map[string]any{"value": 13},
		},
	}

	if got := MGet(data, "list.1.value", 0); got != 13 {
		t.Fatalf("expected %d, got %d", 13, got)
	}

	if got := MGet(data, "list.2.value", 99); got != 99 {
		t.Fatalf("expected default %d for out-of-range index, got %d", 99, got)
	}
}

func TestMGetReturnsDefaultWhenKeyMissing(t *testing.T) {
	data := map[string]any{
		"foo": "bar",
	}

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("expected no panic, got %v", r)
		}
	}()

	if got := MGet(data, "missing", "fallback"); got != "fallback" {
		t.Fatalf("expected default %q, got %q", "fallback", got)
	}
}

func TestMGetNilInterfaceReturnsDefault(t *testing.T) {
	data := map[string]any{
		"nil": any(nil),
	}

	if got := MGet(data, "nil.anything", "default"); got != "default" {
		t.Fatalf("expected default %q for nil interface path, got %q", "default", got)
	}
}
