# protoon-go

Efficiently convert protocol buffer messages to the [TOON](https://github.com/toon-format/spec) format in Go — without an intermediate JSON representation.

## Features

- **Direct conversion**: Uses `google.golang.org/protobuf/reflect/protoreflect` to inspect proto messages and build TOON data structures directly.
- **Tabular optimization**: Repeated proto messages with uniform schemas are automatically encoded in TOON's compact tabular form (`key[N]{field1,field2}:`).
- **Well-known types**: Supports `google.protobuf.Timestamp`, `Duration`, `Struct`, `Value`, `ListValue`, `FieldMask`, and `Any`.
- **Configurable**: Choose between enum numbers (default) or enum names, and optionally emit default/zero values.

## Installation

```bash
go get github.com/apstndb/protoon-go
```

## Usage

```go
package main

import (
    "fmt"
    "log"

    "github.com/apstndb/protoon-go"
    "google.golang.org/protobuf/types/known/timestamppb"
)

type Person struct {
    Name  string `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty"`
    Age   int32  `protobuf:"varint,2,opt,name=age,proto3" json:"age,omitempty"`
}

func main() {
    p := &Person{Name: "Alice", Age: 30}

    b, err := protoon.Marshal(p)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println(string(b))
    // Output:
    // name: Alice
    // age: 30
}
```

### Options

```go
// Emit enum names instead of numbers
b, err := protoon.MarshalOptions{EmitEnumNames: true}.Marshal(msg)

// Emit default/zero values so repeated messages stay in tabular form
b, err := protoon.MarshalOptions{EmitDefaultValues: true}.Marshal(msg)

// Forward toon encoder options (indent, length markers, etc.)
b, err := protoon.MarshalOptions{
    EncoderOptions: []toon.EncoderOption{toon.WithLengthMarkers(true)},
}.Marshal(msg)
```

## How it works

1. `protoon` walks the `proto.Message` using `protoreflect.Message.Range` (or `Descriptor().Fields()` when `EmitDefaultValues` is set).
2. Each field value is mapped to plain Go values (`string`, `int64`, `float64`, `bool`, `[]any`, `map[string]any`, or `toon.Object`).
3. The resulting structure is passed to `toon.Marshal`, which applies TOON-specific optimizations such as tabular arrays.

Because there is no JSON intermediate step, field ordering from the proto descriptor is preserved and numeric precision is maintained for `int64`/`uint64` values (which JSON cannot represent exactly).

## License

MIT
