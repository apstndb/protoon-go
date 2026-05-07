// Package protoon converts protocol buffer messages to the TOON format
// without an intermediate JSON representation, preserving field order and
// enabling TOON-specific tabular optimization for repeated messages.
package protoon

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/toon-format/toon-go"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

// MarshalOptions configures how a proto.Message is converted to TOON.
type MarshalOptions struct {
	// EncoderOptions are forwarded to the toon encoder.
	EncoderOptions []toon.EncoderOption

	// EmitEnumNames emits the protobuf enum name as a string instead of the
	// numeric value. When false (default), enums are emitted as numbers.
	EmitEnumNames bool

	// EmitDefaultValues causes fields set to their zero value to be emitted.
	// By default, only populated fields are emitted (matching proto.Range).
	EmitDefaultValues bool

	// EmitDefaultValuesForTypes emits default fields only for messages whose
	// descriptor FullName matches one of the listed names. This is useful for
	// making specific repeated row-like message types tabular without adding
	// noise to the entire protobuf tree. If EmitDefaultValues is true, this
	// option is ignored (global wins).
	EmitDefaultValuesForTypes []protoreflect.FullName

	// EmitDefaultValuesForMessage is a predicate that can be used for more
	// advanced control over which message types should emit default values.
	// If both EmitDefaultValuesForTypes and EmitDefaultValuesForMessage are
	// set, either matching condition enables default emission for that type.
	// If EmitDefaultValues is true, this option is ignored.
	EmitDefaultValuesForMessage func(protoreflect.MessageDescriptor) bool

	// ProtoJSONCompat causes the output to match the JSON representation
	// produced by google.golang.org/protobuf/encoding/protojson.
	// This affects well-known types (Timestamp, Duration, FieldMask, Any,
	// wrapper types) and enum encoding.
	ProtoJSONCompat bool
}

// Marshal converts a proto.Message to TOON using default options.
func Marshal(m proto.Message) ([]byte, error) {
	return MarshalOptions{}.Marshal(m)
}

// Marshal converts a proto.Message to TOON.
func (o MarshalOptions) Marshal(m proto.Message) ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	if o.ProtoJSONCompat {
		// protojson emits enums as names by default.
		o.EmitEnumNames = true
		// Ensure timestamps are formatted compatibly.
		o.EncoderOptions = append([]toon.EncoderOption{toon.WithTimeFormatter(formatProtoJSONTimestamp)}, o.EncoderOptions...)
	}
	v, err := o.marshalMessage(m.ProtoReflect())
	if err != nil {
		return nil, err
	}
	// toon-go emits an empty string for a root-level empty object,
	// but protojson represents google.protobuf.Empty as {}.
	if obj, ok := v.(toon.Object); ok && obj.IsEmpty() {
		return []byte("{}"), nil
	}
	return toon.Marshal(v, o.EncoderOptions...)
}

func formatProtoJSONTimestamp(t time.Time) string {
	t = t.UTC()
	x := t.Format("2006-01-02T15:04:05.000000000")
	x = strings.TrimSuffix(x, "000")
	x = strings.TrimSuffix(x, "000")
	x = strings.TrimSuffix(x, ".000")
	return x + "Z"
}

func (o MarshalOptions) marshalMessage(pm protoreflect.Message) (any, error) {
	// Handle well-known types before generic message handling.
	if v, ok, err := o.marshalWKT(pm); ok {
		if err != nil {
			return nil, err
		}
		return v, nil
	}

	emitDefaults := o.shouldEmitDefaults(pm.Descriptor())
	var fields []toon.Field

	if emitDefaults {
		desc := pm.Descriptor()
		for i := 0; i < desc.Fields().Len(); i++ {
			fd := desc.Fields().Get(i)
			if fd.ContainingOneof() != nil && !pm.Has(fd) {
				continue
			}
			// For message fields, respect presence: do not emit if not set.
			// This prevents infinite recursion for self-referencing messages
			// and avoids noise from empty nested messages.
			if (fd.Kind() == protoreflect.MessageKind || fd.Kind() == protoreflect.GroupKind) && !pm.Has(fd) {
				continue
			}
			v := pm.Get(fd)
			// Skip empty repeated and map fields even in default-values mode.
			if fd.IsList() && v.List().Len() == 0 {
				continue
			}
			if fd.IsMap() && v.Map().Len() == 0 {
				continue
			}
			val, err := o.marshalFieldValue(fd, v)
			if err != nil {
				return nil, err
			}
			fields = append(fields, toon.Field{Key: o.fieldName(fd), Value: val})
		}
	} else {
		var rangeErr error
		pm.Range(func(fd protoreflect.FieldDescriptor, v protoreflect.Value) bool {
			val, err := o.marshalFieldValue(fd, v)
			if err != nil {
				rangeErr = err
				return false
			}
			fields = append(fields, toon.Field{Key: o.fieldName(fd), Value: val})
			return true
		})
		if rangeErr != nil {
			return nil, rangeErr
		}
	}

	return toon.NewObject(fields...), nil
}

