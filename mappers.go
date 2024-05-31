package sesame

const mapperGetterSrc = `interface {
	Get(name string) (any, error)
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

type singleton[T any] struct {
	instance T
	err      error
	factory  func() (T, error)
	once     sync.Once
}

func newSingleton[T any](fn func() (T, error)) *singleton[T] {
	return &singleton[T]{
		factory: fn,
	}
}

func (s *singleton[T]) Get() (T, error) {
	s.once.Do(func() {
		s.instance, s.err = s.factory()
	})
	return s.instance, s.err
}

func (s *singleton[T]) MustGet() T {
	instance, err := s.Get()
	if err != nil {
		panic(err)
	}
	return instance
}

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
type Mappers interface {
	// Add adds given object to this mappers.
	// Methods name like 'XxxxToYyyy' is automatically registered
	// as mapper or converter funcs.
    Add(name string, mapper any)

	// AddFactory adds given object factory to this mappers.
	AddFactory(name string, factory func(MapperGetter) (any, error))

	// AddMapperFuncFactory adds given mapper function factory to this mappers.
	AddMapperFuncFactory(sourceName string, destName string, factory func(MapperGetter) (any, error))

	// AddConverterFuncFactory adds given converter function factory to this mappers.
	AddConverterFuncFactory(sourceName string, destName string, factory func(MapperGetter) (any, error))

	// Get returns an object with given name.
	Get(name string) (any, error)

	// GetMapperFunc returns a mapper function with given types.
	GetMapperFunc(sourceName string, destName string) (any, error)

	// GetConverterFunc returns a converter function with given types.
	GetConverterFunc(sourceName string, destName string) (any, error)
}

type mappers struct {
	dependencies sync.Map // [string, any]
	factories    sync.Map // [string, func(*mappers) (any, error)]
	parent Mappers
}

type concurrentMappers struct {
	*mappers
	lock sync.Mutex
}

// NewMappers return a new [Mappers] .
func NewMappers(parent ...Mappers) Mappers {
	mappers := &mappers{
		dependencies: sync.Map{},
		factories:    sync.Map{},
	}
	if len(parent) != 0 {
		mappers.parent = parent[0]
	}
	{{MAPPERS}}
	return &concurrentMappers {
		mappers: mappers,
	}

}

var mapperFuncNamePattern = regexp.MustCompile("[A-Z][\\w]+To[A-Z].*")
var mapperNameVersionSuffixPattern = regexp.MustCompile("[vV]\\d+")

func (d *mappers) Add(name string, obj any) {
	d.dependencies.Store(name, obj)
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
	typ := reflect.TypeOf(obj)
    if strings.HasSuffix(name, "Converter") {
		for i := 0; i < typ.NumMethod(); i++ {
			method := typ.Method(i)
			if mapperFuncNamePattern.MatchString(method.Name) {
				ft := reflect.TypeOf(method.Func.Interface())
				if ft.NumIn() != 3 || ft.NumOut() != 2 {
					continue
				}
				in := ft.In(2)
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
				f := reflect.ValueOf(obj).MethodByName(method.Name).Interface()
				d.AddConverterFuncFactory(inName, outName, func(mg MapperGetter) (any, error) {
					return f, nil
				})
			}
		}
	} else if strings.HasSuffix(name, "Mapper") {
		for i := 0; i < typ.NumMethod(); i++ {
			method := typ.Method(i)
			if mapperFuncNamePattern.MatchString(method.Name) {
				ft := reflect.TypeOf(method.Func.Interface())
				if ft.NumIn() != 4 || ft.NumOut() != 1 {
					continue
				}
				in := ft.In(2)
				if in.Kind() == reflect.Ptr {
					in = in.Elem()
				}
				out := ft.In(3)
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
				f := reflect.ValueOf(obj).MethodByName(method.Name).Interface()
				d.AddMapperFuncFactory(inName, outName, func(mg MapperGetter) (any, error) {
					return f, nil
				})
			}
		}
	} else {
		panic("Unknown object type:"+name)
	}
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
			obj, err := factory.(func(*mappers) (any, error))(d)
			if err != nil {
				merr := &merror {
					error: err,
					notFound: false,
				}
				return nil, fmt.Errorf("Failed to create a mapper: %%w", merr)
			}
			d.Add(name, obj)
		} else {
			if d.parent != nil {
				obj, err := d.parent.Get(name)
				if err != nil {
					return nil, err
				}
				return obj, nil
			}
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

func (d *mappers) AddFactory(name string, factory func(MapperGetter) (any, error)) {
	s := newSingleton[any](func() (any, error) {
		return factory(d)
	})
	d.factories.Store(name, func(di *mappers) (any, error) {
		return s.Get()
	})
}

func (d *mappers) AddMapperFuncFactory(sourceName string, destName string, factory func(MapperGetter) (any, error)) {
	d.AddFactory("mapperFunc:"+sourceName+":"+destName, factory)
}

func (d *mappers) AddConverterFuncFactory(sourceName string, destName string, factory func(MapperGetter) (any, error)) {
	d.AddFactory("converterFunc:"+sourceName+":"+destName, factory)
}

func (d *mappers) GetMapperFunc(sourceName string, destName string) (any, error) {
	return d.Get("mapperFunc:"+sourceName + ":" + destName)
}

func (d *mappers) GetConverterFunc(sourceName string, destName string) (any, error) {
	return d.Get("converterFunc:"+sourceName + ":" + destName)
}
`
