package kcheck

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"time"
)

const DefaultTagName = "chk"

const maxNestedDepth = 100

var ErrInvalidInput = errors.New("kcheck: invalid input")

type ValidatorFunc func(Field) error

type Field struct {
	Name      string
	Path      string
	Tag       string
	Param     string
	Value     any
	Kind      reflect.Kind
	IsNil     bool
	IsPointer bool
}

type Validator struct {
	mu    sync.RWMutex
	tag   string
	funcs map[string]ValidatorFunc
}

type mode int

const (
	modeSkip mode = iota
	modeSelect
)

type options struct {
	mode   mode
	fields map[string]struct{}
}

func New() *Validator {
	v := &Validator{
		tag:   DefaultTagName,
		funcs: make(map[string]ValidatorFunc),
	}

	v.RegisterDefaults()
	return v
}

func (v *Validator) Register(name string, fn ValidatorFunc) {
	name = strings.TrimSpace(name)
	if name == "" || fn == nil {
		return
	}

	v.mu.Lock()
	defer v.mu.Unlock()

	v.funcs[name] = fn
}

func (v *Validator) get(name string) (ValidatorFunc, bool) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	fn, ok := v.funcs[name]
	return fn, ok
}

func (v *Validator) Struct(input any) error {
	return v.structWithOptions(input, options{
		mode:   modeSkip,
		fields: map[string]struct{}{},
	})
}

func (v *Validator) StructSkip(input any, skips ...string) error {
	return v.structWithOptions(input, options{
		mode:   modeSkip,
		fields: toSet(skips),
	})
}

func (v *Validator) StructSelect(input any, selected ...string) error {
	return v.structWithOptions(input, options{
		mode:   modeSelect,
		fields: toSet(selected),
	})
}

func (v *Validator) structWithOptions(input any, opts options) error {
	if input == nil {
		return ErrInvalidInput
	}

	original := reflect.ValueOf(input)
	rv := indirectValue(original)

	if rv.Kind() != reflect.Struct {
		return ErrInvalidInput
	}

	var errs Errors
	active := make(map[visit]bool)
	if key, ok := visitFor(original, rv); ok {
		active[key] = true
	}
	v.validateStruct(rv, "", opts, &errs, active, 0)

	return errs.Err()
}

type visit struct {
	typ reflect.Type
	ptr uintptr
}

func visitFor(value, nested reflect.Value) (visit, bool) {
	var ptr uintptr
	tracked := false

	for value.IsValid() && (value.Kind() == reflect.Pointer || value.Kind() == reflect.Interface) {
		if value.IsNil() {
			return visit{}, false
		}
		if value.Kind() == reflect.Pointer {
			ptr = value.Pointer()
			tracked = true
		}
		value = value.Elem()
	}

	if !tracked || !nested.IsValid() {
		return visit{}, false
	}

	return visit{typ: nested.Type(), ptr: ptr}, true
}

func (v *Validator) validateStruct(
	rv reflect.Value,
	parentPath string,
	opts options,
	errs *Errors,
	active map[visit]bool,
	depth int,
) {
	rt := rv.Type()

	for i := 0; i < rv.NumField(); i++ {
		sf := rt.Field(i)
		fv := rv.Field(i)

		if sf.PkgPath != "" {
			continue
		}

		fieldName := sf.Name

		path := fieldName
		if parentPath != "" {
			path = parentPath + "." + fieldName
		}

		ignored := shouldIgnore(fieldName, path, opts)

		if ignored && !shouldDiveForSelectedPath(path, opts) {
			continue
		}

		tag := sf.Tag.Get(v.tag)
		if tag == "-" {
			continue
		}

		if shouldValidateNested(fv) {
			nested, _ := nestedStruct(fv)
			key, tracked := visitFor(fv, nested)
			if !tracked || !active[key] {
				if tracked {
					active[key] = true
				}
				v.validateNested(fv, path, opts, errs, active, depth+1)
				if tracked {
					delete(active, key)
				}
			}
		}

		if ignored {
			continue
		}

		if tag == "" {
			continue
		}

		field := buildField(path, fieldName, fv)

		for _, rule := range parseRules(tag) {
			field.Tag = rule.Name
			field.Param = rule.Param

			fn, ok := v.get(rule.Name)
			if !ok {
				errs.Add(path, fmt.Sprintf("validador [%s] no registrado", rule.Name))
				continue
			}

			if err := fn(field); err != nil {
				errs.Add(path, err.Error())
			}
		}
	}
}

func shouldValidateNested(v reflect.Value) bool {
	for v.IsValid() && (v.Kind() == reflect.Pointer || v.Kind() == reflect.Interface) {
		if v.IsNil() {
			return false
		}
		v = v.Elem()
	}

	if !v.IsValid() {
		return false
	}

	if _, ok := nestedStruct(v); ok {
		return true
	}

	switch v.Kind() {
	case reflect.Array, reflect.Slice, reflect.Map:
		return true
	default:
		return false
	}
}

