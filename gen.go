package sesame

import (
	"fmt"
	"go/types"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"golang.org/x/tools/imports"
)

// OperandType indicates a target for functions.
type OperandType int

const (
	// OperandA means that an operand is 'A'.
	OperandA OperandType = 0

	// OperandB means that an operand is 'B'.
	OperandB OperandType = 1
)

// String implements [fmt].Stringer.
func (v OperandType) String() string {
	if v == OperandA {
		return "A"
	}
	return "B"
}

// Inverted returns an inverted [OperandType] .
func (v OperandType) Inverted() OperandType {
	return OperandType(v ^ 1)
}

// Generation is a definition of the mappings.
type Generation struct {
	// Mappers are a definition of the collection of mappers.
	Mappers *Mappers

	// Mappings is definitions of the mappings.
	Mappings []*Mapping

	// SourceFile is a source file path that contains this configuration.
	SourceFile string
}

// ConfigLoaded is an event handler will be executed when config is loaded.
func (g *Generation) ConfigLoaded(_ string) []error {
	var errs []error
	names := map[string]string{}
	for _, m := range g.Mappings {
		f, ok := names[m.Name]
		if ok {
			msg := fmt.Sprintf(", file: %s", f)
			if f != m.SourceFile {
				msg = fmt.Sprintf(", files: %s %s", f, m.SourceFile)
			}
			errs = append(errs, fmt.Errorf("mappings.name must be an unique(duplicated name: %s%s)", m.Name, msg))
		}
		names[m.Name] = m.SourceFile
	}
	return errs
}

// Mappers is a definition of the mappers.
type Mappers struct {
	// Package is a package of a mappers.
	Package string

	// Destination is a file path that this mappers will be written.
	Destination string

	// SourceFile is a source file path that contains this configuration.
	SourceFile string
}

// ConfigLoaded is an event handler will be executed when config is loaded.
func (m *Mappers) ConfigLoaded(path string) []error {
	var errs []error
	if len(m.Package) == 0 {
		errs = append(errs, fmt.Errorf("%s:\t%s.package must not be empty", m.SourceFile, path))
	}
	if len(m.Destination) == 0 {
		errs = append(errs, fmt.Errorf("%s:\t%s.destination must not be empty", m.SourceFile, path))
	}
	if !filepath.IsAbs(m.Destination) {
		m.Destination = filepath.Join(filepath.Dir(m.SourceFile), m.Destination)
	}
	return errs
}

// Mapping is a definition of the mapping.
type Mapping struct {
	// Name is a name of a mapper.
	Name string

	// Package is a package of a mapper.
	Package string

	// Destination is a file path that this mapper will be written.
	Destination string

	// AtoB is a name of a function.
	AtoB string `mapstructure:"a-to-b"`

	// AtoB is a name of a function.
	BtoA string `mapstructure:"b-to-a"`

	// Bidirectional means this mapping is a bi-directional mapping.
	Bidirectional bool

	// A is a mapping operand.
	A *MappingOperand

	// B is a mapping operand.
	B *MappingOperand

	// SourceFile is a source file path that contains this configuration.
	SourceFile string

	// ObjectMapping is a mapping definition for objects.
	ObjectMapping `mapstructure:",squash"`
}

// ConfigLoaded is an event handler will be executed when config is loaded.
func (m *Mapping) ConfigLoaded(path string) []error {
	var errs []error
	if len(m.Name) == 0 {
		errs = append(errs, fmt.Errorf("%s:\t%s.name must not be empty", m.SourceFile, path))
	}
	if len(m.Destination) == 0 {
		errs = append(errs, fmt.Errorf("%s:\t%s.destination must not be empty", m.SourceFile, path))
	}
	if !filepath.IsAbs(m.Destination) {
		m.Destination = filepath.Join(filepath.Dir(m.SourceFile), m.Destination)
	}
	if len(m.Package) == 0 {
		m.Package = filepath.Base(m.Destination)
	}
	if m.A == nil {
		errs = append(errs, fmt.Errorf("%s:\t%s.a must not be empty", m.SourceFile, path))
	}
	if m.B == nil {
		errs = append(errs, fmt.Errorf("%s:\t%s.b must not be empty", m.SourceFile, path))
	}
	return errs
}

