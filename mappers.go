package s2s

import (
	"reflect"
	"strings"
)

//Mapper that ignores capitalization in names
func CaseInsensitiveMapper(s string) string {
	return strings.ToLower(s)
}

//Mapper that maps pointer types to direct types *T -> T using Zero(T) for nil
func MapPtrToVal(v reflect.Value, t reflect.Type) interface{} {
	if v.Kind() == reflect.Pointer && v.Type().Elem() == t {
		if v.IsNil() {
			return reflect.Zero(t).Interface()
		} else {
			return v.Elem().Interface()
		}
	}
	return v.Interface()
}

//Used to chain together multiple [ValueMapper] in the given order
func CompositeMapper(mappers... ValueMapper) ValueMapper {
	return func(v reflect.Value, t reflect.Type) interface {} {
		res := v.Interface()
		for _, m := range mappers {
			res = m(reflect.ValueOf(res), t)
		}
		return res
	}
}
