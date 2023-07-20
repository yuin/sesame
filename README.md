# sesame

[![https://pkg.go.dev/github.com/yuin/sesame](https://pkg.go.dev/badge/github.com/yuin/sesame.svg)](https://pkg.go.dev/github.com/yuin/sesame)
[![https://github.com/yuin/sesame/actions?query=workflow:test](https://github.com/yuin/sesame/workflows/test/badge.svg?branch=master&event=push)](https://github.com/yuin/sesame/actions?query=workflow:test)
[![https://goreportcard.com/report/github.com/yuin/sesame](https://goreportcard.com/badge/github.com/yuin/sesame)](https://goreportcard.com/report/github.com/yuin/sesame)

> An object-to-object mapper generator for Go that can 'scale'

## Why the name 'sesame'?
sesame is a **go** **ma**pper. Japanese often abbreviate this kind of terms as 'goma'.
[Goma](https://ja.wikipedia.org/wiki/%E3%82%B4%E3%83%9E) means the sesame in Japanese.


## Motivation
Multitier architectures like good-old 3tier architecture, Clean architecture, Hexagonal architecture etc have similar objects in each layers.

It is a hard work that you must write so many bolierplate code to map these objects. There are some kind of libraries(an object-to-object mapper) that simplify this job using reflections.

Object-to-object mappers that use reflections are very easy to use, but these are difficult to 'scale' .

- Hard to debug: Objects in the real world, rather than examples, are often very large. In reflection-based libraries, it can be quite hard to debug if some fields are not mapped correctly.
- Performance: Go's reflection is fast enough for most usecases. But, yes, large applications that selects multitier architectures often have very large objects. Many a little makes a mickle.

sesame generates object-to-object mappers source codes that **DO NOT** use reflections.

## Status
This project is in very early stage. 
Any kind of feedbacks are wellcome.

## Features

- **Fast** : sesame generates object-to-object mappers source codes that **DO NOT** use reflections.
- **Easy to debug** : If some fields are not mapped correctly, you just look a generated mapper source codes.
- **Flexible** : sesame provides various way to map objects.
  - By name
    - Simple field to field mapping
    - Field to nesting field mapping like `TodoModel.UserID -> TodoEntity.User.ID` .
    - Embedded struct mapping
  - By type
  - By helper function that is written in Go
- **Zero 3rd-party dependencies at runtime** : sesame generates codes that depend only standard libraries.
- **Scalable** : 
  - Fast, Easy to debug and flexible. 
  - Mapping configurations can be separated into multiple files.
    - You do not have to edit over 10000 lines of a single YAML file that has 1000 mapping definitions.

## Current limitations
- Your project must be a Go module.

## Installation
### Binary installation
Get a binary from [releases](https://github.com/yuin/sesame/releases) .

### go get
sesame requires Go 1.20+.

```go
$ go install github.com/yuin/sesame/cmd/sesame@latest
```

## Usage
### Outline

1. Create a configuration file(s).
2. Run the `sesame` command.
3. Create a `Mappers` object in your code.
4. (Optional) Add helpers and custom mappers.
5. Get a mapper from `Mappers`.
6. Map objects by the mapper.

See [tests](https://github.com/yuin/sesame/tree/master/testdata) for examples.

### Mapping configuration file
sesame uses mapping configuration files written in YAML. 

`${YOUR_GO_MODULE_ROOT}/sesame.yml`:

```yaml
mappers:                                         # configurations for a mapper collection
  package: mapper                                # package name for generated mapper collection
  destination: ./mapper/mappers_gen.go           # destination for generation
mappings:                                        # configurations for object-to-object mappings
  - name: TodoMapper                             # name of the mapper. This must be unique within all mappers
    package: mapper                              # package name for generated mapper
    destination: ./mapper/todo_mapper_gen.go     # definition for generation
    bidirectional: true                          # generates a-to-b and b-to-a mapping if true(default: false)
    a-to-b: ModelToEntity                        # mapping function name(default: `{AName}To{BName}`)
    b-to-a: EntityToModel                        # mapping function name(default: `{BName}To{AName}`)
    a:                                           # mapping operand A
      package: ./model                           # package path for this operand
      name: TodoModel                            # struct name of this operand
    b:                                           # mapping operand B
      package: ./domain
      name: Todo
    explicitOnly: false                          # sesame maps same names automatically if false(default: false)
    fields:                                      # relationships between A fields and B fields
      - a: Done                                  # you can define nested mappings like UserID
        b: Finished                              # you can define mappings for embedded structs by '*'
      - a: UserID                                # 
        b: User.ID                               #
    ignores:                                     # ignores fields in operand X
      - a: ValidateOnly
_includes:                                       # includes separated configuration files
  - ./*/**/*_sesame.yml
```

And now, you can generate source codes just run `sesame` command in `${YOUR_GO_MODULE_ROOT}`.

This configuration will generate the codes like the following:

`./mapper/todo_mapper_gen.go` :

```go
package mapper

import (
    pkg00002 "time"

    pkg00001 "example.com/testmod/domain"
    pkg00000 "example.com/testmod/model"
)

type TodoMapperHelper interface {
    ModelToEntity(*pkg00000.TodoModel, *pkg00001.Todo) error
    EntityToModel(*pkg00001.Todo, *pkg00000.TodoModel) error
}

type TodoMapper interface {
    ModelToEntity(*pkg00000.TodoModel) (*pkg00001.Todo, error)
    EntityToModel(*pkg00001.Todo) (*pkg00000.TodoModel, error)
}

// ... (TodoMapper default implementation)
```

### Mapping in your code
sesame generates a mapper collection into the `mappers.destination` .
Mapping codes look like the following:

1. Create new Mappers object as a singleton object. The Mappers object is a groutine-safe.

   ````go
   mappers := mapper.NewMappers()           # Creates new Mappers object
   mapper.AddTimeToStringMapper(mappers)    # Add custom mappers
   mappers.AddFactory("TodoMapperHelper", func(ms MapperGetter) (any, error) {
       return &todoMapperHelper{}, nil
   }) # Add helpers

   ```

2. Get a mapper and call it for mapping.

   ```go
   obj, err := mappers.Get("TodoMapper")    # Get mapper by its name
   if err != nil {
       t.Fatal(err)
   }
   todoMapper, _ := obj.(TodoMapper)
   entity, err := todoMapper.ModelToEntity(model) 
   ```

### Custom mappers
By default, sesame can map following types:

- Same types
- Castable types(i.e. `int -> int64`, `type MyType int <-> int`)
- `map`, `slice` and `array`

For others, you can write and register custom mappers.

Example: `string <-> time.Time` mapper

```go
type TimeStringMapper struct {
}

func (m *TimeStringMapper) StringToTime(source string) (*time.Time, error) {
    t, err := time.Parse(time.RFC3339, source)
    if err != nil {
        return nil, err
    }
    return &t, nil
}

func (m *TimeStringMapper) TimeToString(source *time.Time) (string, error) {
    return source.Format(time.RFC3339), nil
}

type Mappers interface {
    AddFactory(string, func(MapperGetter) (any, error))
    AddMapperFuncFactory(string, string, func(MapperGetter) (any, error))
}

func AddTimeToStringMapper(mappers Mappers) {
    stringTime := &TimeStringMapper{}
    mappers.AddFactory("TimeStringMapper", func(m MapperGetter) (any, error) {
        return stringTime, nil
    })
    mappers.AddMapperFuncFactory("string", "time#Time", func(m MapperGetter) (any, error) {
        return stringTime.StringToTime, nil
    })
    mappers.AddMapperFuncFactory("time#Time", "string", func(m MapperGetter) (any, error) {
        return stringTime.TimeToString, nil
    })
}
```

`Mappers.AddMapperFuncFactory` takes qualified type names as arguments. A qualified type name is `FULL_PACKAGE_PATH#TYPENAME`(i.e. `time#Time`, `example.com/testmod/domain#Todo`).

Argument types and return types in custom mapping functions must be a

- Raw value: primitive types(i.e. `string`, `int`, `slice` ...)
- Pointer: others

So `func (m *TimeStringMapper) TimeToString(source *time.Time) (string, error)` defines source type as a pointer(`*time.Time`) and return type as a raw value(`string`) .

### Helpers
You can define helper functions for more complex mappings.

```go
type todoMapperHelper struct {
}

var _ TodoMapperHelper = &todoMapperHelper{} // TodoMapperHelper interface is generated by sesame

func (h *todoMapperHelper) AtoB(source *model.TodoModel, dest *domain.Todo) error {
    if source.ValidateOnly {
        dest.Attributes["ValidateOnly"] = []string{"true"}
    }
    return nil
}

func (h *todoMapperHelper) BtoA(source *domain.Todo, dest *model.TodoModel) error {
    if _, ok := source.Attributes["ValidateOnly"]; ok {
        dest.ValidateOnly = true
    }
    return nil
}
```

and register it as `{MAPPER_NAME}Helper`:

```go
mappers.AddFactory("TodoMapperHelper", func(ms MapperGetter) (any, error) {
    return &todoMapperHelper{}, nil
})
```

Helpers will be called at the end of the generated mapping implementations.

## Donation
BTC: 1NEDSyUmo4SMTDP83JJQSWi1MvQUGGNMZB

## License
MIT

## Author
Yusuke Inuzuka