// MethodName returns a name of a function that
// maps objects.
func (m *Mapping) MethodName(typ OperandType) string {
	if typ == OperandA {
		if len(m.AtoB) != 0 {
			return m.AtoB
		}
		return fmt.Sprintf("%sTo%s", m.A.Name, m.B.Name)
	}
	if len(m.BtoA) != 0 {
		return m.BtoA
	}
	return fmt.Sprintf("%sTo%s", m.B.Name, m.A.Name)
}

// PrivateName return a private-d name.
func (m *Mapping) PrivateName() string {
	return strings.ToLower(m.Name)
}

// ObjectMapping is a mapping definition for objects.
type ObjectMapping struct {
	// ExplicitOnly indicates that implicit mappings should not be
	// performed.
	ExplicitOnly bool

	// Fields is definitions of how fields will be mapped.
	Fields FieldMappings

	// Ignores is definitions of the fileds should be ignored.
	Ignores Ignores
}

// NewObjectMapping creates new [ObjectMapping] .
func NewObjectMapping() *ObjectMapping {
	return &ObjectMapping{}
}

// AddField adds new [FieldMapping] to this definition.
func (m *ObjectMapping) AddField(typ OperandType, v1, v2 string) {
	if typ == OperandA {
		m.Fields = append(m.Fields, &FieldMapping{
			A: v1,
			B: v2,
		})
	} else {
		m.Fields = append(m.Fields, &FieldMapping{
			B: v1,
			A: v2,
		})
	}
}

// MappingOperand is a mapping target.
type MappingOperand struct {
	// Package is a package path
	Package string

	// Name is a type name of the target.
	// This type must be defined in the File.
	Name string

	// SourceFile is a source file path that contains this configuration.
	SourceFile string
}

// ConfigLoaded is an event handler will be executed when config is loaded.
func (m *MappingOperand) ConfigLoaded(path string) []error {
	var errs []error
	if len(m.Package) == 0 {
		errs = append(errs, fmt.Errorf("%s:\t%s.package must not be empty", m.SourceFile, path))
	}
	if len(m.Name) == 0 {
		errs = append(errs, fmt.Errorf("%s:\t%s.name must not be empty", m.SourceFile, path))
	}

	if !filepath.IsAbs(m.Package) {
		m.Package = filepath.Join(filepath.Dir(m.SourceFile), m.Package)
	}

	return errs
}

// FieldMapping is definitions of how fields will be mapped.
type FieldMapping struct {
	// A is a name of the field defined in [Mapping].A.
	A string

	// B is a name of the field defined in [Mapping].B.
	B string

	// SourceFile is a source file path that contains this configuration.
	SourceFile string
}

// Value returns a value by [OperandType] .
func (m *FieldMapping) Value(typ OperandType) string {
	if typ == OperandA {
		return m.A
	}
	return m.B
}

// FieldMappings is a collection of [FieldMapping] s.
type FieldMappings []*FieldMapping

// Pair returns a paired value.
func (f FieldMappings) Pair(typ OperandType, value string) (string, bool) {
	for _, m := range f {
		if m.Value(typ) == value {
			return m.Value(typ.Inverted()), true
		}
	}
	return "", false
}

// ConfigLoaded is an event handler will be executed when config is loaded.
func (f FieldMappings) ConfigLoaded(path string) []error {
	var errs []error
	for i, v := range f {
		if len(v.A) == 0 {
			errs = append(errs, fmt.Errorf("%s:\t%s[%d].a must not be empty", v.SourceFile, path, i))
		}
		if len(v.B) == 0 {
			errs = append(errs, fmt.Errorf("%s:\t%s[%d].b must not be empty", v.SourceFile, path, i))
		}
	}
	return errs
}

// Ignores is a collection of fields should be ignored.
type Ignores []*FieldMapping

// Contains returns true if this collection contains a value.
func (f Ignores) Contains(typ OperandType, value string) bool {
	for _, m := range f {
		if m.Value(typ) == value {
			return true
		}
	}
	return false
}

// ConfigLoaded is an event handler will be executed when config is loaded.
func (f Ignores) ConfigLoaded(path string) []error {
	var errs []error
	for i, v := range f {
		if len(v.A) == 0 && len(v.B) == 0 || len(v.A) != 0 && len(v.B) != 0 {
			errs = append(errs, fmt.Errorf("%s:\t%s[%d] must define ether a or b", v.SourceFile, path, i))
		}
	}
	return errs
}

