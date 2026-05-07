package protoon

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/apstndb/protoon-go/testdata"
	"github.com/toon-format/toon-go"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func toJSON(t *testing.T, data []byte) any {
	t.Helper()
	var v any
	if err := json.Unmarshal(data, &v); err != nil {
		// Try TOON unmarshal first, then JSON marshal/unmarshal
		var tv any
		if terr := toon.Unmarshal(data, &tv); terr != nil {
			t.Fatalf("failed to parse as JSON and TOON: json=%v, toon=%v", err, terr)
		}
		jb, jerr := json.Marshal(tv)
		if jerr != nil {
			t.Fatalf("failed to marshal TOON value to JSON: %v", jerr)
		}
		if err := json.Unmarshal(jb, &v); err != nil {
			t.Fatalf("failed to unmarshal JSON: %v", err)
		}
	}
	return v
}

func protojsonBytes(t *testing.T, m proto.Message) []byte {
	t.Helper()
	b, err := protojson.MarshalOptions{}.Marshal(m)
	if err != nil {
		t.Fatalf("protojson.Marshal failed: %v", err)
	}
	return b
}

func assertJSONEqual(t *testing.T, a, b any) {
	t.Helper()
	ja, err := json.Marshal(a)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}
	jb, err := json.Marshal(b)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}
	if string(ja) != string(jb) {
		t.Errorf("values differ\nleft:  %s\nright: %s", ja, jb)
	}
}

func TestProtoJSONCompatSimple(t *testing.T) {
	p := &testdata.Person{
		Name:   "Alice",
		Age:    30,
		Active: true,
		Tags:   []string{"go", "proto"},
		Status: testdata.Status_STATUS_ACTIVE,
	}

	pj := protojsonBytes(t, p)
	po, err := MarshalOptions{ProtoJSONCompat: true}.Marshal(p)
	if err != nil {
		t.Fatalf("protoon.Marshal failed: %v", err)
	}

	pjVal := toJSON(t, pj)
	poVal := toJSON(t, po)

	t.Logf("protojson: %s", pj)
	t.Logf("protoon:   %s", po)

	assertJSONEqual(t, pjVal, poVal)
}

func TestProtoJSONCompatCompany(t *testing.T) {
	c := &testdata.Company{
		Name: "Acme",
		Employees: []*testdata.Person{
			{Name: "Alice", Age: 30, Active: true, Status: testdata.Status_STATUS_ACTIVE},
			{Name: "Bob", Age: 25, Active: false, Status: testdata.Status_STATUS_INACTIVE},
		},
		Metadata: map[string]string{
			"industry": "tech",
			"size":     "small",
		},
		FoundedAt: timestamppb.New(time.Date(2020, 1, 15, 0, 0, 0, 0, time.UTC)),
		Uptime:    durationpb.New(24 * time.Hour),
	}

	pj := protojsonBytes(t, c)
	po, err := MarshalOptions{ProtoJSONCompat: true}.Marshal(c)
	if err != nil {
		t.Fatalf("protoon.Marshal failed: %v", err)
	}

	pjVal := toJSON(t, pj)
	poVal := toJSON(t, po)

	t.Logf("protojson: %s", pj)
	t.Logf("protoon:   %s", po)

	assertJSONEqual(t, pjVal, poVal)
}

func TestProtoJSONCompatWKT(t *testing.T) {
	attrs, _ := structpb.NewStruct(map[string]any{
		"count": 42,
		"label": "test",
	})
	wkt := &testdata.WKTTest{
		Attributes: attrs,
		Value:      structpb.NewStringValue("hello"),
		Mask:       &fieldmaskpb.FieldMask{Paths: []string{"name", "age"}},
	}

	pj := protojsonBytes(t, wkt)
	po, err := MarshalOptions{ProtoJSONCompat: true}.Marshal(wkt)
	if err != nil {
		t.Fatalf("protoon.Marshal failed: %v", err)
	}

	pjVal := toJSON(t, pj)
	poVal := toJSON(t, po)

	t.Logf("protojson: %s", pj)
	t.Logf("protoon:   %s", po)

	assertJSONEqual(t, pjVal, poVal)
}

func TestProtoJSONCompatAny(t *testing.T) {
	inner := &testdata.Person{Name: "Alice", Age: 30}
	anyMsg, err := anypb.New(inner)
	if err != nil {
		t.Fatal(err)
	}

	pj := protojsonBytes(t, anyMsg)
	po, err := MarshalOptions{ProtoJSONCompat: true}.Marshal(anyMsg)
	if err != nil {
		t.Fatalf("protoon.Marshal failed: %v", err)
	}

	pjVal := toJSON(t, pj)
	poVal := toJSON(t, po)

	t.Logf("protojson: %s", pj)
	t.Logf("protoon:   %s", po)

	assertJSONEqual(t, pjVal, poVal)
}

func TestProtoJSONCompatAllTypes(t *testing.T) {
	m := &testdata.AllTypes{
		DoubleField:   1.5,
		FloatField:    2.5,
		Int32Field:    -10,
		Int64Field:    -20,
		Uint32Field:   30,
		Uint64Field:   40,
		Sint32Field:   -50,
		Sint64Field:   -60,
		Fixed32Field:  70,
		Fixed64Field:  80,
		Sfixed32Field: -90,
		Sfixed64Field: -100,
		BoolField:     true,
		StringField:   "hello",
		BytesField:    []byte("world"),
	}

	pj := protojsonBytes(t, m)
	po, err := MarshalOptions{ProtoJSONCompat: true}.Marshal(m)
	if err != nil {
		t.Fatalf("protoon.Marshal failed: %v", err)
	}

	pjVal := toJSON(t, pj)
	poVal := toJSON(t, po)

	t.Logf("protojson: %s", pj)
	t.Logf("protoon:   %s", po)

	assertJSONEqual(t, pjVal, poVal)
}

func mustJSON(t *testing.T, v any) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}
	return b
}