func (o MarshalOptions) shouldEmitDefaults(md protoreflect.MessageDescriptor) bool {
	if o.EmitDefaultValues {
		return true
	}
	name := md.FullName()
	for _, n := range o.EmitDefaultValuesForTypes {
		if name == n {
			return true
		}
	}
	if o.EmitDefaultValuesForMessage != nil {
		return o.EmitDefaultValuesForMessage(md)
	}
	return false
}

func (o MarshalOptions) fieldName(fd protoreflect.FieldDescriptor) string {
	if o.ProtoJSONCompat {
		return jsonCamelCase(string(fd.Name()))
	}
	return string(fd.Name())
}

func (o MarshalOptions) marshalFieldValue(fd protoreflect.FieldDescriptor, v protoreflect.Value) (any, error) {
	switch {
	case fd.IsList():
		return o.marshalList(fd, v.List())
	case fd.IsMap():
		return o.marshalMap(fd, v.Map())
	default:
		return o.marshalSingular(fd, v)
	}
}

func (o MarshalOptions) marshalSingular(fd protoreflect.FieldDescriptor, v protoreflect.Value) (any, error) {
	switch fd.Kind() {
	case protoreflect.BoolKind:
		return v.Bool(), nil
	case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind:
		return v.Int(), nil // int64
	case protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind:
		if o.ProtoJSONCompat {
			return fmt.Sprintf("%d", v.Int()), nil
		}
		return v.Int(), nil
	case protoreflect.Uint32Kind, protoreflect.Fixed32Kind:
		return v.Uint(), nil // uint64
	case protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
		if o.ProtoJSONCompat {
			return fmt.Sprintf("%d", v.Uint()), nil
		}
		return v.Uint(), nil
	case protoreflect.FloatKind:
		if o.ProtoJSONCompat {
			return json.Number(strconv.FormatFloat(v.Float(), 'g', -1, 32)), nil
		}
		return float32(v.Float()), nil
	case protoreflect.DoubleKind:
		if o.ProtoJSONCompat {
			return json.Number(strconv.FormatFloat(v.Float(), 'g', -1, 64)), nil
		}
		return v.Float(), nil
	case protoreflect.StringKind:
		return v.String(), nil
	case protoreflect.BytesKind:
		return base64.StdEncoding.EncodeToString(v.Bytes()), nil
	case protoreflect.MessageKind, protoreflect.GroupKind:
		return o.marshalMessage(v.Message())
	case protoreflect.EnumKind:
		if o.EmitEnumNames {
			if ev := fd.Enum().Values().ByNumber(v.Enum()); ev != nil {
				return string(ev.Name()), nil
			}
			return "", nil
		}
		return int32(v.Enum()), nil
	default:
		return nil, fmt.Errorf("protoon: unsupported field kind %v", fd.Kind())
	}
}

func (o MarshalOptions) marshalList(fd protoreflect.FieldDescriptor, list protoreflect.List) (any, error) {
	length := list.Len()
	result := make([]any, 0, length)

	if fd.Kind() == protoreflect.MessageKind || fd.Kind() == protoreflect.GroupKind {
		for i := 0; i < length; i++ {
			v, err := o.marshalMessage(list.Get(i).Message())
			if err != nil {
				return nil, err
			}
			result = append(result, v)
		}
	} else {
		for i := 0; i < length; i++ {
			v, err := o.marshalSingular(fd, list.Get(i))
			if err != nil {
				return nil, err
			}
			result = append(result, v)
		}
	}

	return result, nil
}

func (o MarshalOptions) marshalMap(fd protoreflect.FieldDescriptor, m protoreflect.Map) (any, error) {
	result := make(map[string]any, m.Len())
	var rangeErr error
	m.Range(func(k protoreflect.MapKey, v protoreflect.Value) bool {
		key := k.String()
		val, err := o.marshalSingular(fd.MapValue(), v)
		if err != nil {
			rangeErr = err
			return false
		}
		result[key] = val
		return true
	})
	if rangeErr != nil {
		return nil, rangeErr
	}
	return result, nil
}