func (v *Validator) validateNested(
	rv reflect.Value,
	path string,
	opts options,
	errs *Errors,
	active map[visit]bool,
	depth int,
) {
	if depth > maxNestedDepth {
		return
	}

	for rv.IsValid() && (rv.Kind() == reflect.Pointer || rv.Kind() == reflect.Interface) {
		if rv.IsNil() {
			return
		}
		rv = rv.Elem()
	}

	if !rv.IsValid() {
		return
	}

	switch rv.Kind() {
	case reflect.Struct:
		if rv.Type() == reflect.TypeFor[time.Time]() {
			return
		}
		v.validateStruct(rv, path, opts, errs, active, depth)
	case reflect.Array, reflect.Slice:
		for i := 0; i < rv.Len(); i++ {
			itemPath := fmt.Sprintf("%s[%d]", path, i)
			v.validateNested(rv.Index(i), itemPath, opts, errs, active, depth+1)
		}
	case reflect.Map:
		for _, key := range rv.MapKeys() {
			itemPath := fmt.Sprintf("%s[%v]", path, key.Interface())
			v.validateNested(rv.MapIndex(key), itemPath, opts, errs, active, depth+1)
		}
	}
}

func shouldIgnore(fieldName string, path string, opts options) bool {
	switch opts.mode {
	case modeSkip:
		return inSet(opts.fields, fieldName) || inSet(opts.fields, path)

	case modeSelect:
		return !inSet(opts.fields, fieldName) && !inSet(opts.fields, path)

	default:
		return false
	}
}

func shouldDiveForSelectedPath(path string, opts options) bool {
	if opts.mode != modeSelect {
		return false
	}

	prefix := path + "."
	collectionPrefix := path + "["

	for selected := range opts.fields {
		if strings.HasPrefix(selected, prefix) || strings.HasPrefix(selected, collectionPrefix) {
			return true
		}
	}

	return false
}

func toSet(values []string) map[string]struct{} {
	set := make(map[string]struct{}, len(values))

	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}

		set[value] = struct{}{}
	}

	return set
}

func inSet(set map[string]struct{}, value string) bool {
	_, ok := set[value]
	return ok
}

func nestedStruct(v reflect.Value) (reflect.Value, bool) {
	if !v.IsValid() {
		return reflect.Value{}, false
	}

	for v.Kind() == reflect.Pointer || v.Kind() == reflect.Interface {
		if v.IsNil() {
			return reflect.Value{}, false
		}
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return reflect.Value{}, false
	}

	if v.Type() == reflect.TypeFor[time.Time]() {
		return reflect.Value{}, false
	}

	return v, true
}

func indirectValue(v reflect.Value) reflect.Value {
	for v.IsValid() && (v.Kind() == reflect.Pointer || v.Kind() == reflect.Interface) && !v.IsNil() {
		v = v.Elem()
	}

	return v
}

func buildField(path string, name string, v reflect.Value) Field {
	field := Field{
		Name: name,
		Path: path,
	}

	if !v.IsValid() {
		field.IsNil = true
		return field
	}

	for v.IsValid() && (v.Kind() == reflect.Pointer || v.Kind() == reflect.Interface) {
		field.IsPointer = field.IsPointer || v.Kind() == reflect.Pointer
		if v.IsNil() {
			field.IsNil = true
			field.Kind = v.Kind()
			return field
		}
		v = v.Elem()
	}

	field.Kind = v.Kind()

	if v.CanInterface() {
		field.Value = v.Interface()
	}

	return field
}

type rule struct {
	Name  string
	Param string
}

func parseRules(tag string) []rule {
	tag = strings.TrimSpace(tag)
	if tag == "" {
		return nil
	}

	parts := strings.Fields(tag)
	rules := make([]rule, 0, len(parts))

	for _, part := range parts {
		name, param, hasParam := strings.Cut(part, "=")
		name = strings.TrimSpace(name)

		if name == "" {
			continue
		}

		r := rule{Name: name}

		if hasParam {
			r.Param = strings.TrimSpace(param)
		}

		rules = append(rules, r)
	}

	return rules
}

var defaultValidator = New()

func Register(name string, fn ValidatorFunc) {
	defaultValidator.Register(name, fn)
}

func Struct(input any) error {
	return defaultValidator.Struct(input)
}

func Valid(i any, skips ...string) error {
	return defaultValidator.StructSkip(i, skips...)
}

func ValidSelect(i any, selected ...string) error {
	return defaultValidator.StructSelect(i, selected...)
}
