package protoon

import (
	"fmt"
	"reflect"
	"strings"
	"sync"

	"github.com/toon-format/toon-go"
	"google.golang.org/protobuf/proto"
)

// MarshalAny converts any Go value to TOON, recursively converting
// proto.Message fields using protoon logic.
func MarshalAny(v any, opts ...MarshalOption) ([]byte, error) {
	return MarshalOptions{}.MarshalAny(v, opts...)
}

// MarshalAny converts any Go value to TOON.
func (o MarshalOptions) MarshalAny(v any, opts ...MarshalOption) ([]byte, error) {
	for _, opt := range opts {
		opt(&o)
	}
	if o.ProtoJSONCompat {
		o.EmitEnumNames = true
		o.EncoderOptions = append([]toon.EncoderOption{toon.WithTimeFormatter(formatProtoJSONTimestamp)}, o.EncoderOptions...)
	}
	normalized, err := o.marshalAnyValue(v)
	if err != nil {
		return nil, err
	}
	if obj, ok := normalized.(toon.Object); ok && obj.IsEmpty() {
		return []byte("{}"), nil
	}
	return toon.Marshal(normalized, o.EncoderOptions...)
}

func (o MarshalOptions) marshalAnyValue(v any) (any, error) {
	if v == nil {
		return nil, nil
	}

	// Fast path: proto.Message
	if m, ok := v.(proto.Message); ok {
		return o.marshalMessage(m.ProtoReflect())
	}

	// Known concrete types
	switch val := v.(type) {
	case string:
		return val, nil
	case bool:
		return val, nil
	case float32:
		return val, nil
	case float64:
		return val, nil
	case int:
		return int64(val), nil
	case int8:
		return int64(val), nil
	case int16:
		return int64(val), nil
	case int32:
		return int64(val), nil
	case int64:
		return val, nil
	case uint:
		return uint64(val), nil
	case uint8:
		return uint64(val), nil
	case uint16:
		return uint64(val), nil
	case uint32:
		return uint64(val), nil
	case uint64:
		return val, nil
	case toon.Object:
		return o.marshalAnyObject(val)
	case []toon.Field:
		return o.marshalAnyObject(toon.NewObject(val...))
	case map[string]any:
		return o.marshalAnyMap(val)
	case []any:
		return o.marshalAnySlice(val)
	}

	val := reflect.ValueOf(v)
	switch val.Kind() {
	case reflect.Pointer:
		if val.IsNil() {
			return nil, nil
		}
		return o.marshalAnyValue(val.Elem().Interface())
	case reflect.Slice, reflect.Array:
		return o.marshalAnyReflectSlice(val)
	case reflect.Map:
		return o.marshalAnyReflectMap(val)
	case reflect.Struct:
		return o.marshalAnyReflectStruct(val)
	case reflect.String:
		return val.String(), nil
	case reflect.Bool:
		return val.Bool(), nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return val.Int(), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return val.Uint(), nil
	case reflect.Float32:
		return val.Float(), nil
	case reflect.Float64:
		return val.Float(), nil
	default:
		return nil, fmt.Errorf("protoon: unsupported value of type %T", v)
	}
}

func (o MarshalOptions) marshalAnyObject(obj toon.Object) (toon.Object, error) {
	fields := make([]toon.Field, len(obj.Fields))
	for i, f := range obj.Fields {
		v, err := o.marshalAnyValue(f.Value)
		if err != nil {
			return toon.Object{}, err
		}
		fields[i] = toon.Field{Key: f.Key, Value: v}
	}
	return toon.NewObject(fields...), nil
}

func (o MarshalOptions) marshalAnyMap(m map[string]any) (map[string]any, error) {
	result := make(map[string]any, len(m))
	for k, v := range m {
		val, err := o.marshalAnyValue(v)
		if err != nil {
			return nil, err
		}
		result[k] = val
	}
	return result, nil
}

func (o MarshalOptions) marshalAnySlice(s []any) ([]any, error) {
	result := make([]any, len(s))
	for i, v := range s {
		val, err := o.marshalAnyValue(v)
		if err != nil {
			return nil, err
		}
		result[i] = val
	}
	return result, nil
}

func (o MarshalOptions) marshalAnyReflectSlice(val reflect.Value) ([]any, error) {
	length := val.Len()
	result := make([]any, length)
	for i := 0; i < length; i++ {
		v, err := o.marshalAnyValue(val.Index(i).Interface())
		if err != nil {
			return nil, err
		}
		result[i] = v
	}
	return result, nil
}

