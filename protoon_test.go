package protoon

import (
	"strings"
	"testing"
	"time"

	"github.com/apstndb/protoon-go/testdata"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestMarshalSimple(t *testing.T) {
	p := &testdata.Person{
		Name:   "Alice",
		Age:    30,
		Active: true,
		Tags:   []string{"developer", "go"},
		Status: testdata.Status_STATUS_ACTIVE,
	}

	b, err := Marshal(p)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	s := string(b)
	t.Logf("TOON output:\n%s", s)

	if !strings.Contains(s, "name: Alice") {
		t.Errorf("expected name field")
	}
	if !strings.Contains(s, "age: 30") {
		t.Errorf("expected age field")
	}
	if !strings.Contains(s, "active: true") {
		t.Errorf("expected active field")
	}
	if !strings.Contains(s, "tags") {
		t.Errorf("expected tags field")
	}
	if !strings.Contains(s, "status: 1") {
		t.Errorf("expected status field as number")
	}
}

func TestMarshalCompany(t *testing.T) {
	c := &testdata.Company{
		Name: "Acme",
		Employees: []*testdata.Person{
			{Name: "Alice", Age: 30, Active: true, Status: testdata.Status_STATUS_ACTIVE},
			{Name: "Bob", Age: 25, Active: true, Status: testdata.Status_STATUS_INACTIVE},
		},
		Metadata: map[string]string{
			"industry": "tech",
			"size":     "small",
		},
		FoundedAt: timestamppb.New(time.Date(2020, 1, 15, 0, 0, 0, 0, time.UTC)),
		Uptime:    durationpb.New(24 * time.Hour),
	}

	b, err := Marshal(c)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	s := string(b)
	t.Logf("TOON output:\n%s", s)

	// Check that employees use tabular form (same schema objects in array)
	if !strings.Contains(s, "employees[2]{name,age,active,status}:") {
		t.Errorf("expected tabular form for employees, got:\n%s", s)
	}
	if !strings.Contains(s, "name: Acme") {
		t.Errorf("expected company name")
	}
	if !strings.Contains(s, "industry: tech") {
		t.Errorf("expected metadata field")
	}
}

func TestMarshalCompanyWithDefaults(t *testing.T) {
	c := &testdata.Company{
		Name: "Acme",
		Employees: []*testdata.Person{
			{Name: "Alice", Age: 30, Active: true, Status: testdata.Status_STATUS_ACTIVE},
			{Name: "Bob", Age: 25, Active: false, Status: testdata.Status_STATUS_INACTIVE},
		},
	}

	b, err := MarshalOptions{EmitDefaultValues: true}.Marshal(c)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	s := string(b)
	t.Logf("TOON output:\n%s", s)

	// With EmitDefaultValues, both employees have the same fields, so tabular form works.
	if !strings.Contains(s, "employees[2]{name,age,active,status}:") {
		t.Errorf("expected tabular form for employees with defaults, got:\n%s", s)
	}
}

func TestMarshalNested(t *testing.T) {
	n := &testdata.Nested{
		Id: "root",
		Child: &testdata.Nested{
			Id: "child",
		},
		Children: []*testdata.Nested{
			{Id: "a"},
			{Id: "b"},
		},
	}

	b, err := Marshal(n)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	s := string(b)
	t.Logf("TOON output:\n%s", s)

	if !strings.Contains(s, "id: root") {
		t.Errorf("expected root id")
	}
	if !strings.Contains(s, "id: child") {
		t.Errorf("expected child id")
	}
	if !strings.Contains(s, "children") {
		t.Errorf("expected children array")
	}
}

func TestMarshalWKT(t *testing.T) {
	attrs, _ := structpb.NewStruct(map[string]any{
		"count": 42,
		"label": "test",
	})
	wkt := &testdata.WKTTest{
		Attributes: attrs,
		Value:      structpb.NewStringValue("hello"),
		Mask:       &fieldmaskpb.FieldMask{Paths: []string{"name", "age"}},
	}

	b, err := Marshal(wkt)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	s := string(b)
	t.Logf("TOON output:\n%s", s)

	if !strings.Contains(s, "count: 42") {
		t.Errorf("expected struct count field")
	}
	if !strings.Contains(s, "value: hello") {
		t.Errorf("expected value field")
	}
	if !strings.Contains(s, "mask[2]: name,age") {
		t.Errorf("expected mask array, got:\n%s", s)
	}
}

func TestMarshalAllTypes(t *testing.T) {
	m := &testdata.AllTypes{
		DoubleField:  1.5,
		FloatField:   2.5,
		Int32Field:   -10,
		Int64Field:   -20,
		Uint32Field:  30,
		Uint64Field:  40,
		Sint32Field:  -50,
		Sint64Field:  -60,
		Fixed32Field: 70,
		Fixed64Field: 80,
		Sfixed32Field: -90,
		Sfixed64Field: -100,
		BoolField:     true,
		StringField:   "hello",
		BytesField:    []byte("world"),
	}

	b, err := Marshal(m)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	s := string(b)
	t.Logf("TOON output:\n%s", s)

	checks := []string{
		"double_field: 1.5",
		"float_field: 2.5",
		"int32_field: -10",
		"int64_field: -20",
		"uint32_field: 30",
		"uint64_field: 40",
		"bool_field: true",
		"string_field: hello",
	}
	for _, check := range checks {
		if !strings.Contains(s, check) {
			t.Errorf("expected %q in output", check)
		}
	}
}

func TestMarshalEnumNames(t *testing.T) {
	p := &testdata.Person{
		Name:   "Alice",
		Status: testdata.Status_STATUS_ACTIVE,
	}

	b, err := MarshalOptions{EmitEnumNames: true}.Marshal(p)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	s := string(b)
	t.Logf("TOON output:\n%s", s)

	if !strings.Contains(s, "STATUS_ACTIVE") {
		t.Errorf("expected enum name, got:\n%s", s)
	}
}

func TestMarshalNil(t *testing.T) {
	b, err := Marshal(nil)
	if err != nil {
		t.Fatalf("Marshal(nil) failed: %v", err)
	}
	if b != nil {
		t.Errorf("expected nil, got %q", b)
	}
}