// Printer writes generated source codes.
// If dest already exists, Printer appends a new data
// to the end of it.
type Printer interface {
	io.Closer

	// P writes formatted-string and a newline.
	P(string, ...any)

	// WriteDoNotEdit writes a "DO NOT EDIT" header.
	WriteDoNotEdit()

	// AddVar adds a template variable name.
	AddVar(string)

	// ResolveVar resolves a variable value.
	ResolveVar(string, string)
}

type printer struct {
	path string
	f    *os.File
	buf  []string
	vars map[string]string
}

// NewPrinter creates a new [Printer] that writes a data to dest.
func NewPrinter(dest string) (Printer, error) {
	f, err := os.OpenFile(dest, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0755)
	if err != nil {
		return nil, err
	}
	return &printer{
		path: dest,
		f:    f,
		vars: map[string]string{},
	}, nil
}

func (p *printer) P(fm string, args ...any) {
	p.buf = append(p.buf, fmt.Sprintf(fm, args...))
}

func (p *printer) WriteDoNotEdit() {
	p.buf = append(p.buf, `// Code generated by sesame; DO NOT EDIT.`)
}

func (p *printer) AddVar(name string) {
	p.buf = append(p.buf, "{{"+name+"}}")
	p.vars[name] = ""
}

func (p *printer) ResolveVar(name, value string) {
	p.vars[name] = value
}

var goimportsOptions = &imports.Options{
	TabWidth:  8,
	TabIndent: true,
	Comments:  true,
	Fragment:  true,
}

var templatePattern = regexp.MustCompile(`{{(\w+)}}`)

func (p *printer) Close() error {
	data := templatePattern.ReplaceAllStringFunc(strings.Join(p.buf, "\n"), func(s string) string {
		vname := templatePattern.FindStringSubmatch(s)
		if len(vname) == 2 {
			if ret := p.vars[vname[1]]; ret != "" {
				return ret
			}
		}
		return ""
	})

	_, _ = p.f.WriteString(data)
	_, err := p.f.WriteString("\n")
	if err != nil {
		LogFunc(LogLevelError, err.Error())
	}

	err = p.f.Close()
	if err != nil {
		LogFunc(LogLevelError, err.Error())
		return err
	}
	src, err := os.ReadFile(p.path)
	if err != nil {
		LogFunc(LogLevelError, err.Error())
		return err
	}

	res, err := imports.Process(p.path, src, goimportsOptions)
	if err != nil {
		LogFunc(LogLevelError, err.Error())
		return err
	}
	err = os.WriteFile(p.path, res, 0755)
	if err != nil {
		LogFunc(LogLevelError, err.Error())
		return err
	}

	return nil
}

// Generator is an interface that generates mappers.
type Generator interface {
	Generate() error
}

type generator struct {
	config *Generation
}

// NewGenerator creates a new [Generator] .
func NewGenerator(config *Generation) Generator {
	return &generator{
		config: config,
	}
}

type mapperFunc struct {
	name        string
	mappersName string
	funcName    string
	pkg         string
}

