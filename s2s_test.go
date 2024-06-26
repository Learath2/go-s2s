package s2s

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Don't check ExpectedOut if ExpectedErr == nil
type TestCase struct {
	Name        string
	In          interface{}
	Out         interface{}
	ExpectedErr error
	ExpectedOut interface{}
}

type SimpleFrom struct {
	Bool    bool
	String  string
	Int     int
	UInt    uint
	Float   float32
	Complex complex64
}

type SimpleTo struct {
	Bool    bool
	String  string
	Int     int
	UInt    uint
	Float   float32
	Complex complex64
}

type SimpleToMissingField struct {
	Bool   bool
	String string
	//Int   int
	UInt    uint
	Float   float32
	Complex complex64
}

type SimpleToIndirectFields struct {
	Bool    *bool
	String  *string
	Int     *int
	UInt    *uint
	Float   *float32
	Complex *complex64
}

type SimpleToDiffCase struct {
	BooL    bool
	StrIng  string
	Int     int
	Uint    uint
	Float   float32
	ComPlex complex64
}

type EmbeddedTo struct {
	SimpleTo
}

type EmbeddedFrom struct {
	SimpleFrom
}

var exampleFrom = SimpleFrom{true, "test", -10, 20, 3.14, 5 + 12i}
var expectedTo = SimpleTo{true, "test", -10, 20, 3.14, 5 + 12i}

func TestBasic(t *testing.T) {
	// The &s everywhere look nasty
	testMap := []TestCase{
		{"nil to nil", nil, nil, nil, nil},
		{"nil to struct{}", nil, &struct{}{}, nil, &struct{}{}},
		{"struct{} to nil", &struct{}{}, nil, nil, nil},
		{"struct{} to struct{}", &struct{}{}, &struct{}{}, nil, &struct{}{}},
		{"Typed nil to Typed nil", (*SimpleFrom)(nil), (*SimpleTo)(nil), ErrArgumentsInvalid, nil},
		{"int to int", 10, 20, ErrArgumentsInvalid, nil},
		{"*int to *int", new(int), new(int), ErrArgumentsInvalid, nil},
		{"SimpleFrom to struct{}}",
			&exampleFrom,
			&struct{}{},
			nil,
			&struct{}{},
		},
		{"SimpleFrom to SimpleTo",
			&exampleFrom,
			&SimpleTo{},
			nil,
			&expectedTo,
		},
		{"SimpleFrom to SimpleToMissingField",
			&exampleFrom,
			&SimpleToMissingField{},
			nil,
			&SimpleToMissingField{expectedTo.Bool, expectedTo.String, /*expectedTo.Int,*/
				expectedTo.UInt, expectedTo.Float, expectedTo.Complex},
		},
		{"SimpleFrom to SimpleToIndirectFields",
			&exampleFrom,
			&SimpleToIndirectFields{},
			nil,
			&SimpleToIndirectFields{&expectedTo.Bool, &expectedTo.String, &expectedTo.Int,
				&expectedTo.UInt, &expectedTo.Float, &expectedTo.Complex},
		},
	}

	runFunc := func(t *testing.T, c *TestCase) {
		err := MapStruct(c.In, c.Out)
		if c.ExpectedErr == nil {
			assert.Nil(t, err)
			assert.Equal(t, c.ExpectedOut, c.Out)
		} else {
			assert.ErrorIs(t, err, c.ExpectedErr)
		}
	}

	for i := 0; i < len(testMap); i++ {
		k := i
		t.Run(testMap[k].Name, func(t *testing.T) {
			//t.Parallel()
			runFunc(t, &(testMap[k]))
		})
	}
}

type TestCaseEx struct {
	TestCase
	Cfg *MapperConfig
}

