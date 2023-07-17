package sesame

import (
	"fmt"
	"go/types"
	"os"
	"path/filepath"

	"golang.org/x/tools/go/packages"
)

// ParseStruct parses a given Go source code file to find a struct.
func ParseStruct(path string, name string, mctx *MappingContext) (types.Object, error) {
	pkg, err := ParseFile(path, mctx)
	if err != nil {
		return nil, err
	}
	obj := pkg.Scope().Lookup(name)
	if obj == nil {
		return nil, fmt.Errorf("Struct %s not found in %s", name, path)
	}
	_, ok := obj.Type().(*types.Named)
	if !ok {
		return nil, fmt.Errorf("%s in %s is not a struct", name, path)
	}
	_, ok = obj.Type().Underlying().(*types.Struct)
	if !ok {
		return nil, fmt.Errorf("%s in %s is not a struct", name, path)
	}
	return obj, nil
}

// ParseFile parses a given Go source code file.
func ParseFile(pkgPath string, mctx *MappingContext) (*types.Package, error) {
	oldCwd, _ := os.Getwd()
	pkgPath, _ = filepath.Abs(pkgPath)
	rootPath, err := findRootPath(pkgPath)
	if err != nil {
		return nil, err
	}
	_ = os.Chdir(rootPath)
	defer func() {
		_ = os.Chdir(oldCwd)
	}()

	absPkgPath, err := toAbsoluteImportPath(pkgPath)
	if err != nil {
		return nil, err
	}

	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedFiles | packages.NeedCompiledGoFiles |
			packages.NeedImports | packages.NeedDeps | packages.NeedExportFile |
			packages.NeedTypes | packages.NeedSyntax | packages.NeedTypesInfo |
			packages.NeedTypesSizes | packages.NeedModule | packages.NeedEmbedFiles |
			packages.NeedEmbedPatterns,
	}

	pkgs, err := packages.Load(cfg, absPkgPath)
	if err != nil {
		return nil, err
	}
	pkg := pkgs[0].Types

	mctx.AddImport(pkg.Path())
	for _, imp := range pkg.Imports() {
		mctx.AddImport(imp.Path())
	}
	return pkg, nil
}

func findRootPath(path string) (string, error) {
	start := filepath.Dir(path)
	for cur := start; cur != filepath.Dir(cur); cur = filepath.Dir(cur) {
		gomod := filepath.Join(cur, "go.mod")
		if _, err := os.Stat(gomod); err == nil {
			return cur, nil
		}
	}
	return "", fmt.Errorf("Can not resolve qualified package path: %s", path)
}