func (o MarshalOptions) marshalWKT(pm protoreflect.Message) (any, bool, error) {
	switch pm.Descriptor().FullName() {
	case "google.protobuf.Timestamp":
		return pm.Interface().(*timestamppb.Timestamp).AsTime(), true, nil
	case "google.protobuf.Duration":
		if o.ProtoJSONCompat {
			d := pm.Interface().(*durationpb.Duration)
			return formatProtoJSONDuration(d), true, nil
		}
		return pm.Interface().(*durationpb.Duration).AsDuration(), true, nil
	case "google.protobuf.Struct":
		m, err := o.marshalStruct(pm)
		if err != nil {
			return nil, true, err
		}
		return m, true, nil
	case "google.protobuf.Value":
		v, err := o.marshalProtoValue(pm)
		if err != nil {
			return nil, true, err
		}
		return v, true, nil
	case "google.protobuf.ListValue":
		lv, err := o.marshalProtoListValue(pm)
		if err != nil {
			return nil, true, err
		}
		return lv, true, nil
	case "google.protobuf.FieldMask":
		fm := pm.Interface().(*fieldmaskpb.FieldMask)
		if o.ProtoJSONCompat {
			return formatProtoJSONFieldMask(fm), true, nil
		}
		return fm.GetPaths(), true, nil
	case "google.protobuf.Any":
		v, err := o.marshalAny(pm)
		return v, true, err
	case "google.protobuf.BoolValue":
		return pm.Interface().(*wrapperspb.BoolValue).GetValue(), true, nil
	case "google.protobuf.Int32Value":
		return pm.Interface().(*wrapperspb.Int32Value).GetValue(), true, nil
	case "google.protobuf.Int64Value":
		v := pm.Interface().(*wrapperspb.Int64Value).GetValue()
		if o.ProtoJSONCompat {
			return fmt.Sprintf("%d", v), true, nil
		}
		return v, true, nil
	case "google.protobuf.UInt32Value":
		return pm.Interface().(*wrapperspb.UInt32Value).GetValue(), true, nil
	case "google.protobuf.UInt64Value":
		v := pm.Interface().(*wrapperspb.UInt64Value).GetValue()
		if o.ProtoJSONCompat {
			return fmt.Sprintf("%d", v), true, nil
		}
		return v, true, nil
	case "google.protobuf.FloatValue":
		v := pm.Interface().(*wrapperspb.FloatValue).GetValue()
		if o.ProtoJSONCompat {
			return json.Number(strconv.FormatFloat(float64(v), 'g', -1, 32)), true, nil
		}
		return v, true, nil
	case "google.protobuf.DoubleValue":
		v := pm.Interface().(*wrapperspb.DoubleValue).GetValue()
		if o.ProtoJSONCompat {
			return json.Number(strconv.FormatFloat(v, 'g', -1, 64)), true, nil
		}
		return v, true, nil
	case "google.protobuf.StringValue":
		return pm.Interface().(*wrapperspb.StringValue).GetValue(), true, nil
	case "google.protobuf.BytesValue":
		return base64.StdEncoding.EncodeToString(pm.Interface().(*wrapperspb.BytesValue).GetValue()), true, nil
	case "google.protobuf.Empty":
		return toon.NewObject(), true, nil
	}
	return nil, false, nil
}

func formatProtoJSONDuration(d *durationpb.Duration) string {
	secs := d.GetSeconds()
	nanos := d.GetNanos()
	if secs == 0 && nanos == 0 {
		return "0s"
	}
	sign := ""
	if secs < 0 || nanos < 0 {
		sign, secs, nanos = "-", -secs, -nanos
	}
	x := fmt.Sprintf("%s%d.%09d", sign, secs, nanos)
	x = strings.TrimSuffix(x, "000")
	x = strings.TrimSuffix(x, "000")
	x = strings.TrimSuffix(x, ".000")
	return x + "s"
}

func formatProtoJSONFieldMask(fm *fieldmaskpb.FieldMask) string {
	paths := fm.GetPaths()
	cc := make([]string, len(paths))
	for i, p := range paths {
		cc[i] = jsonCamelCase(p)
	}
	return strings.Join(cc, ",")
}

// jsonCamelCase converts a snake_case identifier to a lowerCamelCase identifier,
// matching the behavior of google.golang.org/protobuf/internal/strs.JSONCamelCase.
func jsonCamelCase(s string) string {
	var b []byte
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == '_' && i+1 < len(s) {
			i++
			if s[i] >= 'a' && s[i] <= 'z' {
				b = append(b, s[i]-'a'+'A')
			} else {
				b = append(b, s[i])
			}
		} else {
			b = append(b, c)
		}
	}
	return string(b)
}

func (o MarshalOptions) marshalStruct(pm protoreflect.Message) (map[string]any, error) {
	s := pm.Interface().(*structpb.Struct)
	result := make(map[string]any, len(s.Fields))
	for k, v := range s.Fields {
		val, err := o.marshalProtoValue(v.ProtoReflect())
		if err != nil {
			return nil, err
		}
		result[k] = val
	}
	return result, nil
}