func (g *generator) Generate() error {
	dests := map[string][]*Mapping{}

	for _, mapping := range g.config.Mappings {
		if _, ok := dests[mapping.Destination]; !ok {
			_ = os.Remove(mapping.Destination)
			dests[mapping.Destination] = []*Mapping{}
		}
		dests[mapping.Destination] = append(dests[mapping.Destination], mapping)
	}

	mappersAbsPkg, err := toAbsoluteImportPath(filepath.Dir(g.config.Mappers.Destination))
	if err != nil {
		return err
	}
	var mapperFuncs []*mapperFunc
	mappersContext := NewMappingContext(mappersAbsPkg)

	for dest, mappings := range dests {
		if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
			return err
		}
		LogFunc(LogLevelInfo, "Generate %s", dest)
		printer, err := NewPrinter(dest)
		if err != nil {
			return err
		}
		defer func() {
			_ = printer.Close()
		}()
		p := printer.P

		// Collect all imports
		pkg := ""
		absPkg, err := toAbsoluteImportPath(filepath.Dir(dest))
		if err != nil {
			return err
		}
		mctx := NewMappingContext(absPkg)
		lst := make([]struct {
			Mapping *Mapping
			A       types.Object
			B       types.Object
		}, len(mappings))
		i := 0
		for _, mapping := range mappings {
			LogFunc(LogLevelInfo, "Parse %s#%s", mapping.A.Package, mapping.A.Name)
			a, err := ParseStruct(mapping.A.Package, mapping.A.Name, mctx)
			if err != nil {
				return err
			}
			LogFunc(LogLevelInfo, "Parse %s#%s", mapping.B.Package, mapping.B.Name)
			b, err := ParseStruct(mapping.B.Package, mapping.B.Name, mctx)
			if err != nil {
				return err
			}
			if len(pkg) > 0 && pkg != mapping.Package {
				return fmt.Errorf("Destination %s have multiple package names", dest)
			}
			pkg = mapping.Package
			lst[i].Mapping = mapping
			lst[i].A = a
			lst[i].B = b
			i++
		}
		printer.WriteDoNotEdit()
		p(`package %s`, pkg)
		p(`import (`)
		printer.AddVar("IMPORTS")
		p(`)`)
		p("")
		p("")
		for _, elem := range lst {
			mapping := elem.Mapping
			LogFunc(LogLevelInfo, "Generate %s", mapping.Name)
			a := elem.A
			b := elem.B
			aArgSource := GetPreferableTypeSource(a.Type(), mctx)
			bArgSource := GetPreferableTypeSource(b.Type(), mctx)
			p("type %sHelper interface {", mapping.Name)
			p("  %s(%s, %s) error", mapping.MethodName(OperandA), aArgSource, bArgSource)
			if mapping.Bidirectional {
				p("  %s(%s, %s) error", mapping.MethodName(OperandB), bArgSource, aArgSource)
			}
			p("}")
			p("")
			p("type %s interface {", mapping.Name)
			p("%s(%s) (%s, error) ", mapping.MethodName(OperandA), aArgSource, bArgSource)
			if mapping.Bidirectional {
				p("%s(%s) (%s, error) ", mapping.MethodName(OperandB), bArgSource, aArgSource)
			}
			p("}")
			p("")
			p("var _ %s = &%s{}", mapping.Name, mapping.PrivateName())
			p("")
			p("func New%s(mapperGetter %s) %s {", mapping.Name, mapperGetterSrc, mapping.Name)
			p("  m := &%s{", mapping.PrivateName())
			p("    mapperGetter: mapperGetter,")
			p("  }")
			p("  helper, err := m.mapperGetter.Get(\"%sHelper\")", mapping.Name)
			p("  if err == nil {")
			p("    m.helper = helper.(%sHelper)", mapping.Name)
			p("  }")
			printer.AddVar("INIT_MAPPERS")
			p("  return m")
			p("}")
			p("")
			p("type %s struct {", mapping.PrivateName())
			p("mapperGetter %s", mapperGetterSrc)
			p("helper %sHelper", mapping.Name)
			printer.AddVar("MAPPERS")
			p("}")
			p("")

			LogFunc(LogLevelInfo, "Generate %s#%s", mapping.Name, mapping.MethodName(OperandA))
			if err := genMapFunc(printer, mapping, a, b, OperandA, mctx); err != nil {
				return err
			}

			p("")

			if mapping.Bidirectional {
				LogFunc(LogLevelInfo, "Generate %s#%s", mapping.Name, mapping.MethodName(OperandB))
				if err := genMapFunc(printer, mapping, b, a, OperandB, mctx); err != nil {
					return err
				}
			}

			absPkg, err := toAbsoluteImportPath(filepath.Dir(dest))
			if err != nil {
				return err
			}
			mappersContext.AddImport(absPkg)

			mapperFuncs = append(mapperFuncs, &mapperFunc{
				name:        mapping.Name,
				mappersName: mappersName(a.Type(), b.Type()),
				funcName:    mapping.MethodName(OperandA),
				pkg:         absPkg,
			})
			if mapping.Bidirectional {
				mapperFuncs = append(mapperFuncs, &mapperFunc{
					name:        mapping.Name,
					mappersName: mappersName(b.Type(), a.Type()),
					funcName:    mapping.MethodName(OperandB),
					pkg:         absPkg,
				})
			}

			LogFunc(LogLevelInfo, "Generate %s: Done", mapping.Name)
		}

		var mapperFieldNames []string
		var initMapperFields []string
		for _, mf := range mctx.MapperFuncFields() {
			mapperFieldNames = append(mapperFieldNames, fmt.Sprintf("%s %s", mf.FieldName, mf.Signature(mctx)))
			initMapperFields = append(initMapperFields,
				fmt.Sprintf(`if obj, err := mapperGetter.GetMapperFunc("%s", "%s"); err == nil {`,
					GetQualifiedTypeName(mf.Source), GetQualifiedTypeName(mf.Dest)),
				fmt.Sprintf(`  m.%s = obj.(%s)`, mf.FieldName, mf.Signature(mctx)),
				`}`)
		}
		var imps []string
		for impPath, impAlias := range mctx.Imports() {
			imps = append(imps, fmt.Sprintf("%s \"%s\"", impAlias, impPath))
		}

		printer.ResolveVar("MAPPERS", strings.Join(mapperFieldNames, "\n"))
		printer.ResolveVar("INIT_MAPPERS", strings.Join(initMapperFields, "\n"))
		printer.ResolveVar("IMPORTS", strings.Join(imps, "\n"))
		LogFunc(LogLevelInfo, "Generate %s: Done", dest)
	}

	LogFunc(LogLevelInfo, "Generate %s", g.config.Mappers.Destination)
	if err := genMappers(g.config.Mappers, mapperFuncs, mappersContext); err != nil {
		return err
	}
	LogFunc(LogLevelInfo, "Generate %s: Done", g.config.Mappers.Destination)

	return nil
}

