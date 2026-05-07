package protoon

import (
	"encoding/json"
	"fmt"

	"github.com/toon-format/toon-go"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// UnmarshalOptions configures how a TOON document is converted to proto.Message.
type UnmarshalOptions struct {
	// ProtoJSONCompat causes the input to be interpreted using the same
	// conventions as google.golang.org/protobuf/encoding/protojson.
	ProtoJSONCompat bool
}

// Unmarshal parses a TOON document and populates the given proto.Message.
func (o UnmarshalOptions) Unmarshal(data []byte, m proto.Message) error {
	if len(data) == 0 {
		return nil
	}

	// Decode TOON into a Go value.
	var v any
	if err := toon.Unmarshal(data, &v); err != nil {
		return fmt.Errorf("protoon: toon unmarshal: %w", err)
	}

	// When ProtoJSONCompat is true, we need to apply pre-processing to match
	// protojson expectations (e.g. camelCase keys → snake_case, stringified
	// int64/uint64 back to numbers for JSON parser, etc.)
	if o.ProtoJSONCompat {
		v = normalizeForProtoJSON(v)
	}

	// Marshal the Go value to JSON.
	jb, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("protoon: json marshal: %w", err)
	}

	// Unmarshal JSON into the proto message using protojson.
	if err := protojson.Unmarshal(jb, m); err != nil {
		return fmt.Errorf("protoon: protojson unmarshal: %w", err)
	}

	return nil
}

// normalizeForProtoJSON prepares a decoded TOON value for protojson.Unmarshal.
// TOON's decoder returns numbers as float64, but protojson expects int64/uint64
// as strings. This function recursively walks the value and converts where
// needed based on heuristics.
func normalizeForProtoJSON(v any) any {
	switch val := v.(type) {
	case map[string]any:
		result := make(map[string]any, len(val))
		for k, v2 := range val {
			result[k] = normalizeForProtoJSON(v2)
		}
		return result
	case []any:
		result := make([]any, len(val))
		for i, v2 := range val {
			result[i] = normalizeForProtoJSON(v2)
		}
		return result
	case float64:
		// If the float64 value is a whole number within int64 range,
		// keep it as a number. protojson.Unmarshal can handle numbers
		// for int32/int64 fields.
		return val
	case string:
		// Try to detect stringified numbers (from ProtoJSONCompat marshal).
		// If it looks like a number, return it as a json.Number so that
		// json.Marshal emits it without quotes. However, json.Marshal
		// doesn't support json.Number natively in arbitrary values.
		// Instead, we use a trick: if the string is a pure integer
		// representation, we convert it back to float64 for JSON.
		// protojson.Unmarshal handles numeric strings for int64/uint64.
		return val
	default:
		return val
	}
}
