package sesame

const mapperGetterSrc = `interface {
	Get(id string) (any, error)
	GetAllMappers() (map[string]any, error)
	GetFunc(sourceType reflect.Type, destType reflect.Type) (any, error)
	GetFuncByTypeName(sourceName string, destName string) (any, error)
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
// Mapper id must end with 'Mapper'.
// Converter id must end with 'Converter'.
// MapperHelper id must end with 'Helper'.
type Mappers = interface {
	// Add adds given object to this mappers.
	// Methods name like 'XxxxToYyyy' is automatically registered
	// as mapper or converter funcs.
	Add(id string, mapper any)

	// AddFactory adds given object factory to this mappers.
	//
	//     var mapper MyMapper
	//     f, err := mappers.AddFactory("MyMapper", reflect.TypeOf(&mapper), func(mg MapperGetter) (any, error) {
	//                   return &myMapper{}, nil
	//     })
	AddFactory(id string, typ reflect.Type, factory func(MapperGetter) (any, error))

	// Get returns an object with given id.
	Get(id string) (any, error)

	// GetAllMappers returns an all mappers
	GetAllMappers() (map[string]any, error)

	// GetFunc returns a mapper/converter function with given types.
	//
	//     var source MyType
	//     var dest   DestType
	//     f, err := mappers.GetFunc(reflect.TypeOf(&source), reflect.TypeOf(&dest))
	//
	// If source and dest are struct types, GetFunc will return a mapper function.
	// Otherwise, it will return a converter function.
	GetFunc(sourceType reflect.Type, destType reflect.Type) (any, error)

	// GetFuncByTypeName returns a function with given type names.
	// Type names are package path + "# + type name.
	//
	// Example: "github.com/xxx/pkg#MyType"
	//
	// Note that type names are always not a pointer type.
	// Since this method is mainly used for auto-generated mappers,
	// a type name format may change in the future.
	// So, it is recommended to use GetFunc instead of GetFuncByTypeName.
	GetFuncByTypeName(sourceName string, destName string) (any, error)

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

var funcNamePatter = regexp.MustCompile("[A-Z][\\w]+To[A-Z].*")
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
	if strings.HasPrefix(name, "func:") {
		return
	}
	offset := 1 
	if typ.Kind() == reflect.Interface {
		offset = 0
	}
    if strings.HasSuffix(name, "Converter") {
		for i := 0; i < typ.NumMethod(); i++ {
			method := typ.Method(i)
			if funcNamePatter.MatchString(method.Name) {
				ft := method.Type
				if ft.NumIn() != (2+offset) || (ft.NumOut() != 2 && ft.NumOut() != 3) {
					continue
				}
				in := ft.In(1+offset)
				out := ft.Out(0)
				d.addFuncFactory(in, out, func(mg MapperGetter) (any, error) {
					obj, _ := mg.Get(name)
				    return reflect.ValueOf(obj).MethodByName(method.Name).Interface(), nil
				})
			}
		}
	} else if strings.HasSuffix(name, "Mapper") {
		for i := 0; i < typ.NumMethod(); i++ {
			method := typ.Method(i)
			if funcNamePatter.MatchString(method.Name) {
				ft := method.Type
				if ft.NumIn() != (3+offset) || ft.NumOut() != 1 {
					continue
				}
				in := ft.In(1+offset)
				out := ft.In(2+offset)
				d.addFuncFactory(in, out, func(mg MapperGetter) (any, error) {
					obj, _ := mg.Get(name)
				    return reflect.ValueOf(obj).MethodByName(method.Name).Interface(), nil
				})
			}
		}
	} else {
		panic("Unknown object type:"+name)
	}
}

func (d *mappers) Add(id string, obj any) {
	d.dependencies.Store(id, obj)
	d.addMethods(id, reflect.TypeOf(obj))
}

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
				merr := &merror {
					error: err,
					notFound: false,
				}
				return nil, fmt.Errorf("Failed to create a mapper: %%w", merr)
			}
			d.dependencies.Store(id, obj)
		} else {
			merr := &merror {
				error: fmt.Errorf("Object %%s not found", id),
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
	d.dependencies.Range(func(key, value any) bool {
		ids = append(ids, key.(string))
		return true
	})
	d.factories.Range(func(key, value any) bool {
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

func (d *mappers) AddFactory(id string, typ reflect.Type, factory func(MapperGetter) (any, error)) {
	d.factories.Store(id, func(di MapperGetter) (any, error) {
		return factory(di)
	})
	d.addMethods(id, typ)
}

func (d *mappers) addFuncFactory(sourceType ,destType reflect.Type, factory func(MapperGetter) (any, error)) {
	sourceName := d.toTypeName(sourceType)
	destName := d.toTypeName(destType)
	d.factories.Store("func:"+sourceName+":"+destName, func(di MapperGetter) (any, error) {
		return factory(di)
	})
}

func (d *mappers) GetFunc(sourceType, destType reflect.Type) (any, error) {
	sourceName := d.toTypeName(sourceType)
	destName := d.toTypeName(destType)
	return d.GetFuncByTypeName(sourceName, destName)
}

func (d *mappers) GetFuncByTypeName(sourceName, destName string) (any, error) {
	return d.Get("func:"+sourceName + ":" + destName)
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

func (d *mappers) toTypeName(typ reflect.Type) string {
    if typ.Kind() == reflect.Ptr {
    	typ = typ.Elem()
    }
    name := typ.Name()
    if len(typ.PkgPath()) != 0 {
    	name = typ.PkgPath() + "#" + name
    }
	return name
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
func (t TypedMappers[T]) AddFactory(id string, factory func(MapperGetter) (T, error)) {
	var v T
	rt := reflect.TypeOf(&v)
	if rt.Elem().Kind() == reflect.Interface || rt.Elem().Kind() == reflect.Ptr {
		rt = rt.Elem()
	} 

	t.m.AddFactory(id, rt, func(mg MapperGetter) (any, error) {
		return factory(mg)
	})
}

// Get returns an object with given id.
func (t TypedMappers[T]) Get(id string) (T, error) {
    var iv T
	obj, err := t.m.Get(id)
	if err != nil {
		return iv, err
	}
	v, ok := obj.(T)
	if !ok {
		return iv, fmt.Errorf("object %%s is not a %%T", id, iv)
	}
	return v, nil
}

`
