package sesame

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"unicode"

	"dario.cat/mergo"
	"github.com/mitchellh/mapstructure"
	"gopkg.in/yaml.v3"

	"github.com/bmatcuk/doublestar/v4"
)

// LoadConfigFS read a config file from `path` in `fs`.
func LoadConfigFS(target any, path string, fs fs.FS) error {
	m, err := loadMap(path, fs)
	if err != nil {
		return err
	}

	sm := map[string]any{}
	for key, value := range m {
		sm[key.(string)] = value
	}

	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		DecodeHook: mapstructure.ComposeDecodeHookFunc(
			mapstructure.StringToTimeDurationHookFunc(),
		),
		WeaklyTypedInput: true,
		Result:           target,
	})
	if err != nil {
		return err
	}

	err = decoder.Decode(sm)
	if err != nil {
		return fmt.Errorf("Failed to map to a structure: %w", err)
	}
	v := reflect.ValueOf(&target)
	errs := walkConfig(v, "$")
	if len(errs) != 0 {
		return errors.Join(errs...)
	}
	return nil
}

type simpleFS struct {
}

func (s *simpleFS) Open(path string) (fs.File, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	return f, nil
}

// LoadConfig read a config file from `path` relative to the current directory.
func LoadConfig(target any, path string) error {
	return LoadConfigFS(target, path, &simpleFS{})
}

func loadMap(path string, fs fs.FS) (m map[any]any, err error) {
	f, err := fs.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() {
		err = f.Close()
	}()
	data, err := io.ReadAll(f)
	if err != nil {
		return nil, err
	}

	tm := map[any]any{}
	err = yaml.Unmarshal(data, &tm)
	if err != nil {
		return nil, fmt.Errorf("Failed to unmarshal a YAML file '%s': %w", path, err)
	}

	var files []string
	includes, ok := tm["_includes"]
	if ok {
		for _, include := range includes.([]any) {
			fullPath := include.(string)
			if !filepath.IsAbs(fullPath) {
				fullPath = filepath.Join(filepath.Dir(path), include.(string))
			}
			paths, err := doublestar.Glob(fs, fullPath)
			if err != nil {
				return nil, err
			}
			files = append(files, paths...)
		}
	}

	m = map[any]any{}
	for _, file := range files {
		file = os.Expand(file, envMapper)
		include, err := loadMap(file, fs)
		if err != nil {
			return nil, err
		}
		_ = mergo.Merge(&m, include, mergo.WithOverride, mergo.WithAppendSlice)
	}

	_ = mergo.Merge(&m, tm, mergo.WithOverride, mergo.WithAppendSlice)
	expandEnvVars(reflect.ValueOf(&m), path)
	return m, nil
}

var sourceFileKey = reflect.ValueOf("sourceFile")

func expandEnvVars(v reflect.Value, path string) {
	for v.Kind() == reflect.Ptr || v.Kind() == reflect.Interface {
		v = v.Elem()
	}
	switch v.Kind() {
	case reflect.Array, reflect.Slice:
		for i := 0; i < v.Len(); i++ {
			child := v.Index(i)
			if child.Elem().Kind() == reflect.String {
				s := os.Expand(child.Elem().String(), envMapper)
				child.Set(reflect.ValueOf(s))
			} else {
				expandEnvVars(v.Index(i), path)
			}
		}
	case reflect.Map:
		for _, k := range v.MapKeys() {
			child := v.MapIndex(k)
			if child.Elem().Kind() == reflect.String {
				s := os.Expand(child.Elem().String(), envMapper)
				v.SetMapIndex(k, reflect.ValueOf(s))
			} else {
				expandEnvVars(v.MapIndex(k), path)
			}
		}
		if !v.MapIndex(sourceFileKey).IsValid() {
			value := reflect.ValueOf(path)
			v.SetMapIndex(sourceFileKey, value)
		}
	default:
	}
}

func envMapper(placeholderName string) string {
	split := strings.SplitN(placeholderName, ":", 2)
	defValue := ""
	if len(split) == 2 {
		placeholderName = split[0]
		defValue = split[1]
	}

	val, ok := os.LookupEnv(placeholderName)
	if !ok {
		return defValue
	}

	return val
}

func walkConfig(v reflect.Value, path string) []error {
	var errs []error
	switch v.Kind() {
	case reflect.Ptr, reflect.Interface:
		errs = append(errs, walkConfig(v.Elem(), path)...)
	case reflect.Slice:
		if handler, ok := v.Interface().(interface {
			ConfigLoaded(string) []error
		}); ok {
			errs = append(errs, handler.ConfigLoaded(path)...)
		}
		prevSourceFile := ""
		offset := 0
		for i := 0; i < v.Len(); i++ {
			child := v.Index(i)
			if child.Kind() == reflect.Ptr {
				child = child.Elem()
			}
			if child.Kind() == reflect.Struct {
				if f := child.FieldByName("SourceFile"); f.IsValid() && f.String() != prevSourceFile {
					offset = i
				}
			}
			childPath := fmt.Sprintf("%s[%d]", path, i-offset)
			if handler, ok := v.Index(i).Interface().(interface {
				ConfigLoaded(string) []error
			}); ok {
				errs = append(errs, handler.ConfigLoaded(childPath)...)
			}
			errs = append(errs, walkConfig(v.Index(i), childPath)...)
		}
	case reflect.Struct:
		if handler, ok := v.Addr().Interface().(interface {
			ConfigLoaded(string) []error
		}); ok {
			errs = append(errs, handler.ConfigLoaded(path)...)
		}
		for i := 0; i < v.NumField(); i++ {
			t := v.Type().Field(i).Tag.Get("mapstructure")
			name := v.Type().Field(i).Name
			childPath := toConfigName(name)
			if len(path) != 0 {
				childPath = path + "." + toConfigName(name)
			}
			if strings.Contains(t, "squash") {
				childPath = path
			}
			errs = append(errs, walkConfig(v.Field(i), childPath)...)
		}
	default:
	}
	return errs
}

func toConfigName(v string) string {
	runes := []rune(v)
	if len(runes) > 1 && unicode.IsUpper(runes[0]) && unicode.IsLower(runes[1]) {
		ret := append([]rune{}, unicode.ToLower(runes[0]))
		return string(append(ret, runes[1:]...))
	}
	if len(runes) == 1 {
		return string(append([]rune{}, unicode.ToLower(runes[0])))
	}

	return v
}