func genMappers(mappers *Mappers, mapperFuncs []*mapperFunc, mctx *MappingContext) error {
	dest := mappers.Destination
	if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
		return err
	}
	_ = os.Remove(dest)
	printer, err := NewPrinter(dest)
	if err != nil {
		return err
	}
	defer func() {
		_ = printer.Close()
	}()
	p := printer.P

	printer.WriteDoNotEdit()
	p("")
	p(`package %s`, mappers.Package)
	p("")
	for _, line := range strings.Split(mappersSrc, "\n") {
		p(line)
	}

	var ms []string
	added := map[string]struct{}{}
	for _, m := range mapperFuncs {
		prefix := ""
		if alias := mctx.GetImportAlias(m.pkg); len(alias) > 0 {
			prefix = alias + "."
		}

		if _, ok := added[m.name]; !ok {
			ms = append(ms, fmt.Sprintf(`mappers.AddFactory("%s", func(ms MapperGetter) (any, error) {`, m.name))
			ms = append(ms, fmt.Sprintf("return %sNew%s(ms), nil", prefix, m.name))
			ms = append(ms, "})")
			added[m.name] = struct{}{}
		}
		ms = append(ms, fmt.Sprintf(`mappers.AddFactory("%s", func(ms MapperGetter) (any, error) {`, m.mappersName))
		ms = append(ms, fmt.Sprintf(`obj, err := ms.Get("%s")`, m.name))
		ms = append(ms, `if err != nil { return nil, err }`)
		ms = append(ms, fmt.Sprintf(`mapper, _ :=obj.(%s%s)`, prefix, m.name))
		ms = append(ms, fmt.Sprintf(`return mapper.%s, nil`, m.funcName))
		ms = append(ms, `})`)
	}
	printer.ResolveVar("MAPPERS", strings.Join(ms, "\n"))

	var imps []string
	for impPath, impAlias := range mctx.Imports() {
		imps = append(imps, fmt.Sprintf("%s \"%s\"", impAlias, impPath))
	}

	printer.ResolveVar("IMPORTS", strings.Join(imps, "\n"))

	return nil
}

func genMapFunc(printer Printer, mapping *Mapping,
	source types.Object, dest types.Object, typ OperandType, mctx *MappingContext) error {
	p := printer.P

	p("func (m *%s) %s(source *%s) (*%s, error) {",
		mapping.PrivateName(), mapping.MethodName(typ),
		GetSource(source.Type(), mctx), GetSource(dest.Type(), mctx))
	p("  dest := &%s{}", GetSource(dest.Type(), mctx))
	if err := genMapFuncBody(printer, source, "source", dest, "dest", &mapping.ObjectMapping, typ, mctx); err != nil {
		return err
	}
	p("  if m.helper != nil {")
	p("     if err := m.helper.%s(source, dest); err != nil {", mapping.MethodName(typ))
	p("       return nil, err")
	p("     }")
	p("  }")
	p("  return dest, nil")
	p("}")

	return nil
}

