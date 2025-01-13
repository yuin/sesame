package sesame

import (
	"fmt"
	"go/types"
	"regexp"
)

// MappingContext is an interface that contains contextual data for
// the generation.
type MappingContext struct {
	absPkgPath          string
	aliasCount          int
	aliasBase           string
	imports             map[string]string
	importHashes        map[string]int
	varCount            int
	mapperFuncFields    []*MapperFuncField
	mapperFuncCount     int
	converterFuncFields []*ConverterFuncField
	converterFuncCount  int
}

// MapperFuncField is a mapper function field.
type MapperFuncField struct {
	// FieldName is a name of the field.
	FieldName string

	// MapperFuncName is a name of the mapper function.
	MapperFuncName string

	// Source is a source type of the function.
	Source types.Type

	// Dest is a source type of the function.
	Dest types.Type
}

// Signature returns a function signature.
func (m *MapperFuncField) Signature(mctx *MappingContext) string {
	return fmt.Sprintf("func(%s.Context, %s, %s) error",
		mctx.GetImportAlias("context"),
		GetPreferableTypeSource(m.Source, mctx),
		GetPointerTypeSource(m.Dest, mctx))
}

// ConverterFuncField is a mapper function field.
type ConverterFuncField struct {
	// FieldName is a name of the field.
	FieldName string

	// ConverterFuncName is a name of the mapper function.
	ConverterFuncName string

	// Source is a source type of the function.
	Source types.Type

	// Dest is a source type of the function.
	Dest types.Type
}

// Signature returns a function signature.
func (c *ConverterFuncField) Signature(mctx *MappingContext) string {
	destIsPointerPreferable := IsPointerPreferableType(c.Dest)

	if destIsPointerPreferable {
		return fmt.Sprintf("func(%s.Context, %s) (%s, error)",
			mctx.GetImportAlias("context"),
			GetNillableTypeSource(c.Source, mctx),
			GetPreferableTypeSource(c.Dest, mctx))
	}
	return fmt.Sprintf("func(%s.Context, %s) (%s, bool, error)",
		mctx.GetImportAlias("context"),
		GetNillableTypeSource(c.Source, mctx),
		GetPreferableTypeSource(c.Dest, mctx))
}

// NewMappingContext returns new [MappingContext] .
func NewMappingContext(absPkgPath string) *MappingContext {
	mctx := &MappingContext{
		absPkgPath:          absPkgPath,
		aliasCount:          0,
		aliasBase:           "pkg",
		imports:             map[string]string{},
		importHashes:        map[string]int{},
		mapperFuncFields:    []*MapperFuncField{},
		mapperFuncCount:     0,
		converterFuncFields: []*ConverterFuncField{},
		converterFuncCount:  0,
	}
	mctx.AddImport("context")
	return mctx
}

// AbsolutePackagePath returns na absolute package path of a file will be
// generated this mapping.
func (c *MappingContext) AbsolutePackagePath() string {
	return c.absPkgPath
}

var pkgr = regexp.MustCompile(`[^a-zA-Z0-9]`)

// AddImport adds import path and generate new alias name for it.
func (c *MappingContext) AddImport(path string) {
	if path == c.AbsolutePackagePath() {
		return
	}
	if _, ok := c.imports[path]; !ok {
		h := pkgr.ReplaceAllString(path, "_")
		if _, ok := c.importHashes[path]; !ok {
			c.importHashes[path] = 0
		} else {
			c.importHashes[path]++
		}
		hv := c.importHashes[path]
		if hv == 0 {
			c.imports[path] = fmt.Sprintf("%s_%s", c.aliasBase, h)
		} else {
			c.imports[path] = fmt.Sprintf("%s_%s_%05d", c.aliasBase, h, c.importHashes[path])
		}
		c.aliasCount++
	}
}

// GetImportAlias returns an alias for the given import path.
func (c *MappingContext) GetImportAlias(path string) string {
	c.AddImport(path)
	v, ok := c.imports[path]
	if !ok {
		return ""
	}
	return v
}

// GetImportPath returns a fully qualified path for the given import alias.
// If alias is not found, GetImportPath returns given alias.
func (c *MappingContext) GetImportPath(alias string) string {
	if alias == "" {
		return c.AbsolutePackagePath()
	}
	for key, value := range c.imports {
		if value == alias {
			return key
		}
	}
	return alias
}

// Imports returns a map of the all imports.
// Result map key is an import path and value is an alias.
func (c *MappingContext) Imports() map[string]string {
	return c.imports
}

// NextVarCount returns a var count and increments it.
func (c *MappingContext) NextVarCount() int {
	v := c.varCount
	c.varCount++
	return v
}

// AddMapperFuncField adds a mapper function and generates a field name for it.
func (c *MappingContext) AddMapperFuncField(sourceType types.Type, destType types.Type) {
	sname := GetQualifiedTypeName(sourceType)
	dname := GetQualifiedTypeName(destType)
	if sname == dname {
		return
	}
	mapperFuncName := mappersName(sourceType, destType)
	for _, m := range c.mapperFuncFields {
		if m.MapperFuncName == mapperFuncName {
			return
		}
	}
	fieldName := fmt.Sprintf("mapper%05d", c.mapperFuncCount)
	c.mapperFuncCount++
	c.mapperFuncFields = append(c.mapperFuncFields, &MapperFuncField{
		FieldName:      fieldName,
		MapperFuncName: mapperFuncName,
		Source:         sourceType,
		Dest:           destType,
	})
}

// GetMapperFuncFieldName returns a mapper function field name.
func (c *MappingContext) GetMapperFuncFieldName(sourceType types.Type, destType types.Type) *MapperFuncField {
	mapperFuncName := mappersName(sourceType, destType)
	for _, m := range c.mapperFuncFields {
		if m.MapperFuncName == mapperFuncName {
			return m
		}
	}
	return nil
}

// MapperFuncFields returns a list of [MapperFuncField] .
func (c *MappingContext) MapperFuncFields() []*MapperFuncField {
	return c.mapperFuncFields
}

// AddConverterFuncField adds a converter function and generates a field name for it.
func (c *MappingContext) AddConverterFuncField(sourceType types.Type, destType types.Type) {
	sname := GetQualifiedTypeName(sourceType)
	dname := GetQualifiedTypeName(destType)
	if sname == dname {
		return
	}
	converterFuncName := convertersName(sourceType, destType)
	for _, m := range c.converterFuncFields {
		if m.ConverterFuncName == converterFuncName {
			return
		}
	}
	fieldName := fmt.Sprintf("converter%05d", c.converterFuncCount)
	c.converterFuncCount++
	c.converterFuncFields = append(c.converterFuncFields, &ConverterFuncField{
		FieldName:         fieldName,
		ConverterFuncName: converterFuncName,
		Source:            sourceType,
		Dest:              destType,
	})
}

// GetConverterFuncFieldName returns a converter function field name.
func (c *MappingContext) GetConverterFuncFieldName(sourceType types.Type, destType types.Type) *ConverterFuncField {
	converterFuncName := convertersName(sourceType, destType)
	for _, m := range c.converterFuncFields {
		if m.ConverterFuncName == converterFuncName {
			return m
		}
	}
	return nil
}

// ConverterFuncFields returns a list of [ConverterFuncField] .
func (c *MappingContext) ConverterFuncFields() []*ConverterFuncField {
	return c.converterFuncFields
}
