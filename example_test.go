package protoon

import (
	"fmt"
	"log"

	"github.com/apstndb/protoon-go/testdata"
	"google.golang.org/protobuf/types/known/anypb"
)

func ExampleMarshal() {
	p := &testdata.Person{
		Name:   "Alice",
		Age:    30,
		Active: true,
		Tags:   []string{"go", "proto"},
		Status: testdata.Status_STATUS_ACTIVE,
	}

	b, err := Marshal(p)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(b))
	// Output:
	// name: Alice
	// age: 30
	// active: true
	// status: 1
	// tags[2]: go,proto
}

func ExampleMarshalOptions_emitEnumNames() {
	p := &testdata.Person{
		Name:   "Alice",
		Status: testdata.Status_STATUS_ACTIVE,
	}

	b, err := MarshalOptions{EmitEnumNames: true}.Marshal(p)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(b))
	// Output:
	// name: Alice
	// status: STATUS_ACTIVE
}

func ExampleMarshalOptions_emitDefaultValues() {
	c := &testdata.Company{
		Name: "Acme",
		Employees: []*testdata.Person{
			{Name: "Alice", Age: 30, Active: true, Status: testdata.Status_STATUS_ACTIVE},
			{Name: "Bob", Age: 25, Active: false, Status: testdata.Status_STATUS_INACTIVE},
		},
	}

	b, err := MarshalOptions{EmitDefaultValues: true}.Marshal(c)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(b))
	// Output:
	// name: Acme
	// employees[2]{name,age,active,status}:
	//   Alice,30,true,1
	//   Bob,25,false,2
}

func ExampleMarshal_any() {
	inner := &testdata.Person{Name: "Alice", Age: 30}
	anyMsg, err := anypb.New(inner)
	if err != nil {
		log.Fatal(err)
	}

	b, err := Marshal(anyMsg)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(b))
	// Output:
	// name: Alice
	// age: 30
}
