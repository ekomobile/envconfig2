package envconfig

import (
	"encoding"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"
)

// Decoder has the same semantics as Setter, but takes higher precedence.
// It is provided for historical compatibility.
type Decoder interface {
	Decode(value string) error
}

// Setter is implemented by types can self-deserialize values.
// Any type that implements flag.Value also implements Setter.
type Setter interface {
	Set(value string) error
}

// CheckDisallowed checks that no environment variables with the prefix are set
// that we don't know how or expected to parse. This is likely only meaningful with
// a non-empty prefix.
func CheckDisallowed(spec any, optsValues ...Option) error {
	opts := defaultOptions().apply(optsValues...)

	infos, err := gatherInfo(spec, opts)
	if err != nil {
		return err
	}

	vars := make(map[string]struct{})
	for _, info := range infos {
		vars[info.key] = struct{}{}
	}

	if opts.prefix != "" {
		opts.prefix = strings.ToUpper(opts.prefix) + "_"
	}

	for _, env := range os.Environ() {
		if !strings.HasPrefix(env, opts.prefix) {
			continue
		}
		v := strings.SplitN(env, "=", 2)[0]
		if _, found := vars[v]; !found {
			return fmt.Errorf("unknown environment variable %s", v)
		}
	}

	return nil
}

// Process populates the specified struct based on environment variables
func Process(spec any, optsValues ...Option) error {
	opts := defaultOptions().apply(optsValues...)

	vars, err := gatherInfo(spec, opts)
	if err != nil {
		return err
	}

	for _, v := range vars {
		value, isLoaded, valueErr := v.value()
		if valueErr != nil {
			return valueErr
		}

		if !isLoaded {
			if v.isRequired() {
				return fmt.Errorf("required key %s missing value", v.key)
			}
			continue
		}

		valueErr = processField(value, v.field)
		if valueErr != nil {
			return &ParseError{
				KeyName:   v.key,
				FieldName: v.fieldType.Name,
				TypeName:  v.field.Type().String(),
				Value:     value,
				Err:       valueErr,
			}
		}
	}

	return err
}

// MustProcess is the same as Process but panics if an error occurs
func MustProcess(spec any, options ...Option) {
	if err := Process(spec, options...); err != nil {
		panic(err)
	}
}

func processField(value string, field reflect.Value) error {
	typ := field.Type()

	decoder := decoderFrom(field)
	if decoder != nil {
		return decoder.Decode(value)
	}
	// look for Set method if Decode not defined
	setter := setterFrom(field)
	if setter != nil {
		return setter.Set(value)
	}

	if t := textUnmarshaler(field); t != nil {
		return t.UnmarshalText([]byte(value))
	}

	if b := binaryUnmarshaler(field); b != nil {
		return b.UnmarshalBinary([]byte(value))
	}

	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
		if field.IsNil() {
			field.Set(reflect.New(typ))
		}
		field = field.Elem()
	}

	switch typ.Kind() {
	case reflect.String:
		field.SetString(value)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		var (
			val int64
			err error
		)
		if field.Kind() == reflect.Int64 && typ.PkgPath() == "time" && typ.Name() == "Duration" {
			var d time.Duration
			d, err = time.ParseDuration(value)
			val = int64(d)
		} else {
			val, err = strconv.ParseInt(value, 0, typ.Bits())
		}
		if err != nil {
			return err
		}

		field.SetInt(val)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		val, err := strconv.ParseUint(value, 0, typ.Bits())
		if err != nil {
			return err
		}
		field.SetUint(val)
	case reflect.Bool:
		val, err := strconv.ParseBool(value)
		if err != nil {
			return err
		}
		field.SetBool(val)
	case reflect.Float32, reflect.Float64:
		val, err := strconv.ParseFloat(value, typ.Bits())
		if err != nil {
			return err
		}
		field.SetFloat(val)
	case reflect.Slice:
		sl := reflect.MakeSlice(typ, 0, 0)
		if typ.Elem().Kind() == reflect.Uint8 {
			sl = reflect.ValueOf([]byte(value))
		} else if strings.TrimSpace(value) != "" {
			vals := strings.Split(value, ",")
			sl = reflect.MakeSlice(typ, len(vals), len(vals))
			for i, val := range vals {
				err := processField(val, sl.Index(i))
				if err != nil {
					return err
				}
			}
		}
		field.Set(sl)
	case reflect.Map:
		mp := reflect.MakeMap(typ)
		if strings.TrimSpace(value) != "" {
			pairs := strings.Split(value, ",")
			for _, pair := range pairs {
				kvpair := strings.Split(pair, ":")
				if len(kvpair) != 2 {
					return fmt.Errorf("invalid map item: %q", pair)
				}
				k := reflect.New(typ.Key()).Elem()
				err := processField(kvpair[0], k)
				if err != nil {
					return err
				}
				v := reflect.New(typ.Elem()).Elem()
				err = processField(kvpair[1], v)
				if err != nil {
					return err
				}
				mp.SetMapIndex(k, v)
			}
		}
		field.Set(mp)
	}

	return nil
}

func interfaceFrom(field reflect.Value, fn func(interface{}, *bool)) {
	// it may be impossible for a struct field to fail this check
	if !field.CanInterface() {
		return
	}
	var ok bool
	fn(field.Interface(), &ok)
	if !ok && field.CanAddr() {
		fn(field.Addr().Interface(), &ok)
	}
}

func decoderFrom(field reflect.Value) (d Decoder) {
	interfaceFrom(field, func(v interface{}, ok *bool) { d, *ok = v.(Decoder) })
	return d
}

func setterFrom(field reflect.Value) (s Setter) {
	interfaceFrom(field, func(v interface{}, ok *bool) { s, *ok = v.(Setter) })
	return s
}

func textUnmarshaler(field reflect.Value) (t encoding.TextUnmarshaler) {
	interfaceFrom(field, func(v interface{}, ok *bool) { t, *ok = v.(encoding.TextUnmarshaler) })
	return t
}

func binaryUnmarshaler(field reflect.Value) (b encoding.BinaryUnmarshaler) {
	interfaceFrom(field, func(v interface{}, ok *bool) { b, *ok = v.(encoding.BinaryUnmarshaler) })
	return b
}

func isTrue(s string) bool {
	b, _ := strconv.ParseBool(s)
	return b
}
