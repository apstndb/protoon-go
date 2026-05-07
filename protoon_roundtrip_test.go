package protoon

import (
	"testing"

	"github.com/apstndb/protoon-go/testdata"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

func TestRoundTrip(t *testing.T) {
	original := &testdata.Person{
		Name:   "Alice",
		Age:    30,
		Active: true,
		Tags:   []string{"go", "proto"},
		Status: testdata.Status_STATUS_ACTIVE,
	}

	// Marshal to TOON
	b, err := MarshalOptions{ProtoJSONCompat: true}.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}
	t.Logf("TOON:\n%s", string(b))

	// Unmarshal back
	var decoded testdata.Person
	if err := (UnmarshalOptions{ProtoJSONCompat: true}).Unmarshal(b, &decoded); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	// Compare using proto.Equal
	if !proto.Equal(original, (&decoded)) {
		t.Errorf("round-trip mismatch:\noriginal: %s\ndecoded:  %s",
			protojson.Format(original), protojson.Format(&decoded))
	}
}

func TestRoundTripCompany(t *testing.T) {
	original := &testdata.Company{
		Name: "Acme",
		Employees: []*testdata.Person{
			{Name: "Alice", Age: 30, Active: true, Status: testdata.Status_STATUS_ACTIVE},
			{Name: "Bob", Age: 25, Active: false, Status: testdata.Status_STATUS_INACTIVE},
		},
		Metadata: map[string]string{
			"industry": "tech",
			"size":     "small",
		},
	}

	b, err := MarshalOptions{ProtoJSONCompat: true}.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}
	t.Logf("TOON:\n%s", string(b))

	var decoded testdata.Company
	if err := (UnmarshalOptions{ProtoJSONCompat: true}).Unmarshal(b, &decoded); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if !proto.Equal(original, (&decoded)) {
		t.Errorf("round-trip mismatch:\noriginal: %s\ndecoded:  %s",
			protojson.Format(original), protojson.Format(&decoded))
	}
}

func TestRoundTripAllTypes(t *testing.T) {
	original := &testdata.AllTypes{
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

	b, err := MarshalOptions{ProtoJSONCompat: true}.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}
	t.Logf("TOON:\n%s", string(b))

	var decoded testdata.AllTypes
	if err := (UnmarshalOptions{ProtoJSONCompat: true}).Unmarshal(b, &decoded); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if !proto.Equal(original, (&decoded)) {
		t.Errorf("round-trip mismatch:\noriginal: %s\ndecoded:  %s",
			protojson.Format(original), protojson.Format(&decoded))
	}
}
