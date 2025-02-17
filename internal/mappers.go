package internal

const mapperGetterSrc = `interface {
	Get(id string) (any, error)
	GetAllMappers() (map[string]any, error)
	GetFunc(id string, sourceType reflect.Type, destType reflect.Type) (any, error)
	GetFuncByTypeName(id string, sourceName string, destName string) (any, error)
}`

const mappersSrc = `
import (
	{{IMPORTS}}
	"github.com/yuin/sesame"
)

// NewMappers return a new [sesame.Mappers] .
// Mappers are goroutine safe for get actions.
// Mappers are not goroutine safe for add/merge actions.
func NewMappers() sesame.Mappers {
	mappers := 	sesame.NewMappers()
	{{MAPPERS}}
	return mappers
}
`