func genMapFuncBody(printer Printer,
	source types.Object, sourceNameBase string,
	dest types.Object, destNameBase string,
	mapping *ObjectMapping, typ OperandType, mctx *MappingContext) error {
	p := printer.P
	sourceStruct, ok := GetStructType(source.Type())
	if !ok {
		return fmt.Errorf("%s is not a struct", source.Type())
	}
	destStruct, ok := GetStructType(dest.Type())
	if !ok {
		return fmt.Errorf("%s is not a struct", dest.Type())
	}

	destName, ok := mapping.Fields.Pair(typ, "*")
	if ok { // embedded
		destField, _ := GetField(destStruct, destName)
		err := genFieldMapStmts(printer, sourceNameBase, destField.Type(), destNameBase+"."+destName, destField.Type(), mctx)
		if err != nil {
			return err
		}
	} else {
		for i := 0; i < sourceStruct.NumFields(); i++ {
			sourceField := sourceStruct.Field(i)
			if mapping.Ignores.Contains(typ, sourceField.Name()) {
				continue
			}
			found := false
			var destFieldType types.Type
			var destFieldNameBase string
			destName, ok := mapping.Fields.Pair(typ, sourceField.Name())
			if ok { // map explicitly
				if destName == "*" { // embedded
					found = true
					destFieldType = sourceField.Type()
					destFieldNameBase = destNameBase
				} else {
					parts := strings.SplitN(destName, ".", -1)
					destField, ok := GetField(destStruct, destName)
					if ok {
						found = true
						destFieldType = destField.Type()
						destFieldNameBase = destNameBase + "." + destName
					}
					if len(parts) > 1 {
						for i := 1; i < len(parts); i++ {
							nestName := strings.Join(parts[:i], ".")
							nestField, ok := GetField(destStruct, nestName)
							if ok {
								p("if %s.%s == nil {", destNameBase, nestName)
								p("  %s.%s = %s{}", destNameBase, nestName, strings.Replace(GetSource(nestField.Type(), mctx), "*", "&", 1))
								p("}")
							}
						}
					}
				}
			} else if !mapping.ExplicitOnly { // map implicitly
				destField, ok := GetField(destStruct, sourceField.Name())
				if ok {
					found = true
					destFieldType = destField.Type()
					destFieldNameBase = destNameBase + "." + destField.Name()
				}
			}

			if !found {
				LogFunc(LogLevelDebug, "%s.%s.%s is ignored", source.Pkg().Name(), source.Name(), sourceField.Name())
				continue
			}

			err := genFieldMapStmts(printer, sourceNameBase+"."+sourceField.Name(),
				sourceField.Type(), destFieldNameBase, destFieldType, mctx)
			if err != nil {
				return err
			}
		}

		for _, fm := range mapping.Fields {
			sourceFieldName := fm.Value(typ)
			destFieldName := fm.Value(typ.Inverted())

			parts := strings.SplitN(sourceFieldName, ".", 2)
			if len(parts) > 1 {
				f, ok := GetField(sourceStruct, parts[0])
				if !ok {
					continue
				}

				nestMapping := NewObjectMapping()
				nestMapping.ExplicitOnly = true
				nestMapping.AddField(typ, parts[1], destFieldName)
				err := genMapFuncBody(printer, f, sourceNameBase+"."+parts[0],
					dest, destNameBase, nestMapping, typ, mctx)
				if err != nil {
					return err
				}
			}
		}

	}
	return nil
}