func (o MarshalOptions) marshalProtoValue(pm protoreflect.Message) (any, error) {
	v := pm.Interface().(*structpb.Value)
	switch kind := v.Kind.(type) {
	case *structpb.Value_NullValue:
		return nil, nil
	case *structpb.Value_NumberValue:
		return kind.NumberValue, nil
	case *structpb.Value_StringValue:
		return kind.StringValue, nil
	case *structpb.Value_BoolValue:
		return kind.BoolValue, nil
	case *structpb.Value_StructValue:
		return o.marshalStruct(kind.StructValue.ProtoReflect())
	case *structpb.Value_ListValue:
		return o.marshalProtoListValue(kind.ListValue.ProtoReflect())
	default:
		return nil, fmt.Errorf("protoon: unsupported structpb.Value kind %T", kind)
	}
}

func (o MarshalOptions) marshalProtoListValue(pm protoreflect.Message) ([]any, error) {
	lv := pm.Interface().(*structpb.ListValue)
	result := make([]any, 0, len(lv.Values))
	for _, v := range lv.Values {
		val, err := o.marshalProtoValue(v.ProtoReflect())
		if err != nil {
			return nil, err
		}
		result = append(result, val)
	}
	return result, nil
}

func (o MarshalOptions) marshalAny(pm protoreflect.Message) (any, error) {
	a := pm.Interface().(*anypb.Any)
	if !o.ProtoJSONCompat {
		// Non-compat mode: unwrap the embedded message.
		mt, err := protoregistry.GlobalTypes.FindMessageByURL(a.GetTypeUrl())
		if err != nil {
			return toon.NewObject(
				toon.Field{Key: "type_url", Value: a.GetTypeUrl()},
				toon.Field{Key: "value", Value: base64.StdEncoding.EncodeToString(a.GetValue())},
			), nil
		}
		m := mt.New()
		if err := proto.Unmarshal(a.GetValue(), m.Interface()); err != nil {
			return toon.NewObject(
				toon.Field{Key: "type_url", Value: a.GetTypeUrl()},
				toon.Field{Key: "value", Value: base64.StdEncoding.EncodeToString(a.GetValue())},
			), nil
		}
		return o.marshalMessage(m)
	}

	// ProtoJSON compat mode: emit @type field.
	mt, err := protoregistry.GlobalTypes.FindMessageByURL(a.GetTypeUrl())
	if err != nil {
		return toon.NewObject(
			toon.Field{Key: "@type", Value: a.GetTypeUrl()},
			toon.Field{Key: "value", Value: base64.StdEncoding.EncodeToString(a.GetValue())},
		), nil
	}
	m := mt.New()
	if err := proto.Unmarshal(a.GetValue(), m.Interface()); err != nil {
		return toon.NewObject(
			toon.Field{Key: "@type", Value: a.GetTypeUrl()},
			toon.Field{Key: "value", Value: base64.StdEncoding.EncodeToString(a.GetValue())},
		), nil
	}

	// Check if embedded type is a well-known type with custom JSON encoding.
	if wkt := wellKnownTypeFullName(mt.Descriptor().FullName()); wkt {
		v, err := o.marshalMessage(m)
		if err != nil {
			return nil, err
		}
		return toon.NewObject(
			toon.Field{Key: "@type", Value: a.GetTypeUrl()},
			toon.Field{Key: "value", Value: v},
		), nil
	}

	// Otherwise, merge @type into the embedded message fields.
	v, err := o.marshalMessage(m)
	if err != nil {
		return nil, err
	}
	obj, ok := v.(toon.Object)
	if !ok {
		// Should not happen for a regular message.
		return toon.NewObject(
			toon.Field{Key: "@type", Value: a.GetTypeUrl()},
			toon.Field{Key: "value", Value: v},
		), nil
	}
	fields := append([]toon.Field{{Key: "@type", Value: a.GetTypeUrl()}}, obj.Fields...)
	return toon.NewObject(fields...), nil
}

// wellKnownTypeFullName reports whether a message type has custom JSON
// encoding as a well-known type (excluding Struct, Value, ListValue which
// are handled inline).
func wellKnownTypeFullName(name protoreflect.FullName) bool {
	if name.Parent() != "google.protobuf" {
		return false
	}
	switch name.Name() {
	case "Any", "Timestamp", "Duration", "FieldMask", "BoolValue",
		"Int32Value", "Int64Value", "UInt32Value", "UInt64Value",
		"FloatValue", "DoubleValue", "StringValue", "BytesValue",
		"Empty":
		return true
	}
	return false
}
