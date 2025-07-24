// Package sesame provides a simple and type-safe object mapper.
package sesame

import (
	"context"
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"sync"
)

// Error is an error type for sesame.
type Error struct {
	error
	notFound    bool
	isMapper    bool
	isConverter bool
}

// NotFound returns true if the error is a not found error.
func (e *Error) NotFound() bool {
	return e.notFound
}

// IsMapper returns true if an object is a mapper, not a converter.
func (e *Error) IsMapper() bool {
	return e.isMapper
}

// IsConverter returns true if an object is a converter, not a mapper.
func (e *Error) IsConverter() bool {
	return e.isConverter
}

func (e *Error) Unwrap() error {
	return e.error
}

func merrorf(format string, a ...any) error {
	return &Error{
		error: fmt.Errorf(format, a...),
	}
}

type mfunc struct {
	SourceType reflect.Type
	DestType   reflect.Type
	Func       reflect.Method
	ObjectID   string
	Global     bool
}

type addOptions struct {
	NoGlobals bool
}

// AddOption is a type for mappers add operations.
type AddOption func(*addOptions)

// WithNoGlobals prevents to register an objects as a global one.
func WithNoGlobals() AddOption {
	return func(o *addOptions) {
		o.NoGlobals = true
	}
}

// MapperGetter is a getter interface for mappers.
type MapperGetter interface {
	// Get returns an object with given id.
	Get(id string) (any, error)

	// GetAllMappers returns an all mappers
	GetAllMappers() (map[string]any, error)

	// GetFunc returns a mapper/converter function with given type names
	// that is defined in the given mapper. If id is an empty string, it will returns
	// a global function.
	//
	//     var source MyType
	//     var dest   DestType
	//     f, err := mappers.GetFunc(reflect.TypeOf(&source), reflect.TypeOf(&dest))
	//
	// If source and dest are struct types, GetFunc will return a mapper function.
	// Otherwise, it will return a converter function.
	GetFunc(id string, sourceType reflect.Type, destType reflect.Type) (any, error)

	// GetFuncByTypeName returns a mapper/converter function with given type names
	// that is defined in the given mapper. If id is an empty string, it will returns
	// a global function.
	//
	// Type names are package path + "# + type name.
	//
	// Example: "github.com/xxx/pkg#MyType"
	//
	// Note that type names are always not a pointer type.
	// Since this method is mainly used for auto-generated mappers,
	// a type name format may change in the future.
	// So, it is recommended to use GetFunc instead of GetFuncByTypeName.
	GetFuncByTypeName(id string, sourceName string, destName string) (any, error)
}

// Mappers is a collection of mappers.
// Mapper id must end with 'Mapper'.
// Converter id must end with 'Converter'.
// MapperHelper id must end with 'Helper'.
type Mappers interface {
	MapperGetter

	// Add adds given object to this mappers.
	// Methods name like 'XxxxToYyyy' is automatically registered
	// as a global mapper/converter functions.
	Add(id string, mapper any, opts ...AddOption)

	// AddFactory adds given object factory to this mappers.
	//
	//     var mapper MyMapper
	//     f, err := mappers.AddFactory("MyMapper", reflect.TypeOf(&mapper), func(mg MapperGetter) (any, error) {
	//                   return &myMapper{}, nil
	//     })
	//
	// Methods name like 'XxxxToYyyy' is automatically registered
	// as a global mapper/converter functions.
	AddFactory(id string, typ reflect.Type, factory func(MapperGetter) (any, error),
		opts ...AddOption)

	// Merge merges given mappers into this mapper.
	// If same name object is already exists in this mappers, it will be overwritten by given mappers.
	Merge(MapperGetter) error
}

type mappers struct {
	dependencies sync.Map // [string, any]
	factories    sync.Map // [string, func(MapperGetter) (any, error)]
	findex       sync.Map // [string, []mfunc]
}

type concurrentMappers struct {
	*mappers
	lock sync.Mutex
}

// NewMappers return a new [Mappers] .
// Mappers are goroutine safe for get actions.
// Mappers are not goroutine safe for add/merge actions.
func NewMappers() Mappers {
	mappers := &mappers{
		dependencies: sync.Map{},
		factories:    sync.Map{},
		findex:       sync.Map{},
	}
	return &concurrentMappers{
		mappers: mappers,
	}

}

var funcNamePatter = regexp.MustCompile(`[A-Z][\w]+To[A-Z].*`)
var mapperNameVersionSuffixPattern = regexp.MustCompile(`[vV]\d+`)

func (d *concurrentMappers) Get(id string) (any, error) {
	d.lock.Lock()
	defer d.lock.Unlock()
	return d.mappers.Get(id)
}

