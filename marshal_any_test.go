package protoon

import (
	"strings"
	"testing"

	"github.com/apstndb/protoon-go/testdata"
	"github.com/toon-format/toon-go"
)

func TestMarshalAnySimple(t *testing.T) {
	p := &testdata.Person{
		Name:   "Alice",
		Age:    30,
		Active: true,
	}

	b, err := MarshalAny(p)
	if err != nil {
		t.Fatalf("MarshalAny failed: %v", err)
	}
	t.Logf("TOON:\n%s", string(b))

	if !strings.Contains(string(b), "name: Alice") {
		t.Errorf("expected name field")
	}
}

func TestMarshalAnyStructWithProto(t *testing.T) {
	type Response struct {
		Person  *testdata.Person `toon:"person"`
		Count   int              `toon:"count"`
		Message string           `toon:"message"`
	}

	resp := Response{
		Person: &testdata.Person{
			Name:   "Alice",
			Age:    30,
			Active: true,
		},
		Count:   42,
		Message: "ok",
	}

	b, err := MarshalAny(resp)
	if err != nil {
		t.Fatalf("MarshalAny failed: %v", err)
	}
	t.Logf("TOON:\n%s", string(b))

	s := string(b)
	if !strings.Contains(s, "person:") {
		t.Errorf("expected person field")
	}
	if !strings.Contains(s, "name: Alice") {
		t.Errorf("expected nested name field")
	}
	if !strings.Contains(s, "count: 42") {
		t.Errorf("expected count field")
	}
	if !strings.Contains(s, "message: ok") {
		t.Errorf("expected message field")
	}
}

func TestMarshalAnySliceWithProto(t *testing.T) {
	people := []*testdata.Person{
		{Name: "Alice", Age: 30},
		{Name: "Bob", Age: 25},
	}

	b, err := MarshalAny(people)
	if err != nil {
		t.Fatalf("MarshalAny failed: %v", err)
	}
	t.Logf("TOON:\n%s", string(b))

	// Proto messages with uniform schemas are encoded in tabular form
	if !strings.Contains(string(b), "[2]{name,age}:") {
		t.Errorf("expected tabular form")
	}
	if !strings.Contains(string(b), "Alice,30") {
		t.Errorf("expected Alice")
	}
	if !strings.Contains(string(b), "Bob,25") {
		t.Errorf("expected Bob")
	}
}

func TestMarshalAnyMapWithProto(t *testing.T) {
	data := map[string]any{
		"user": &testdata.Person{Name: "Alice", Age: 30},
		"meta": map[string]any{
			"count": 42,
		},
	}

	b, err := MarshalAny(data)
	if err != nil {
		t.Fatalf("MarshalAny failed: %v", err)
	}
	t.Logf("TOON:\n%s", string(b))

	s := string(b)
	if !strings.Contains(s, "user:") {
		t.Errorf("expected user field")
	}
	if !strings.Contains(s, "meta:") {
		t.Errorf("expected meta field")
	}
}

func TestMarshalAnyToonObject(t *testing.T) {
	obj := toon.NewObject(
		toon.Field{Key: "person", Value: &testdata.Person{Name: "Alice", Age: 30}},
		toon.Field{Key: "count", Value: 42},
	)

	b, err := MarshalAny(obj)
	if err != nil {
		t.Fatalf("MarshalAny failed: %v", err)
	}
	t.Logf("TOON:\n%s", string(b))

	s := string(b)
	if !strings.Contains(s, "person:") {
		t.Errorf("expected person field")
	}
	if !strings.Contains(s, "count: 42") {
		t.Errorf("expected count field")
	}
}

func TestMarshalAnyProtoJSONCompat(t *testing.T) {
	type Response struct {
		Person  *testdata.Person `toon:"person"`
		Count   int              `toon:"count"`
	}

	resp := Response{
		Person: &testdata.Person{
			Name:   "Alice",
			Age:    30,
			Status: testdata.Status_STATUS_ACTIVE,
		},
		Count: 42,
	}

	b, err := MarshalAny(resp, WithProtoJSONCompat())
	if err != nil {
		t.Fatalf("MarshalAny failed: %v", err)
	}
	t.Logf("TOON:\n%s", string(b))

	s := string(b)
	if !strings.Contains(s, "status: STATUS_ACTIVE") {
		t.Errorf("expected enum name in compat mode, got:\n%s", s)
	}
	if !strings.Contains(s, "person:") {
		t.Errorf("expected person field")
	}
}
