package sesame

const mapperGetterSrc = `interface {
	Get(name string) (any, error)
	GetMapperFunc(sourceName string, destName string) (any, error)
}`

const mappersSrc = `
import (
	"fmt"
	"sync"
	"time"
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

// Mappers is a collection of mappers.
type Mappers interface {
	// AddFactory adds given object factory to this mappers.
	AddFactory(name string, factory func(Mappers) (any, error))

	// AddMapperFuncFactory adds given mapper function factory to this mappers.
	AddMapperFuncFactory(sourceName string, destName string, factory func(Mappers) (any, error))

	// Get returns an object with given name.
	Get(name string) (any, error)

	// GetMapperFunc returns a mapper function with given types.
	GetMapperFunc(sourceName string, destName string) (any, error)
}

type mappers struct {
	dependencies sync.Map // [string, any]
	factories    sync.Map // [string, func(*mappers) (any, error)]
}

// NewMappers return a new [Mappers] .
func NewMappers() Mappers {
	mappers := &mappers{
		dependencies: sync.Map{},
		factories:    sync.Map{},
	}
	{{MAPPERS}}
	return mappers
}

func (d *mappers) AddFactory(name string, factory func(Mappers) (any, error)) {
	s := newSingleton[any](func() (any, error) {
		return factory(d)
	})
	d.factories.Store(name, func(di *mappers) (any, error) {
		return s.Get()
	})
}

func (d *mappers) AddMapperFuncFactory(sourceName string, destName string, factory func(Mappers) (any, error)) {
	d.AddFactory(sourceName+":"+destName, factory)
}

func (d *mappers) Get(name string) (any, error) {
	if v, ok := d.dependencies.Load(name); !ok || v == nil {
		factory, fok := d.factories.Load(name)
		if fok && factory != nil {
			obj, err := factory.(func(*mappers) (any, error))(d)
			if err != nil {
				return nil, fmt.Errorf("Failed to create a mapper: %%w", err)
			}
			d.dependencies.Store(name, obj)
		} else {
			return nil, fmt.Errorf("Object %%s not found", name)
		}
	}
	obj, _ := d.dependencies.Load(name)
	return obj, nil
}

func (d *mappers) GetMapperFunc(sourceName string, destName string) (any, error) {
	return d.Get(sourceName + ":" + destName)
}

`