func genFieldMapStmts(printer Printer,
	sourceName string, sourceType types.Type,
	destName string, destType types.Type,
	mctx *MappingContext) error {
	p := printer.P
	mctx.AddMapperFuncField(sourceType, destType)
	switch typ := sourceType.(type) {
	case *types.Array:
		dtype, ok := destType.(*types.Array)
		if !ok {
			return fmt.Errorf("type mismatch: %s and %s should be an array", sourceName, destName)
		}
		// TODO: test slice and array and array size
		// TODO: support a conversion slice and array?

		i := mctx.NextVarCount()
		p("for i%d, elm := range %s {", i, sourceName)
		n := mctx.NextVarCount()
		p("\t\tvar tmp%d %s", n, GetSource(dtype.Elem(), mctx))
		if err := genFieldMapStmts(printer, "elm", typ.Elem(), fmt.Sprintf("tmp%d", n), dtype.Elem(), mctx); err != nil {
			return err
		}
		genAssignStmt(printer, fmt.Sprintf("tmp%d", n), typ.Elem(), fmt.Sprintf("%s[i%d]", destName, i), dtype.Elem(), mctx)
		p("}")
	case *types.Slice:
		dtype, ok := destType.(*types.Slice)
		if !ok {
			return fmt.Errorf("type mismatch: %s and %s should be a slice", sourceName, destName)
		}
		// TODO: test slice and array and array size
		// TODO: support a conversion slice and array?

		p("for _, elm := range %s {", sourceName)
		n := mctx.NextVarCount()
		p("var tmp%d %s", n, GetSource(dtype.Elem(), mctx))
		if err := genFieldMapStmts(printer, "elm", typ.Elem(), fmt.Sprintf("tmp%d", n), dtype.Elem(), mctx); err != nil {
			return err
		}
		p("%s = append(%s, tmp%d)", destName, destName, n)
		p("}")
	case *types.Map:
		// TODO: support a conversion map and struct?
		dtype, ok := destType.(*types.Map)
		if !ok {
			return fmt.Errorf("type mismatch: %s and %s should be a map", sourceType, destType)
		}

		p("%s = make(%s)", destName, GetSource(destType, mctx))
		p("for key, elm := range %s {", sourceName)
		n := mctx.NextVarCount()
		p("var tmp%d %s", n, GetSource(dtype.Elem(), mctx))
		if err := genFieldMapStmts(printer, "elm", typ.Elem(), fmt.Sprintf("tmp%d", n), dtype.Elem(), mctx); err != nil {
			return err
		}
		genAssignStmt(printer, fmt.Sprintf("tmp%d", n), typ.Elem(), fmt.Sprintf("%s[key]", destName), dtype.Elem(), mctx)
		p("}")
	case *types.Chan:
		LogFunc(LogLevelInfo, "chan type %s ignored", sourceName)
	default:
		genAssignStmt(printer, sourceName, sourceType, destName, destType, mctx)
	}
	return nil
}

func genAssignStmt(printer Printer,
	sourceName string, sourceType types.Type,
	destName string, destType types.Type,
	mctx *MappingContext) {
	p := printer.P

	sourceTypeName := GetQualifiedTypeName(sourceType)
	_, sourceIsPointer := sourceType.(*types.Pointer)
	sourceIsPointerPreferable := IsPointerPreferableType(sourceType)
	destTypeName := GetQualifiedTypeName(destType)
	_, destIsPointer := destType.(*types.Pointer)
	destIsPointerPreferable := IsPointerPreferableType(destType)

	// Try to execute custom mapper
	argName := ""
	switch {
	case sourceIsPointerPreferable && sourceIsPointer:
		argName = sourceName
	case sourceIsPointerPreferable && !sourceIsPointer:
		argName = "&" + sourceName
	case !sourceIsPointerPreferable && sourceIsPointer:
		argName = "*" + sourceName
	case !sourceIsPointerPreferable && !sourceIsPointer:
		argName = sourceName
	}

	mf := mctx.GetMapperFuncFieldName(sourceType, destType)
	if mf != nil {
		p("if m.%s != nil {", mf.FieldName)
		p("  if v, err := m.%s(%s); err != nil {", mf.FieldName, argName)
		p("    return nil, err")
		p("  } else {")
		switch {
		case destIsPointer && destIsPointerPreferable:
			p("%s = v", destName)
		case destIsPointer && !destIsPointerPreferable:
			p("%s = &v", destName)
		case !destIsPointer && destIsPointerPreferable:
			p("%s = *v", destName)
		case !destIsPointer && !destIsPointerPreferable:
			p("%s = v", destName)
		}
		p("  }")
		p("}")
	}

	if sourceTypeName == destTypeName {
		switch {
		case sourceIsPointer && destIsPointer:
			p("%s = %s", destName, sourceName)
		case sourceIsPointer && !destIsPointer:
			p("%s = *%s", destName, sourceName)
		case !sourceIsPointer && destIsPointer:
			p("%s = &%s", destName, sourceName)
		case !sourceIsPointer && !destIsPointer:
			p("%s = %s", destName, sourceName)

		}
		return
	}

	if CanCast(sourceType, destType) {
		genAssignStmt(printer, fmt.Sprintf("%s(%s)", GetSource(destType, mctx), sourceName),
			destType, destName, destType, mctx)
		return
	}

}