func TestEmbeds(t *testing.T) {
	t.Run("SimpleFrom to EmbeddedTo", func(t *testing.T) {
		var to EmbeddedTo
		err := MapStruct(&exampleFrom, &to)
		assert.Nil(t, err)
		assert.Equal(t, expectedTo, to.SimpleTo)
	})

	t.Run("EmbeddedFrom to SimpleTo", func(t *testing.T) {
		var to SimpleTo
		err := MapStruct(&EmbeddedFrom{exampleFrom}, &to)
		assert.Nil(t, err)
		assert.Equal(t, expectedTo, to)
	})
}

func TestEx(t *testing.T) {
	//defCfg := DefaultConfig
	const testStr = "foo"
	testMap := []TestCaseEx{
		{TestCase{"SimpleFrom to SimpleToMissingField",
			&exampleFrom,
			&SimpleToMissingField{},
			ErrMissingField,
			&SimpleToMissingField{expectedTo.Bool, expectedTo.String, /*expectedTo.Int,*/
				expectedTo.UInt, expectedTo.Float, expectedTo.Complex}},
			&MapperConfig{
				SkipMissingField:     false,
				SkipFailedConversion: true,
				MapNilToZeroImplicit: true,
			},
		},
		{TestCase{"SimpleFrom to SimpleToDiffCase",
			&exampleFrom,
			&SimpleToDiffCase{},
			nil,
			&SimpleToDiffCase{expectedTo.Bool, expectedTo.String, expectedTo.Int,
				expectedTo.UInt, expectedTo.Float, expectedTo.Complex}},
			&MapperConfig{
				NameMapper:           CaseInsensitiveMapper,
				SkipMissingField:     true,
				SkipFailedConversion: true,
				MapNilToZeroImplicit: true,
			},
		},
		{TestCase{"ValueMapper(map all ints to 42)",
			&exampleFrom,
			&SimpleTo{},
			nil,
			&SimpleTo{expectedTo.Bool, expectedTo.String, 42,
				expectedTo.UInt, expectedTo.Float, expectedTo.Complex}},
			&MapperConfig{
				ValueMapper: func(v reflect.Value, t reflect.Type) any {
					switch t.Kind() {
					case reflect.Int:
						return 42
					default:
						return v.Interface()
					}
				},
				SkipMissingField:     true,
				SkipFailedConversion: true,
				MapNilToZeroImplicit: true,
			},
		},
		{TestCase{"ValueMapper(test return ptr convenience)",
			&exampleFrom,
			&SimpleTo{},
			nil,
			&SimpleTo{*new(bool), testStr, expectedTo.Int,
				expectedTo.UInt, expectedTo.Float, expectedTo.Complex}},
			&MapperConfig{
				ValueMapper: func(v reflect.Value, t reflect.Type) any {
					switch t.Kind() {
					// Test non-nil
					case reflect.String:
						rvh := testStr
						return &rvh
					// Test nil to zero convenience aswell
					case reflect.Bool:
						return (*bool)(nil)
					default:
						return v.Interface()
					}
				},
				SkipMissingField:     true,
				SkipFailedConversion: true,
				MapNilToZeroImplicit: true,
			},
		},
		{TestCase{"ValueMapper(test nil to zero non implicit)",
			&exampleFrom,
			&SimpleTo{},
			ErrInvalidConversion,
			nil},
			&MapperConfig{
				ValueMapper: func(v reflect.Value, t reflect.Type) any {
					switch t.Kind() {
					// Test nil to zero convenience
					case reflect.Bool:
						return (*bool)(nil)
					default:
						return v.Interface()
					}
				},
				SkipMissingField:     true,
				SkipFailedConversion: true,
				MapNilToZeroImplicit: false,
			},
		},
		{TestCase{"ValueMapper(test failed conversion not-allowed)",
			&exampleFrom,
			&SimpleTo{},
			ErrInvalidConversion,
			nil},
			&MapperConfig{
				ValueMapper: func(v reflect.Value, t reflect.Type) any {
					switch t.Kind() {
					//Test invalid conversion int -> string
					case reflect.Int:
						return ""
					default:
						return v.Interface()
					}
				},
				SkipMissingField:     true,
				SkipFailedConversion: false,
				MapNilToZeroImplicit: true,
			},
		},
		{TestCase{"ValueMapper(test failed conversion allowed)",
			&exampleFrom,
			&SimpleTo{},
			nil,
			&SimpleTo{expectedTo.Bool, expectedTo.String, *new(int),
				expectedTo.UInt, expectedTo.Float, expectedTo.Complex}},
			&MapperConfig{
				ValueMapper: func(v reflect.Value, t reflect.Type) any {
					switch t.Kind() {
					//Test invalid conversion int -> string
					case reflect.Int:
						return ""
					default:
						return v.Interface()
					}
				},
				SkipMissingField:     true,
				SkipFailedConversion: true,
				MapNilToZeroImplicit: true,
			},
		},
	}

	runFunc := func(t *testing.T, c *TestCaseEx) {
		err := MapStructEx(*c.Cfg, c.In, c.Out)
		if c.ExpectedErr == nil {
			assert.Nil(t, err)
			assert.Equal(t, c.ExpectedOut, c.Out)
		} else {
			assert.ErrorIs(t, err, c.ExpectedErr)
		}
	}

	for i := 0; i < len(testMap); i++ {
		k := i
		t.Run(testMap[k].Name, func(t *testing.T) {
			//t.Parallel()
			runFunc(t, &(testMap[k]))
		})
	}
}