func (d *mappers) Get(id string) (any, error) {
	if v, ok := d.dependencies.Load(id); !ok || v == nil {
		factory, fok := d.factories.Load(id)
		if fok && factory != nil {
			obj, err := factory.(func(MapperGetter) (any, error))(d)
			if err != nil {
				merr := &Error{
					error: err,
				}
				return nil, fmt.Errorf("failed to create a mapper: %w", merr)
			}
			d.dependencies.Store(id, obj)
		} else {
			merr := &Error{
				error:    fmt.Errorf("object %s not found", id),
				notFound: true,
			}
			return nil, merr
		}
	}
	obj, _ := d.dependencies.Load(id)
	return obj, nil
}

func (d *mappers) GetAllMappers() (map[string]any, error) {
	mappers := map[string]any{}
	var err error
	var ids []string
	d.dependencies.Range(func(key, _ any) bool {
		ids = append(ids, key.(string))
		return true
	})
	d.factories.Range(func(key, _ any) bool {
		ids = append(ids, key.(string))
		return true
	})
	for _, id := range ids {
		if strings.HasSuffix(id, "Mapper") && !strings.Contains(id, ":") {
			mappers[id], err = d.Get(id)
			if err != nil {
				return nil, err
			}
		}
	}
	return mappers, nil
}

func (d *concurrentMappers) GetAllMappers() (map[string]any, error) {
	d.lock.Lock()
	defer d.lock.Unlock()
	return d.mappers.GetAllMappers()
}

func (d *concurrentMappers) GetFunc(id string, sourceType, destType reflect.Type) (any, error) {
	d.lock.Lock()
	defer d.lock.Unlock()
	return d.mappers.GetFunc(id, sourceType, destType)
}

func (d *mappers) GetFunc(id string, sourceType, destType reflect.Type) (any, error) {
	return d.GetFuncByTypeName(id, toTypeName(sourceType), toTypeName(destType))
}

func (d *concurrentMappers) GetFuncByTypeName(id string, sourceName, destName string) (any, error) {
	d.lock.Lock()
	defer d.lock.Unlock()
	return d.mappers.GetFuncByTypeName(id, sourceName, destName)
}

func (d *mappers) GetFuncByTypeName(id string, sourceName, destName string) (any, error) {
	v, ok := d.findex.Load(toTypeIndexFromString("func:", sourceName, destName))
	if !ok {
		merr := &Error{
			error:    fmt.Errorf("functions for %s -> %s not found", sourceName, destName),
			notFound: true,
		}

		return nil, merr
	}
	lst := v.([]mfunc)
	// Merge method merges a given mapper to the end of the list, so we need to search from the end.
	for i := len(lst) - 1; i >= 0; i-- {
		if lst[i].Global && id == "" || lst[i].ObjectID == id {
			obj, err := d.Get(lst[i].ObjectID)
			if err != nil {
				return nil, err
			}
			return reflect.ValueOf(obj).MethodByName(lst[i].Func.Name).Interface(), nil
		}
	}
	merr := &Error{
		error:    fmt.Errorf("functions for %s -> %s not found", sourceName, destName),
		notFound: true,
	}

	return nil, merr
}

func (d *mappers) Add(id string, obj any, opts ...AddOption) {
	options := addOptions{}
	for _, o := range opts {
		o(&options)
	}

	d.dependencies.Store(id, obj)
	d.addMethods(id, reflect.TypeOf(obj), !options.NoGlobals)
}

func (d *mappers) AddFactory(id string, typ reflect.Type,
	factory func(MapperGetter) (any, error), opts ...AddOption) {
	options := addOptions{}
	for _, o := range opts {
		o(&options)
	}
	d.factories.Store(id, func(di MapperGetter) (any, error) {
		return factory(di)
	})
	d.addMethods(id, typ, !options.NoGlobals)
}

func (d *mappers) Merge(other MapperGetter) error {
	ms, ok := other.(interface {
		UnsafeDependencies() *sync.Map
		UnsafeFactories() *sync.Map
		UnsafeFuncIndex() *sync.Map
	})
	if !ok {
		return merrorf("can not merge %T, must be a Mappers generated by the sesame", other)
	}

	ms.UnsafeDependencies().Range(func(k, v any) bool {
		d.dependencies.Store(k, v)
		return true
	})
	ms.UnsafeFactories().Range(func(k, v any) bool {
		d.factories.Store(k, v)
		return true
	})
	ms.UnsafeFuncIndex().Range(func(k, v any) bool {
		lv := v.([]mfunc)
		fv, _ := d.findex.LoadOrStore(k, []mfunc{})
		d.findex.Store(k, append(fv.([]mfunc), lv...))
		return true
	})
	return nil
}

func (d *mappers) UnsafeDependencies() *sync.Map {
	return &d.dependencies
}

func (d *mappers) UnsafeFactories() *sync.Map {
	return &d.factories
}

func (d *mappers) UnsafeFuncIndex() *sync.Map {
	return &d.findex
}

