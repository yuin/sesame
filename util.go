package sesame

import (
	"fmt"
	"reflect"
)

func getType[T any]() reflect.Type {
	var v T
	rt := reflect.TypeOf(&v)
	if rt.Elem().Kind() == reflect.Interface || rt.Elem().Kind() == reflect.Ptr {
		rt = rt.Elem()
	}
	return rt
}

func toTypeName(typ reflect.Type) string {
	if typ.Kind() == reflect.Ptr {
		return toTypeName(typ.Elem())
	}

	if typ.Kind() == reflect.Slice {
		return "[]" + toTypeName(typ.Elem())
	}

	if typ.Kind() == reflect.Array {
		return fmt.Sprintf("[%d]%s", typ.Len(), toTypeName(typ.Elem()))
	}

	if typ.Kind() == reflect.Map {
		return fmt.Sprintf("map[%s]%s", toTypeName(typ.Key()), toTypeName(typ.Elem()))
	}

	name := typ.Name()
	if len(typ.PkgPath()) != 0 {
		name = typ.PkgPath() + "#" + name
	}
	if len(name) == 0 {
		name = typ.String()
	}
	return name
}

func toTypeIndex(prefix string, typ1, typ2 reflect.Type) string {
	return toTypeIndexFromString(prefix, toTypeName(typ1), toTypeName(typ2))
}

func toTypeIndexFromString(prefix string, typ1, typ2 string) string {
	return prefix + typ1 + ":" + typ2
}