func TestExSpy(t *testing.T) {
	type vLogEntry struct {
		v interface{}
		t reflect.Type
	}
	vLog := []vLogEntry{}
	spyVMapper := func(v reflect.Value, t reflect.Type) any {
		vLog = append(vLog, vLogEntry{v.Interface(), t})
		return v.Interface()
	}

	nLog := []string{}
	spyNMapper := func(s string) string {
		nLog = append(nLog, s)
		return s
	}

	cfg := DefaultConfig
	cfg.NameMapper = spyNMapper
	cfg.ValueMapper = spyVMapper

	t.Run("Spy on hooks", func(t *testing.T) {
		err := MapStructEx(cfg, &exampleFrom, &SimpleTo{})
		assert.Nil(t, err)

		assert.Equal(t, []string{
			"Bool", "String", "Int", "UInt", "Float", "Complex",
			"Bool", "String", "Int", "UInt", "Float", "Complex",
		}, nLog)
		assert.Equal(t, []vLogEntry{
			{true, reflect.TypeOf(true)},
			{"test", reflect.TypeOf("test")},
			{-10, reflect.TypeOf(-10)},
			{uint(20), reflect.TypeOf(uint(20))},
			{float32(3.14), reflect.TypeOf(float32(3.14))},
			{complex64(5 + 12i), reflect.TypeOf(complex64(5 + 12i))},
		}, vLog)
	})
}

func TestIncludedMappers(t *testing.T) {
	t.Run("Composite Mapper", func(t *testing.T) {
		log := []string{}
		aMapper := func(v reflect.Value, t reflect.Type) any {
			log = append(log, "a")
			return v.Interface()
		}

		bMapper := func(v reflect.Value, t reflect.Type) any {
			log = append(log, "b")
			return v.Interface()
		}

		cfg := DefaultConfig
		cfg.ValueMapper = CompositeMapper(aMapper, bMapper)

		err := MapStructEx(cfg, &exampleFrom, &SimpleTo{})
		assert.Nil(t, err)

		assert.Equal(t, []string{
			"a", "b", "a", "b",
			"a", "b", "a", "b",
			"a", "b", "a", "b",
		}, log)

		log = []string{}
		cfg.ValueMapper = CompositeMapper(bMapper, aMapper)
		err = MapStructEx(cfg, &exampleFrom, &SimpleTo{})
		assert.Nil(t, err)

		assert.Equal(t, []string{
			"b", "a", "b", "a",
			"b", "a", "b", "a",
			"b", "a", "b", "a",
		}, log)
	})
}
