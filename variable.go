package envconfig

import (
	"fmt"
	"os"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

const (
	TagEnvconfig  = "envconfig"
	TagIgnored    = "ignored"
	TagDefault    = "default"
	TagSplitWords = "split_words"
	TagRequired   = "required"
	TagFile       = "file"
)

// variable maintains information about the configuration variable
type variable struct {
	key       string
	altKey    string
	fieldType reflect.StructField
	field     reflect.Value
	// Tags      reflect.StructTag
	Opts *options
}

// GatherInfo gathers information about the specified struct
func gatherInfo(spec any, opts *options) (vars []*variable, err error) {
	s := reflect.ValueOf(spec)

	if s.Kind() != reflect.Ptr {
		return nil, ErrInvalidSpecification
	}
	s = s.Elem()
	if s.Kind() != reflect.Struct {
		return nil, ErrInvalidSpecification
	}
	typeOfSpec := s.Type()

	// over allocate an info array, we will extend if needed later
	vars = make([]*variable, 0, s.NumField())

	for i := 0; i < s.NumField(); i++ {
		field := s.Field(i)
		fieldType := typeOfSpec.Field(i)
		if !field.CanSet() || isTrue(fieldType.Tag.Get(TagIgnored)) {
			continue
		}

		for field.Kind() == reflect.Ptr {
			if field.IsNil() {
				if field.Type().Elem().Kind() != reflect.Struct {
					// nil pointer to a non-struct: leave it alone
					break
				}
				// nil pointer to struct: create a zero instance
				field.Set(reflect.New(field.Type().Elem()))
			}
			field = field.Elem()
		}

		// Capture information about the config varItem
		varItem := variable{
			field:     field,
			fieldType: fieldType,
			// Tags:      fieldType.Tag,
			Opts: opts,
		}

		varItem.key, varItem.altKey = resolveKey(varItem.Opts.prefix, fieldType)

		vars = append(vars, &varItem)

		if field.Kind() == reflect.Struct {
			// honor Decode if present
			if decoderFrom(field) == nil && setterFrom(field) == nil && textUnmarshaler(field) == nil && binaryUnmarshaler(field) == nil {
				innerOpts := opts.copy()
				if !fieldType.Anonymous {
					innerOpts.prefix = varItem.key
				}

				embeddedPtr := field.Addr().Interface()
				embeddedVars, recursionErr := gatherInfo(embeddedPtr, innerOpts)
				if recursionErr != nil {
					return nil, recursionErr
				}
				vars = append(vars[:len(vars)-1], embeddedVars...)
			}
		}
	}

	return vars, nil
}

func (v *variable) isRequired() bool {
	return isTrue(v.fieldType.Tag.Get(TagRequired))
}

func (v *variable) value() (value string, isLoaded bool, err error) {
	envNames := []string{v.key}

	if v.altKey != "" {
		envNames = append(envNames, v.altKey)
	}

	for _, envName := range envNames {
		value, isLoaded, err = v.tryEnv(envName)
		if err != nil {
			return
		}
		if isLoaded { // Found some value
			break
		}
	}

	// Trim space
	if isLoaded && v.Opts.trimSpaces {
		value = strings.TrimSpace(value)
	}

	// Load default value
	if !isLoaded {
		value, isLoaded = v.fieldType.Tag.Lookup(TagDefault)
	}

	return
}

func (v *variable) tryEnv(envName string) (value string, isLoaded bool, err error) {
	// ENV value
	if value, isLoaded = os.LookupEnv(envName); isLoaded {
		return
	}

	// Load from file
	return v.loadFromFile(envName)
}

func (v *variable) loadFromFile(envName string) (value string, isLoaded bool, err error) {
	tagValue, needLoad := v.resolveFileLoading()
	if !needLoad {
		return
	}

	tagValue = strings.TrimSpace(tagValue)
	if tagValue == "" {
		tagValue = v.Opts.defaultFileSuffix
	}

	var filePath string
	var isFilePathLoaded bool

	// Try to acquire file path from env named by `{v.EnvNames}_{tagValue}`
	var fileEnvName = strings.ToUpper(envName + tagValue)
	if filePath, isFilePathLoaded = os.LookupEnv(fileEnvName); isFilePathLoaded {
		filePath = strings.TrimSpace(filePath)

		// if envName is set it must contain a file path
		if filePath == "" {
			err = fmt.Errorf("environment vairable %s is empty", tagValue)
			return
		}
	}

	if !isFilePathLoaded {
		return
	}

	// try file
	bytes, err := os.ReadFile(filePath)
	if err != nil {
		return
	}
	value = string(bytes)
	isLoaded = true

	return
}

func (v *variable) resolveFileLoading() (tagValue string, needLoad bool) {
	// Loading from file
	if tagFileValue, tagFileExists := v.fieldType.Tag.Lookup(TagFile); tagFileExists { // if file tag exists
		// check if it is purely bool
		tagFileBool, tagFileBoolErr := strconv.ParseBool(tagFileValue)
		if tagFileBoolErr == nil { // if it's a boolean
			// enable file loading if it's "true"
			if tagFileBool {
				return "", true
			}
		} else { // if it contains non-bool value, return the value
			return tagFileValue, true
		}
	} else { // no file tag
		// accept defaults
		if v.Opts.isLoadFromFile {
			return "", true
		}
	}

	return "", false
}

func resolveKey(prefix string, fieldType reflect.StructField) (key, altKey string) {
	altKey = strings.TrimSpace(fieldType.Tag.Get(TagEnvconfig))

	if altKey != "" {
		altKey = strings.ToUpper(altKey)
		key = altKey

	} else {
		// Best effort to un-pick camel casing as separate words
		if isTrue(fieldType.Tag.Get(TagSplitWords)) {
			key = strings.Join(splitWords(fieldType.Name), "_")
		} else {
			key = fieldType.Name
		}
	}

	if prefix != "" {
		key = prefix + "_" + key
	}

	key = strings.ToUpper(key)

	return
}

var gatherRegexp = regexp.MustCompile("([^A-Z]+|[A-Z]+[^A-Z]+|[A-Z]+)")
var acronymRegexp = regexp.MustCompile("([A-Z]+)([A-Z][^A-Z]+)")

func splitWords(in string) (out []string) {
	matches := gatherRegexp.FindAllStringSubmatch(in, -1)
	if len(matches) > 0 {
		for _, words := range matches {
			if m := acronymRegexp.FindStringSubmatch(words[0]); len(m) == 3 {
				out = append(out, m[1], m[2])
			} else {
				out = append(out, words[0])
			}
		}
	}

	return
}
