package sesame

const mapperGetterSrc = `interface {
	Get(name string) (any, error)
	GetAllMappers() (map[string]any, error)
	GetMapperFunc(sourceName string, destName string) (any, error)
	GetConverterFunc(sourceName string, destName string) (any, error)
}`

const mappersSrc = `
import (
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"sync"
	{{IMPORTS}}
)

type merror struct {
	error
	notFound bool
}

func (e *merror) NotFound() bool {
	return e.notFound
}

func (e *merror) Unwrap() error {
	return e.error
}

type MapperGetter = ` + mapperGetterSrc + `

// Mappers is a collection of mappers.
// Mapper name must end with 'Mapper'.
// Converter name must end with 'Converter'.
// MapperHelper name must end with 'Helper'.
type Mappers = interface {
	// Add adds given object to this mappers.
	// Methods name like 'XxxxToYyyy' is automatically registered
	// as mapper or converter funcs.
    Add(name string, mapper any)

	// AddFactory adds given object factory to this mappers.
	AddFactory(name string, typ reflect.Type, factory func(MapperGetter) (any, error))

	// Get returns an object with given name.
	Get(name string) (any, error)

	// GetAllMappers returns an all mappers
	GetAllMappers() (map[string]any, error)

	// GetMapperFunc returns a mapper function with given types.
	GetMapperFunc(sourceName string, destName string) (any, error)

	// GetConverterFunc returns a converter function with given types.
	GetConverterFunc(sourceName string, destName string) (any, error)

	// Merge merges given mappers into this mapper.
	// If same name object is already exists in this mappers, it will be overwritten by given mappers.
	Merge(MapperGetter) error
}

type mappers struct {
	dependencies sync.Map // [string, any]
	factories    sync.Map // [string, func(MapperGetter) (any, error)]
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
	}
	{{MAPPERS}}
	return &concurrentMappers {
		mappers: mappers,
	}

}

var mapperFuncNamePattern = regexp.MustCompile("[A-Z][\\w]+To[A-Z].*")
var mapperNameVersionSuffixPattern = regexp.MustCompile("[vV]\\d+")

func (d *mappers) UnsafeDependencies () *sync.Map {
	return &d.dependencies
}

func (d *mappers) UnsafeFactories () *sync.Map {
	return &d.factories
}

func (d *mappers) addMethods(name string, typ reflect.Type) {
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
	if strings.HasPrefix(name, "mapperFunc:") {
		return
	}
	if strings.HasPrefix(name, "converterFunc:") {
		return
	}
	offset := 1 
	if typ.Kind() == reflect.Interface {
		offset = 0
	}
    if strings.HasSuffix(name, "Converter") {
		for i := 0; i < typ.NumMethod(); i++ {
			method := typ.Method(i)
			if mapperFuncNamePattern.MatchString(method.Name) {
				ft := method.Type
				if ft.NumIn() != (2+offset) || (ft.NumOut() != 2 && ft.NumOut() != 3) {
					continue
				}
				in := ft.In(1+offset)
				if in.Kind() == reflect.Ptr {
					in = in.Elem()
				}
				out := ft.Out(0)
				if out.Kind() == reflect.Ptr {
					out = out.Elem()
				}
				inName := in.Name()
				if len(in.PkgPath()) != 0 {
					inName = in.PkgPath() + "#" + inName
				}
				outName := out.Name()
				if len(out.PkgPath()) != 0 {
					outName = out.PkgPath() + "#" + outName
				}
				d.addConverterFuncFactory(inName, outName, func(mg MapperGetter) (any, error) {
					obj, _ := mg.Get(name)
				    return reflect.ValueOf(obj).MethodByName(method.Name).Interface(), nil
				})
			}
		}
	} else if strings.HasSuffix(name, "Mapper") {
		for i := 0; i < typ.NumMethod(); i++ {
			method := typ.Method(i)
			if mapperFuncNamePattern.MatchString(method.Name) {
				ft := method.Type
				if ft.NumIn() != (3+offset) || ft.NumOut() != 1 {
					continue
				}
				in := ft.In(1+offset)
				if in.Kind() == reflect.Ptr {
					in = in.Elem()
				}
				out := ft.In(2+offset)
				if out.Kind() == reflect.Ptr {
					out = out.Elem()
				}
				inName := in.Name()
				if len(in.PkgPath()) != 0 {
					inName = in.PkgPath() + "#" + inName
				}
				outName := out.Name()
				if len(out.PkgPath()) != 0 {
					outName = out.PkgPath() + "#" + outName
				}
				d.addMapperFuncFactory(inName, outName, func(mg MapperGetter) (any, error) {
					obj, _ := mg.Get(name)
				    return reflect.ValueOf(obj).MethodByName(method.Name).Interface(), nil
				})
			}
		}
	} else {
		panic("Unknown object type:"+name)
	}
}

func (d *mappers) Add(name string, obj any) {
	d.dependencies.Store(name, obj)
	d.addMethods(name, reflect.TypeOf(obj))
}

func (d *concurrentMappers) Get(name string) (any, error) {
	d.lock.Lock()
	defer d.lock.Unlock()
	return d.mappers.Get(name)
}

func (d *mappers) Get(name string) (any, error) {
	if v, ok := d.dependencies.Load(name); !ok || v == nil {
		factory, fok := d.factories.Load(name)
		if fok && factory != nil {
			obj, err := factory.(func(MapperGetter) (any, error))(d)
			if err != nil {
				merr := &merror {
					error: err,
					notFound: false,
				}
				return nil, fmt.Errorf("Failed to create a mapper: %%w", merr)
			}
			d.dependencies.Store(name, obj)
		} else {
			merr := &merror {
				error: fmt.Errorf("Object %%s not found", name),
				notFound: true,
			}
			return nil, merr
		}
	}
	obj, _ := d.dependencies.Load(name)
	return obj, nil
}

func (d *mappers) GetAllMappers() (map[string]any, error) {
	mappers := map[string]any{}
	var err error
	var names []string
	d.dependencies.Range(func(key, value any) bool {
		names = append(names, key.(string))
		return true
	})
	d.factories.Range(func(key, value any) bool {
		names = append(names, key.(string))
		return true
	})
	for _, name := range names {
		if strings.HasSuffix(name, "Mapper") && !strings.Contains(name, ":") {
		    mappers[name], err = d.Get(name)
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

func (d *mappers) AddFactory(name string, typ reflect.Type, factory func(MapperGetter) (any, error)) {
	d.factories.Store(name, func(di MapperGetter) (any, error) {
		return factory(di)
	})
	d.addMethods(name, typ)
}

func (d *mappers) addMapperFuncFactory(sourceName string, destName string, factory func(MapperGetter) (any, error)) {
	d.factories.Store("mapperFunc:"+sourceName+":"+destName, func(di MapperGetter) (any, error) {
		return factory(di)
	})
}

func (d *mappers) addConverterFuncFactory(sourceName string, destName string, factory func(MapperGetter) (any, error)) {
	d.factories.Store("converterFunc:"+sourceName+":"+destName, func(di MapperGetter) (any, error) {
		return factory(di)
	})
}

func (d *mappers) GetMapperFunc(sourceName string, destName string) (any, error) {
	return d.Get("mapperFunc:"+sourceName + ":" + destName)
}

func (d *mappers) GetConverterFunc(sourceName string, destName string) (any, error) {
	return d.Get("converterFunc:"+sourceName + ":" + destName)
}

func (d *mappers) Merge(other MapperGetter) error {
	ms, ok := other.(interface {
      UnsafeDependencies () *sync.Map 
      UnsafeFactories () *sync.Map 
    })

	if !ok {
		return fmt.Errorf("can not merge %%T, must be a mapper generated by the sesame", other)
	}

	ms.UnsafeDependencies().Range(func(k, v interface{}) bool {
		d.dependencies.Store(k, v)
		return true
	})
	ms.UnsafeFactories().Range(func(k, v interface{}) bool {
		d.factories.Store(k, v)
		return true
	})
	return nil
}

// TypedMappers is a type-safe mappers.
type TypedMappers[T any] struct {
	m Mappers
}

// NewTypedMappers returns a new TypedMappers.
func NewTypedMappers[T any](m Mappers) TypedMappers[T] {
	return TypedMappers[T]{m: m}
}

// AddFactory adds a factory function to this mappers.
func (t TypedMappers[T]) AddFactory(name string, factory func(MapperGetter) (T, error)) {
	var v T
	rt := reflect.TypeOf(&v)
	if rt.Elem().Kind() == reflect.Interface || rt.Elem().Kind() == reflect.Ptr {
		rt = rt.Elem()
	} 

	t.m.AddFactory(name, rt, func(mg MapperGetter) (any, error) {
		return factory(mg)
	})
}

// Get returns an object with given name.
func (t TypedMappers[T]) Get(name string) (T, error) {
    var iv T
	obj, err := t.m.Get(name)
	if err != nil {
		return iv, err
	}
	v, ok := obj.(T)
	if !ok {
		return iv, fmt.Errorf("object %%s is not a %%T", name, iv)
	}
	return v, nil
}

`