func (d *mappers) addMethods(name string, typ reflect.Type, global bool) {
	loc := mapperNameVersionSuffixPattern.FindAllStringIndex(name, -1)
	if len(loc) > 0 {
		lastMatch := loc[len(loc)-1]
		if lastMatch[1] == len(name) {
			name = name[0:lastMatch[0]]
		}
	}
	if strings.HasSuffix(name, "Helper") {
		return
	}
	if strings.HasPrefix(name, "func:") {
		return
	}

	for _, f := range d.funcs(name, typ) {
		f.Global = global
		index := toTypeIndex("func:", f.SourceType, f.DestType)
		lv, _ := d.findex.LoadOrStore(index, []mfunc{})
		d.findex.Store(index, append(lv.([]mfunc), f))
	}
}

func (d *mappers) funcs(id string, typ reflect.Type) []mfunc {
	var funcs []mfunc
	offset := 1
	if typ.Kind() == reflect.Interface {
		offset = 0
	}
	if strings.HasSuffix(id, "Converter") {
		for i := 0; i < typ.NumMethod(); i++ {
			method := typ.Method(i)
			ft := method.Type
			if funcNamePatter.MatchString(method.Name) {
				if ft.NumIn() != (2+offset) || (ft.NumOut() != 2 && ft.NumOut() != 3) {
					continue
				}
			}
			funcs = append(funcs, mfunc{
				SourceType: ft.In(1 + offset),
				DestType:   ft.Out(0),
				Func:       method,
				ObjectID:   id,
			})

		}
	} else if strings.HasSuffix(id, "Mapper") {
		for i := 0; i < typ.NumMethod(); i++ {
			method := typ.Method(i)
			ft := method.Type
			if funcNamePatter.MatchString(method.Name) {
				if ft.NumIn() != (3+offset) || ft.NumOut() != 1 {
					continue
				}
			}
			funcs = append(funcs, mfunc{
				SourceType: ft.In(1 + offset),
				DestType:   ft.In(2 + offset),
				Func:       method,
				ObjectID:   id,
			})
		}
	}
	return funcs
}

// AddFactory adds a factory function to the given mappers.
func AddFactory[T any](mappers Mappers, id string, factory func(MapperGetter) (T, error),
	opts ...AddOption) {
	rt := getType[T]()
	mappers.AddFactory(id, rt, func(mg MapperGetter) (any, error) {
		return factory(mg)
	}, opts...)
}

// Get returns an object with given id.
func Get[T any](mappers MapperGetter, id string) (T, error) {
	var iv T
	obj, err := mappers.Get(id)
	if err != nil {
		return iv, err
	}
	v, ok := obj.(T)
	if !ok {
		return iv, merrorf("object %s is not a %T", id, iv)
	}
	return v, nil
}

// GetMapperFunc returns a mapper/converter function with given type names.
// T and U must be struct pointer types.
func GetMapperFunc[T any, U any](mappers MapperGetter,
	id string) (func(context.Context, T, U) error, error) {
	t1 := getType[T]()
	t2 := getType[U]()
	f, err := mappers.GetFunc(id, t1, t2)
	if err != nil {
		return nil, err
	}
	v, ok := f.(func(context.Context, T, U) error)
	if !ok {
		_, ok := f.(func(context.Context, T) (U, error))
		if ok {
			return nil, &Error{
				error:       fmt.Errorf("function is a converter function, not a mapper function"),
				isConverter: true,
			}
		}
		return nil, merrorf("function is not a mapper/converter function")
	}
	return v, nil
}

// GetToPrimitiveConverterFunc returns a converter function with given type names.
// T must be a pointer type. U must be a primitive type like int, string, arrays etc.
func GetToPrimitiveConverterFunc[T any, U any](mappers MapperGetter,
	id string) (func(context.Context, T) (U, bool, error), error) {
	t1 := getType[T]()
	t2 := getType[U]()
	f, err := mappers.GetFunc(id, t1, t2)
	if err != nil {
		return nil, err
	}
	v, ok := f.(func(context.Context, T) (U, bool, error))
	if !ok {
		return nil, fmt.Errorf("function is not a converter function")
	}
	return v, nil
}

// GetToObjectConverterFunc returns a converter function with given type names.
// T must be a pointer type. U must not be a primitive type like int, string, etc.
func GetToObjectConverterFunc[T any, U any](mappers MapperGetter,
	id string) (func(context.Context, T) (U, error), error) {
	t1 := getType[T]()
	t2 := getType[U]()
	f, err := mappers.GetFunc(id, t1, t2)
	if err != nil {
		return nil, err
	}
	v, ok := f.(func(context.Context, T) (U, error))
	if !ok {
		_, ok := f.(func(context.Context, T, U) error)
		if ok {
			return nil, &Error{
				error:    fmt.Errorf("function is a mapper function, not a converter function"),
				isMapper: true,
			}
		}
		return nil, merrorf("function is not a converter/mapper function")
	}
	return v, nil
}
