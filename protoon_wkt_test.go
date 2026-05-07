package protoon

import (
	"testing"
	"time"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func TestMarshalWKTWrapperTypes(t *testing.T) {
	tests := []struct {
		name     string
		msg      proto.Message
		want     string
		wantCompat string
	}{
		{
			name: "BoolValue",
			msg:  wrapperspb.Bool(true),
			want: "true",
		},
		{
			name: "Int32Value",
			msg:  wrapperspb.Int32(42),
			want: "42",
		},
		{
			name: "Int64Value",
			msg:  wrapperspb.Int64(42),
			want: "42",
			wantCompat: `"42"`,
		},
		{
			name: "UInt32Value",
			msg:  wrapperspb.UInt32(42),
			want: "42",
		},
		{
			name: "UInt64Value",
			msg:  wrapperspb.UInt64(42),
			want: "42",
			wantCompat: `"42"`,
		},
		{
			name:       "FloatValue",
			msg:        wrapperspb.Float(3.14),
			want:       "3.140000104904175",
			wantCompat: "3.14",
		},
		{
			name: "DoubleValue",
			msg:  wrapperspb.Double(3.14),
			want: "3.14",
		},
		{
			name: "StringValue",
			msg:  wrapperspb.String("hello"),
			want: "hello",
		},
		{
			name: "BytesValue",
			msg:  wrapperspb.Bytes([]byte{0x01, 0x02, 0x03}),
			want: "AQID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b, err := Marshal(tt.msg)
			if err != nil {
				t.Fatalf("Marshal failed: %v", err)
			}
			got := string(b)
			if got != tt.want {
				t.Errorf("Marshal() = %q, want %q", got, tt.want)
			}

			if tt.wantCompat != "" {
				bCompat, err := MarshalOptions{ProtoJSONCompat: true}.Marshal(tt.msg)
				if err != nil {
					t.Fatalf("Marshal compat failed: %v", err)
				}
				gotCompat := string(bCompat)
				if gotCompat != tt.wantCompat {
					t.Errorf("Marshal compat() = %q, want %q", gotCompat, tt.wantCompat)
				}
			}
		})
	}
}

func TestMarshalWKTEmpty(t *testing.T) {
	b, err := Marshal(&emptypb.Empty{})
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}
	if string(b) != "{}" {
		t.Errorf("Marshal(Empty) = %q, want %q", string(b), "{}")
	}
}

func TestMarshalWKTAnyWithWellKnownType(t *testing.T) {
	// Any containing a Timestamp (a well-known type with custom JSON encoding)
	ts := timestamppb.New(time.Date(2020, 1, 15, 0, 0, 0, 0, time.UTC))
	anyMsg, err := anypb.New(ts)
	if err != nil {
		t.Fatal(err)
	}

	b, err := MarshalOptions{ProtoJSONCompat: true}.Marshal(anyMsg)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	got := string(b)
	t.Logf("Any(Timestamp) compat = %s", got)

	// ProtoJSONCompat mode should emit @type and value fields for well-known types
	if !contains(got, `"@type"`) {
		t.Errorf("expected @type field")
	}
	if !contains(got, `"2020-01-15T00:00:00Z"`) {
		t.Errorf("expected value field with formatted timestamp")
	}
}

func TestMarshalWKTStructAndValue(t *testing.T) {
	s := &structpb.Struct{
		Fields: map[string]*structpb.Value{
			"null":   structpb.NewNullValue(),
			"bool":   structpb.NewBoolValue(true),
			"number": structpb.NewNumberValue(3.14),
			"string": structpb.NewStringValue("hello"),
		},
	}

	b, err := Marshal(s)
	if err != nil {
		t.Fatalf("Marshal(Struct) failed: %v", err)
	}
	got := string(b)
	t.Logf("Struct = %s", got)

	if !contains(got, "null: null") {
		t.Errorf("expected null field")
	}
	if !contains(got, "bool: true") {
		t.Errorf("expected bool field")
	}
	if !contains(got, "number: 3.14") {
		t.Errorf("expected number field")
	}
	if !contains(got, "string: hello") {
		t.Errorf("expected string field")
	}
}

func TestMarshalWKTFieldMaskCompat(t *testing.T) {
	fm := &fieldmaskpb.FieldMask{Paths: []string{"user_name", "profile_picture"}}

	b, err := MarshalOptions{ProtoJSONCompat: true}.Marshal(fm)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}
	got := string(b)
	t.Logf("FieldMask compat = %s", got)

	// protojson emits camelCase comma-separated string
	if got != `"userName,profilePicture"` {
		t.Errorf("FieldMask compat = %q, want %q", got, `"userName,profilePicture"`)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsImpl(s, substr))
}

func containsImpl(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
