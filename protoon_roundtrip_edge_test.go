package protoon

import (
	"math"
	"testing"
	"time"

	"github.com/apstndb/protoon-go/testdata"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestRoundTripInt64Precision(t *testing.T) {
	original := &testdata.AllTypes{
		Int64Field:  math.MaxInt64,
		Uint64Field: math.MaxUint64,
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

func TestRoundTripWKT(t *testing.T) {
	original := &testdata.Company{
		Name:      "Acme",
		FoundedAt: timestamppb.New(time.Date(2020, 1, 15, 0, 0, 0, 0, time.UTC)),
		Uptime:    durationpb.New(24 * time.Hour),
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

func TestRoundTripFieldMask(t *testing.T) {
	original := &testdata.WKTTest{
		Mask: &fieldmaskpb.FieldMask{Paths: []string{"user_name", "profile_picture"}},
	}

	b, err := MarshalOptions{ProtoJSONCompat: true}.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}
	t.Logf("TOON:\n%s", string(b))

	var decoded testdata.WKTTest
	if err := (UnmarshalOptions{ProtoJSONCompat: true}).Unmarshal(b, &decoded); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if !proto.Equal(original, (&decoded)) {
		t.Errorf("round-trip mismatch:\noriginal: %s\ndecoded:  %s",
			protojson.Format(original), protojson.Format(&decoded))
	}
}

func TestRoundTripEmptyMessage(t *testing.T) {
	original := &testdata.Person{
		Name: "test",
	}

	b, err := MarshalOptions{ProtoJSONCompat: true}.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}
	t.Logf("TOON:\n%s", string(b))

	var decoded testdata.Person
	if err := (UnmarshalOptions{ProtoJSONCompat: true}).Unmarshal(b, &decoded); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if !proto.Equal(original, (&decoded)) {
		t.Errorf("round-trip mismatch:\noriginal: %s\ndecoded:  %s",
			protojson.Format(original), protojson.Format(&decoded))
	}
}
