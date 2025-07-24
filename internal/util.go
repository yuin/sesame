package internal

import (
	"bytes"
	"fmt"
	"go/types"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// GetMethod finds a *[types].Func by name.
// If a method not found, GetField returns false.
func GetMethod(nm *types.Named, name string, ignoreCase bool) (*types.Func, bool) {
	for i := 0; i < nm.NumMethods(); i++ {
		f := nm.Method(i)
		if (f.Name() == name) ||
			(ignoreCase && strings.EqualFold(f.Name(), name)) {
			return f, true
		}
	}
	return nil, false
}

// GetField finds a *[types].Var by name.
// If a field not found, GetField returns false.
func GetField(st *types.Struct, name string, ignoreCase bool) (*types.Var, bool) {
	parts := strings.SplitN(name, ".", 2)
	if len(parts) > 1 {
		for i := 0; i < st.NumFields(); i++ {
			f := st.Field(i)
			if (f.Name() == parts[0]) ||
				(ignoreCase && strings.EqualFold(f.Name(), parts[0])) {
				s, ok := GetStructType(f.Type())
				if !ok {
					return nil, false
				}
				return GetField(s, parts[1], ignoreCase)
			}
		}
	} else {
		for i := 0; i < st.NumFields(); i++ {
			f := st.Field(i)
			if (f.Name() == name) ||
				(ignoreCase && strings.EqualFold(f.Name(), name)) {
				return f, true
			}
		}
	}
	return nil, false
}

// GetStructType returns a struct type if an underlying type
// is a struct type.
func GetStructType(typ types.Type) (*types.Struct, bool) {
	switch t := typ.(type) {
	case *types.Pointer:
		return GetStructType(t.Elem())
	case *types.Named:
		return GetStructType(t.Obj().Type().Underlying())
	case *types.Struct:
		return t, true
	}
	return nil, false
}

// GetNamedType returns a named type if an underlying type
// is a struct type.
func GetNamedType(typ types.Type) (*types.Named, bool) {
	switch t := typ.(type) {
	case *types.Pointer:
		return GetNamedType(t.Elem())
	case *types.Named:
		return t, true
	}
	return nil, false
}

// GetSource returns a string representation with an alias package name.
func GetSource(typ types.Type, mctx *MappingContext) string {
	switch t := typ.(type) {
	case *types.Pointer:
		return "*" + GetSource(t.Elem(), mctx)
	case *types.Map:
		return "map[" + GetSource(t.Key(), mctx) + "]" + GetSource(t.Elem(), mctx)
	case *types.Slice:
		return "[]" + GetSource(t.Elem(), mctx)
	case *types.Array:
		return fmt.Sprintf("[%d]%s", t.Len(), GetSource(t.Elem(), mctx))
	case *types.Named:
		pkg := t.Obj().Pkg()
		if pkg == nil {
			return GetSource(t.Obj().Type().Underlying(), mctx)
		}
		if mctx.AbsolutePackagePath() != t.Obj().Pkg().Path() {
			alias := mctx.GetImportAlias(t.Obj().Pkg().Path())
			name := t.Obj().Name()
			typeArgs := t.TypeArgs()
			if typeArgs != nil {
				var tps []string
				for i := 0; i < typeArgs.Len(); i++ {
					tps = append(tps, GetSource(typeArgs.At(i), mctx))
				}
				name = t.Obj().Name() + "[" + strings.Join(tps, ",") + "]"
			}
			return alias + "." + name
		}
		return t.Obj().Name()
	case *types.Basic:
		return t.Name()
	default:
		return t.String()
	}
}

// CanCast returns true if sourceType can be (safely) casted into destType.
func CanCast(sourceType, destType types.Type) bool {
	// type MyType int
	//
	// sourceType: MyType, destType: int
	// or
	// ourceType: int, destType: MyType
	sourceTypeName := GetQualifiedTypeName(sourceType)
	sourceUnderlyingTypeName := GetQualifiedTypeName(sourceType.Underlying())
	destTypeName := GetQualifiedTypeName(destType)
	destUnderlyingTypeName := GetQualifiedTypeName(destType.Underlying())
	if sourceTypeName == destUnderlyingTypeName || destTypeName == sourceUnderlyingTypeName {
		return true
	}

	// A small size integer can be casted into a large size integer
	sourceBasicType, sok := sourceType.(*types.Basic)
	destBasicType, dok := destType.(*types.Basic)
	if !sok || !dok {
		return false
	}
	if strings.HasPrefix(sourceBasicType.Name(), "int") &&
		strings.HasPrefix(destBasicType.Name(), "int") {
		stype := to32(sourceBasicType.Name())
		sbit, _ := strconv.Atoi(stype[3:])
		dtype := to32(destBasicType.Name())
		dbit, _ := strconv.Atoi(dtype[3:])
		if dbit > sbit {
			return true
		}
	}
	if strings.HasPrefix(sourceBasicType.Name(), "uint") &&
		strings.HasPrefix(destBasicType.Name(), "uint") {
		stype := to32(sourceBasicType.Name())
		sbit, _ := strconv.Atoi(stype[4:])
		dtype := to32(destBasicType.Name())
		dbit, _ := strconv.Atoi(dtype[4:])
		if dbit < sbit {
			return true
		}
	}

	// TODO: other types(like string <=> []byte)?

	return false

}

func to32(t string) string {
	if t == "int" {
		return "int32"
	}
	if t == "uint" {
		return "uint32"
	}
	return t
}

// IsPointerPreferableType returns true if given type seems to be better for using as a
// pointer.
//
// - If a typ is a defined in the universe, interfaces, slices or maps, it returns false.
// - Otherwise(e.g. interface, structs), it returns true.
func IsPointerPreferableType(typ types.Type) bool {
	if ptyp, ok := typ.(*types.Pointer); ok {
		return IsPointerPreferableType(ptyp.Elem())
	}

	name := ""
	if v, ok := typ.Underlying().(interface {
		Name() string
	}); ok {
		name = v.Name()
	}
	if types.Universe.Lookup(name) != nil {
		return false
	}
	if _, ok := typ.(*types.Interface); ok {
		return false
	}
	if _, ok := typ.(*types.Array); ok {
		return false
	}
	return true
}

// IsNillableType returns true if given type can be nil.
func IsNillableType(typ types.Type) bool {
	if named, ok := typ.(*types.Named); ok {
		return IsNillableType(named.Obj().Type().Underlying())
	}
	if _, ok := typ.(*types.Interface); ok {
		return true
	}
	if _, ok := typ.(*types.Pointer); ok {
		return true
	}
	tn := typ.String()
	if strings.HasPrefix(tn, "[]") || strings.HasPrefix(tn, "map[") {
		return true
	}
	// array type can not be nil
	return false
}

// GetPreferableTypeSource returns
//   - If type is defined in the universe, a type without pointer
//   - If type is an interface, a type without pointer
//   - If type is a slice or a map, a type without pointer
//   - Otherwise, a type with pointer
func GetPreferableTypeSource(typ types.Type, mctx *MappingContext) string {
	if named, ok := typ.(*types.Named); ok {
		if _, iok := named.Obj().Type().Underlying().(*types.Interface); iok {
			return GetSource(named, mctx)
		}
	}

	if ptyp, ok := typ.(*types.Pointer); ok {
		if !IsPointerPreferableType(ptyp) {
			return GetSource(ptyp.Elem(), mctx)
		}
	} else if IsPointerPreferableType(typ) {
		tn := typ.String()
		if !(strings.HasPrefix(tn, "[]") || strings.HasPrefix(tn, "map[")) {
			return "*" + GetSource(typ, mctx)
		}
	}
	return GetSource(typ, mctx)
}

// GetNillableTypeSource returns a source code of a nillable type.
//
// - If type is an interface, a type without pointer
// - If type is a slice or a map, a type without pointer
// - Otherwise, a type with pointer.
func GetNillableTypeSource(typ types.Type, mctx *MappingContext) string {
	if named, ok := typ.(*types.Named); ok {
		if _, iok := named.Obj().Type().Underlying().(*types.Interface); iok {
			return GetSource(named, mctx)
		}
	}
	if ptyp, ok := typ.(*types.Pointer); ok {
		return "*" + GetSource(ptyp.Elem(), mctx)
	}
	tn := typ.String()
	if strings.HasPrefix(tn, "[]") || strings.HasPrefix(tn, "map[") {
		return GetSource(typ, mctx)
	}
	return "*" + GetSource(typ, mctx)
}

// GetValueTypeSource returns a source code of a type without a pointer.
func GetValueTypeSource(typ types.Type, mctx *MappingContext) string {
	if ptyp, ok := typ.(*types.Pointer); ok {
		return GetSource(ptyp.Elem(), mctx)
	}
	return GetSource(typ, mctx)
}

// GetStructPointerTypeSource returns a source code of a type with a pointer.
func GetStructPointerTypeSource(typ types.Type, mctx *MappingContext) string {
	if named, ok := typ.(*types.Named); ok {
		if _, iok := named.Obj().Type().Underlying().(*types.Interface); iok {
			return "*" + GetSource(named, mctx)
		}
	}

	if ptyp, ok := typ.(*types.Pointer); ok {
		return "*" + GetSource(ptyp.Elem(), mctx)
	}
	return "*" + GetSource(typ, mctx)
}

// GetQualifiedTypeName returns a qualified name of given type.
// Qualified name is a string joinning package and name with #.
func GetQualifiedTypeName(typ types.Type) string {
	if ptyp, ok := typ.(*types.Pointer); ok {
		return getQualifiedTypeName(ptyp.Elem())
	}
	return getQualifiedTypeName(typ)
}

func getQualifiedTypeName(typ types.Type) string {
	switch t := typ.(type) {
	case *types.Pointer:
		//return "*" + getQualifiedTypeName(t.Elem())
		return getQualifiedTypeName(t.Elem())
	case *types.Map:
		return "map[" + getQualifiedTypeName(t.Key()) + "]" + getQualifiedTypeName(t.Elem())
	case *types.Slice:
		return "[]" + getQualifiedTypeName(t.Elem())
	case *types.Array:
		return fmt.Sprintf("[%d]%s", t.Len(), getQualifiedTypeName(t.Elem()))
	case *types.Named:
		pkg := t.Obj().Pkg()
		if pkg == nil {
			return t.Obj().Name()
		}
		return t.Obj().Pkg().Path() + "#" + t.Obj().Name()
	case *types.Basic:
		return t.Name()
	default:
		return t.String()
	}
}

var modulePattern = regexp.MustCompile(`^\s*module\s*(.*)`)

func toAbsoluteImportPath(path string) (string, error) {
	if isModPackage(path) {
		parts := strings.Split(path, "/")
		i := len(parts) - 1
		for ; i >= 0; i-- {
			pkg := strings.Join(parts[:i+1], "/")
			cmd := exec.Command("go", "list", "-f", "'{{.Dir}}'", "-m", pkg)
			var out bytes.Buffer
			cmd.Stdout = &out
			if err := cmd.Run(); err != nil {
				continue
			}
			p := strings.Trim(strings.TrimSpace(out.String()), "'")
			if i != len(parts)-1 {
				p = filepath.Join(p, filepath.Join(parts[i+1:]...))
			}
			return p, nil
		}
		return "", fmt.Errorf("Can not resolve qualified package path: %s", path)
	}
	var buf []string
	start := path
	for cur := start; cur != filepath.Dir(cur); cur = filepath.Dir(cur) {
		gomod := filepath.Join(cur, "go.mod")
		if _, err := os.Stat(gomod); err == nil {
			b, err := os.ReadFile(gomod)
			if err != nil {
				return "", nil
			}
			p := modulePattern.FindAllStringSubmatch(string(b), 1)[0][1]
			buf = append([]string{p}, buf...)
			return strings.Join(buf, "/"), nil
		}
		buf = append([]string{filepath.Base(cur)}, buf...)
	}
	return "", fmt.Errorf("Can not resolve qualified package path: %s", path)
}

func isModPackage(pkg string) bool {
	if filepath.IsAbs(pkg) {
		return false
	}
	if strings.Contains(pkg, "./") || strings.Contains(pkg, "../") ||
		strings.Contains(pkg, ".\\") || strings.Contains(pkg, "..\\") {
		return false
	}

	parts := strings.Split(pkg, "/")
	if len(parts) < 2 {
		return false
	}
	return strings.Contains(parts[0], ".")
}

var methodPattern = regexp.MustCompile(`func \([^\)]*\)\.[\w]+\(([^\)]*)\)`)

// GetParamsCount returns a number of arguments of a function.
// TODO: f.Signature() was added in Go1.23. Use it instead of this function.
func GetParamsCount(f *types.Func) int {
	s := f.String()
	mm := methodPattern.FindStringSubmatch(s)
	if len(mm) == 0 {
		return 0
	}
	return strings.Count(mm[1], " ")
}