func (o MarshalOptions) marshalAnyReflectMap(val reflect.Value) (map[string]any, error) {
	if val.Type().Key().Kind() != reflect.String {
		return nil, fmt.Errorf("protoon: unsupported map key type %s", val.Type().Key())
	}
	result := make(map[string]any, val.Len())
	iter := val.MapRange()
	for iter.Next() {
		v, err := o.marshalAnyValue(iter.Value().Interface())
		if err != nil {
			return nil, err
		}
		result[iter.Key().String()] = v
	}
	return result, nil
}

func (o MarshalOptions) marshalAnyReflectStruct(val reflect.Value) (toon.Object, error) {
	meta := cachedStructMeta(val.Type())
	fields := make([]toon.Field, 0, len(meta.fields))
	for _, field := range meta.fields {
		childValue := fieldValueByIndex(val, field.index)
		if field.omitEmpty && isEmptyValue(childValue) {
			continue
		}
		child, err := o.marshalAnyValue(childValue.Interface())
		if err != nil {
			return toon.Object{}, fmt.Errorf("protoon: %s: %w", field.name, err)
		}
		fields = append(fields, toon.Field{Key: field.name, Value: child})
	}
	return toon.NewObject(fields...), nil
}

// structMeta mirrors toon-go's internal struct metadata caching.
type structMeta struct {
	fields []structField
}

type structField struct {
	name      string
	index     []int
	omitEmpty bool
}

var structMetaCache sync.Map // map[reflect.Type]structMeta

func cachedStructMeta(t reflect.Type) structMeta {
	if v, ok := structMetaCache.Load(t); ok {
		return v.(structMeta)
	}
	meta := computeStructMeta(t)
	structMetaCache.Store(t, meta)
	return meta
}

func computeStructMeta(t reflect.Type) structMeta {
	var fields []structField
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if f.PkgPath != "" { // unexported
			continue
		}
		if f.Anonymous {
			// Embedded struct: flatten its fields
			embedded := cachedStructMeta(f.Type)
			for _, ef := range embedded.fields {
				fields = append(fields, structField{
					name:      ef.name,
					index:     append([]int{i}, ef.index...),
					omitEmpty: ef.omitEmpty,
				})
			}
			continue
		}
		name, omitEmpty := parseStructTag(f)
		if name == "-" {
			continue
		}
		if name == "" {
			name = f.Name
		}
		fields = append(fields, structField{
			name:      name,
			index:     []int{i},
			omitEmpty: omitEmpty,
		})
	}
	return structMeta{fields: fields}
}

func parseStructTag(f reflect.StructField) (name string, omitEmpty bool) {
	tag := f.Tag.Get("toon")
	if tag == "" {
		tag = f.Tag.Get("json")
	}
	if tag == "" {
		return "", false
	}
	parts := strings.Split(tag, ",")
	name = parts[0]
	for _, p := range parts[1:] {
		if p == "omitempty" {
			omitEmpty = true
		}
	}
	return name, omitEmpty
}

func fieldValueByIndex(v reflect.Value, index []int) reflect.Value {
	for j, i := range index {
		v = v.Field(i)
		// Only dereference if this is not the last field in the index path.
		// This preserves pointer types for proto.Message assertion while
		// still flattening embedded pointer structs.
		if v.Kind() == reflect.Pointer && j < len(index)-1 {
			if v.IsNil() {
				return reflect.Value{}
			}
			v = v.Elem()
		}
	}
	return v
}

func isEmptyValue(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Array, reflect.Map, reflect.Slice, reflect.String:
		return v.Len() == 0
	case reflect.Bool:
		return !v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Interface, reflect.Pointer:
		return v.IsNil()
	}
	return false
}

// MarshalOption is a functional option for MarshalAny.
type MarshalOption func(*MarshalOptions)

// WithProtoJSONCompat enables ProtoJSON-compatible encoding for MarshalAny.
func WithProtoJSONCompat() MarshalOption {
	return func(o *MarshalOptions) {
		o.ProtoJSONCompat = true
	}
}

// WithEmitDefaultValues causes default-valued fields to be emitted.
func WithEmitDefaultValues() MarshalOption {
	return func(o *MarshalOptions) {
		o.EmitDefaultValues = true
	}
}
