// Package s2s maps structs to structs using reflection
// It does this in an opinionated way by default, but is configurable
package s2s

import (
	"errors"
	"fmt"
	"reflect"
)

var ErrArgumentsInvalid = errors.New("input or output argument not pointer to struct")
var ErrInvalidConversion = errors.New("ValueMapper return can't be assigned to targetType")
var ErrMissingField = errors.New("Destination struct is missing field")

// Maps field names to allow flexibility in mapping
type NameMapper = func(string) string

// Expected to map `value` to a value of either `targetType` or `*targetType`
// Can be chained using [CompositeMapper]
type ValueMapper = func(value reflect.Value, targetType reflect.Type) interface{}

type MapperConfig struct {
	// This function if set will be called every time a field name is used
	// allowing you to map handle conversions like CamelCase -> snake_case
	NameMapper           NameMapper

	// This function if set will be called when mapping an input value 
	// to an output fields type.
	// For convenience you are allowed to return both a value directly or
	// behind a pointer.
	// Also if `MapNilToZeroImplicit` is set (default), returned typed nil
	// pointers will be implicitly mapped to the zero value if the output
	// field is not of pointer type
	ValueMapper          ValueMapper

	// If set (default) the mapper will not error on fields that can't be
	// mapped to an output field
	SkipMissingField     bool

	// If set (default) the mapper will not error if a conversion from input
	// field type to output field type fails
	SkipFailedConversion bool

	// If set (default) the mapper will implicitly convert typed nil pointers
	// to zero values of the type as needed
	MapNilToZeroImplicit bool
}

var DefaultConfig = MapperConfig{
	NameMapper:           nil,
	ValueMapper:          nil,
	SkipMissingField:     true,
	SkipFailedConversion: true,
	MapNilToZeroImplicit: true,
}

// MapStruct takes a pointer to an input struct and a pointer to an output struct.
// It will perform a mapping using the default options
func MapStruct(input interface{}, output interface{}) error {
	return mapStruct(DefaultConfig, input, output)
}

// MapStructEx takes an additional argument compared to [MapStruct] for configuration.
func MapStructEx(cfg MapperConfig, input interface{}, output interface{}) error {
	return mapStruct(cfg, input, output)
}

func mapStruct(cfg MapperConfig, input interface{}, output interface{}) error {
	iTyp, oTyp := reflect.TypeOf(input), reflect.TypeOf(output)
	// Handle untyped nil
	if iTyp == nil || oTyp == nil {
		return nil
	}

	if iTyp.Kind() != reflect.Pointer || oTyp.Kind() != reflect.Pointer {
		return ErrArgumentsInvalid
	}

	iTyp, oTyp = iTyp.Elem(), oTyp.Elem()
	if iTyp.Kind() != reflect.Struct || oTyp.Kind() != reflect.Struct {
		return ErrArgumentsInvalid
	}

	iVal := reflect.ValueOf(input).Elem()
	oVal := reflect.ValueOf(output).Elem()

	// Handle typed nil
	if !iVal.IsValid() || !oVal.IsValid() {
		return ErrArgumentsInvalid
	}

	fieldMap := map[string]int{}
	for i := 0; i < oTyp.NumField(); i++ {
		f := oTyp.Field(i)
		mappedName := f.Name
		if cfg.NameMapper != nil {
			mappedName = cfg.NameMapper(mappedName)
		}
		fieldMap[mappedName] = i
	}

	for i := 0; i < iVal.NumField(); i++ {
		iField := iTyp.Field(i)
		iFieldVal := iVal.Field(i)
		mappedName := iField.Name
		if cfg.NameMapper != nil {
			mappedName = cfg.NameMapper(mappedName)
		}

		oIdx, ok := fieldMap[mappedName]
		if !ok {
			if !cfg.SkipMissingField {
				return fmt.Errorf("%w: %q", ErrMissingField, mappedName)
			}
			continue
		}

		oField := oTyp.Field(oIdx)
		oFieldVal := oVal.Field(oIdx)

		mappedInput := iFieldVal.Interface()
		if cfg.ValueMapper != nil {
			mappedInput = cfg.ValueMapper(iFieldVal, oField.Type)
		}
		mappedInputVal := reflect.ValueOf(mappedInput)

		//Decision: Don't modify anything if the returned type is directly assignable
		if !mappedInputVal.Type().AssignableTo(oFieldVal.Type()) {
			if mappedInputVal.Kind() == reflect.Pointer && mappedInputVal.Type().Elem().AssignableTo(oFieldVal.Type()) {
				//Convenience: Allow ValueMapper to return either oField.Type or *oField.Type
				if mappedInputVal.IsNil() {
					if !cfg.MapNilToZeroImplicit {
						return fmt.Errorf("%w: Can't map <nil>(%s) to (%s)", ErrInvalidConversion, mappedInputVal.Type().Name(), oFieldVal.Type()) 
					}
					mappedInputVal = reflect.Zero(oFieldVal.Type())
				} else {
					mappedInputVal = mappedInputVal.Elem()
				}
			} else if reflect.PointerTo(mappedInputVal.Type()).AssignableTo(oFieldVal.Type()) {
				//Decision: Do not modify the things that the output object already points to
				//Convenience: If oField has type *mappedInputVal.Type, allocate a new one
				valHolder := reflect.New(mappedInputVal.Type())
				valHolder.Elem().Set(mappedInputVal)
				mappedInputVal = valHolder
			} else {
				if cfg.SkipFailedConversion {
					continue
				}
				return ErrInvalidConversion
			}
		}

		oFieldVal.Set(mappedInputVal)
	}

	return nil
}
