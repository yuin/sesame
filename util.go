package sesame

import "reflect"

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
		typ = typ.Elem()
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
