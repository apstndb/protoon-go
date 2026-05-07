package protoon

import (
	"strings"
	"testing"

	"github.com/apstndb/protoon-go/testdata"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// TestEmitDefaultValuesPerType verifies that per-type default emission
// makes specific repeated message types tabular without adding noise
// to the rest of the protobuf tree.
func TestEmitDefaultValuesPerType(t *testing.T) {
	// Simulate a heterogeneous structure similar to QueryPlan:
	// Company = heterogeneous container, Person = row-like type

	// For this test we use actual protobuf messages:
	// Company = heterogeneous container, Person = row-like type
	company := &testdata.Company{
		Name: "Acme",
		Employees: []*testdata.Person{
			{Name: "Alice", Age: 30, Active: true, Status: testdata.Status_STATUS_ACTIVE},
			{Name: "Bob", Age: 25, Active: false, Status: testdata.Status_STATUS_INACTIVE},
		},
		Metadata: map[string]string{
			"industry": "tech",
		},
	}

	// Without per-type defaults: employees are not uniform (active differs)
	// so they render as a list, not a table.
	bSparse, err := MarshalOptions{ProtoJSONCompat: true}.Marshal(company)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}
	t.Logf("Sparse (no defaults):\n%s", string(bSparse))

	// With per-type defaults for Person: all Person rows emit the same
	// fields, enabling tabular form.
	bTable, err := MarshalOptions{
		ProtoJSONCompat: true,
		EmitDefaultValuesForTypes: []protoreflect.FullName{
			"testdata.Person",
		},
	}.Marshal(company)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}
	t.Logf("Per-type defaults for Person:\n%s", string(bTable))

	// Verify the per-type output has tabular form for employees
	if !strings.Contains(string(bTable), "employees[2]{name,age,active,status}:") {
		t.Errorf("expected tabular form for employees in per-type mode, got:\n%s", string(bTable))
	}

	// Verify metadata is still sparse (not affected by per-type rule)
	if strings.Contains(string(bTable), "metadata{}:") || strings.Contains(string(bTable), "metadata[0]:") {
		t.Errorf("metadata should remain sparse, got:\n%s", string(bTable))
	}
}

// TestEmitDefaultValuesPerTypePredicate verifies the predicate callback form.
func TestEmitDefaultValuesPerTypePredicate(t *testing.T) {
	company := &testdata.Company{
		Name: "Acme",
		Employees: []*testdata.Person{
			{Name: "Alice", Age: 30, Active: true, Status: testdata.Status_STATUS_ACTIVE},
			{Name: "Bob", Age: 25, Active: false, Status: testdata.Status_STATUS_INACTIVE},
		},
	}

	bTable, err := MarshalOptions{
		ProtoJSONCompat: true,
		EmitDefaultValuesForMessage: func(md protoreflect.MessageDescriptor) bool {
			return md.FullName() == "testdata.Person"
		},
	}.Marshal(company)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}
	t.Logf("Predicate-based defaults:\n%s", string(bTable))

	if !strings.Contains(string(bTable), "employees[2]{name,age,active,status}:") {
		t.Errorf("expected tabular form for employees, got:\n%s", string(bTable))
	}
}

// TestEmitDefaultValuesGlobalWins verifies that global EmitDefaultValues
// still takes precedence over per-type settings.
func TestEmitDefaultValuesGlobalWins(t *testing.T) {
	company := &testdata.Company{
		Name:      "Acme",
		Employees: []*testdata.Person{},
		Metadata:  map[string]string{},
	}

	// Global EmitDefaultValues=true: zero-valued scalar fields are emitted,
	// but empty repeated/map fields are still skipped to avoid noise.
	bGlobal, err := MarshalOptions{
		ProtoJSONCompat:   true,
		EmitDefaultValues: true,
	}.Marshal(company)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}
	t.Logf("Global defaults:\n%s", string(bGlobal))

	// Empty lists/maps are skipped even in global default mode
	if strings.Contains(string(bGlobal), "employees[0]:") {
		t.Errorf("empty employees list should be skipped even with global defaults, got:\n%s", string(bGlobal))
	}

	// But scalar zero values should appear
	if !strings.Contains(string(bGlobal), "name: Acme") {
		t.Errorf("expected name field, got:\n%s", string(bGlobal))
	}

	// Per-type with global=true should behave the same as global
	bMixed, err := MarshalOptions{
		ProtoJSONCompat:           true,
		EmitDefaultValues:         true,
		EmitDefaultValuesForTypes: []protoreflect.FullName{"testdata.Company"},
	}.Marshal(company)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	if strings.Contains(string(bMixed), "employees[0]:") {
		t.Errorf("global should still win (empty lists skipped), got:\n%s", string(bMixed))
	}
}

// TestEmitDefaultValuesNestedMessage verifies per-type emission works
// for nested message fields too, while avoiding infinite recursion on
// self-referencing types.
func TestEmitDefaultValuesNestedMessage(t *testing.T) {
	nested := &testdata.Nested{
		Id: "root",
		Child: &testdata.Nested{
			Id: "child",
		},
		Children: []*testdata.Nested{
			{Id: "a"},
			{Id: "b"},
		},
	}

	// Without defaults
	bSparse, err := Marshal(nested)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}
	t.Logf("Sparse nested:\n%s", string(bSparse))

	// With per-type defaults for Nested
	bDefault, err := MarshalOptions{
		EmitDefaultValuesForTypes: []protoreflect.FullName{
			"testdata.Nested",
		},
	}.Marshal(nested)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}
	t.Logf("Per-type defaults for Nested:\n%s", string(bDefault))

	// The root should include "child" because it's set, and "children" because they have elements
	if !strings.Contains(string(bDefault), "id: root") {
		t.Errorf("expected root id, got:\n%s", string(bDefault))
	}
	if !strings.Contains(string(bDefault), "id: child") {
		t.Errorf("expected child id, got:\n%s", string(bDefault))
	}
	// children[2] should appear because the list is non-empty
	if !strings.Contains(string(bDefault), "children[2]{id}:") {
		t.Errorf("expected children list, got:\n%s", string(bDefault))
	}
}
