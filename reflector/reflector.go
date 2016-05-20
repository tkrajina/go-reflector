package reflector

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

type fieldListingType int

const (
	fieldsAll                fieldListingType = iota
	fieldsFlattenAnonymous                    = iota
	fieldsNoFlattenAnonymous                  = iota
)

type Obj struct {
	iface interface{}

	isStruct      bool
	isPtrToStruct bool

	// If ptr to struct, this field will contain the type of that struct
	underlyingType reflect.Type

	objType reflect.Type
	objKind reflect.Kind
}

func NewFromType(ty reflect.Type) *Obj {
	return New(reflect.New(ty).Interface())
}

func New(obj interface{}) *Obj {
	o := &Obj{iface: obj}
	o.objType = reflect.TypeOf(obj)
	o.objKind = o.objType.Kind()

	ty := o.Type()
	if ty.Kind() == reflect.Struct {
		o.isStruct = true
	}
	if ty.Kind() == reflect.Ptr && ty.Elem().Kind() == reflect.Struct {
		ty = ty.Elem()
		o.isPtrToStruct = true
	}
	o.underlyingType = ty
	return o
}

// Fields returns fields. Don't list fields inside Anonymous fields as distinct fields
func (o *Obj) Fields() []ObjField {
	return o.fields(reflect.TypeOf(o.iface), fieldsNoFlattenAnonymous)
}

// FieldsFlattened returns fields. Will not list Anonymous fields but it will list fields declared in those anonymous fields
func (o Obj) FieldsFlattened() []ObjField {
	return o.fields(reflect.TypeOf(o.iface), fieldsFlattenAnonymous)
}

// FieldsFlattened returns fields. List both anonymous fields and fields declared inside anonymous fields.
func (o Obj) FieldsAll() []ObjField {
	return o.fields(reflect.TypeOf(o.iface), fieldsAll)
}

// FindDoubleFields checks if this object has declared multiple fields with a same name (by checking recursively Anonymous
// fields and their fields)
func (o Obj) FindDoubleFields() []string {
	fields := map[string]int{}
	res := []string{}
	for _, f := range o.FieldsAll() {
		counter := 0
		if counter := fields[f.name]; counter == 1 {
			res = append(res, f.name)
		}
		fields[f.name] = counter + 1
	}
	return res
}

func (o *Obj) fields(ty reflect.Type, listingType fieldListingType) []ObjField {
	fields := make([]ObjField, 0)

	if ty.Kind() == reflect.Ptr {
		ty = ty.Elem()
	}

	if ty.Kind() != reflect.Struct {
		return fields // No need to populate nonstructs
	}

	for i := 0; i < ty.NumField(); i++ {
		field := ty.Field(i)

		k := field.Type.Kind()
		if isExportable(field) {
			if listingType == fieldsAll {
				fields = append(fields, *newObjField(o, field.Name))
				if k == reflect.Struct && field.Anonymous {
					fields = append(fields, o.fields(field.Type, listingType)...)
				}
			} else {
				if listingType == fieldsFlattenAnonymous && k == reflect.Struct && field.Anonymous {
					fields = append(fields, o.fields(field.Type, listingType)...)
				} else {
					fields = append(fields, *newObjField(o, field.Name))
				}
			}
		}
	}

	return fields
}

func (o Obj) IsPtr() bool {
	return o.objKind == reflect.Ptr
}

func (o Obj) IsStructOrPtrToStruct() bool {
	return o.isStruct || o.isPtrToStruct
}

func (o *Obj) Field(name string) *ObjField {
	return newObjField(o, name)
}

func (o Obj) Type() reflect.Type {
	return o.objType
}

func (o Obj) Kind() reflect.Kind {
	return o.objKind
}

func (o *Obj) Method(name string) *ObjMethod {
	return newObjMethod(o, name)
}

func (o *Obj) Methods() []ObjMethod {
	res := []ObjMethod{}
	ty := o.Type()
	for i := 0; i < ty.NumMethod(); i++ {
		method := ty.Method(i)
		res = append(res, *newObjMethod(o, method.Name))
	}
	return res
}

type ObjField struct {
	obj  *Obj
	name string

	valueField  reflect.Value
	structField reflect.StructField

	fieldKind reflect.Kind
	fieldType reflect.Type

	valid bool
}

func newObjField(obj *Obj, name string) *ObjField {
	res := &ObjField{
		obj:  obj,
		name: name,
	}

	res.valid = false
	res.fieldKind = reflect.Invalid
	if res.obj.IsStructOrPtrToStruct() {
		var found bool
		var valueField reflect.Value
		var structField reflect.StructField
		if res.obj.isPtrToStruct {
			valueField = reflect.ValueOf(res.obj.iface).Elem().FieldByName(res.name)
			structField, found = reflect.TypeOf(res.obj.iface).Elem().FieldByName(res.name)
		} else {
			valueField = reflect.ValueOf(res.obj.iface).FieldByName(res.name)
			structField, found = reflect.TypeOf(res.obj.iface).FieldByName(res.name)
		}
		res.valueField = valueField
		res.structField = structField
		res.valid = found && valueField.IsValid()
		if res.valid {
			res.fieldType = structField.Type
			res.fieldKind = structField.Type.Kind()
		} else {
			res.fieldKind = reflect.Invalid
		}
	}

	return res
}

func (of *ObjField) assertValid() error {
	if !of.valid {
		return fmt.Errorf("Invalid field %s", of.name)
	}
	return nil
}

func (of *ObjField) IsValid() bool {
	return of.valid
}

func (of *ObjField) Name() string {
	return of.name
}

func (of *ObjField) Kind() reflect.Kind {
	return of.fieldKind
}

func (of *ObjField) Type() reflect.Type {
	return of.fieldType
}

func (of *ObjField) Tag(tag string) (string, error) {
	if err := of.assertValid(); err != nil {
		return "", err
	}
	return of.structField.Tag.Get(tag), nil
}

func (of *ObjField) Tags() (map[string]string, error) {
	if err := of.assertValid(); err != nil {
		return nil, err
	}

	res := map[string]string{}
	tag := of.structField.Tag

	// This code is copied/modified from: reflect/type.go:
	for tag != "" {
		// Skip leading space.
		i := 0
		for i < len(tag) && tag[i] == ' ' {
			i++
		}
		tag = tag[i:]
		if tag == "" {
			break
		}

		// Scan to colon. A space, a quote or a control character is a syntax error.
		// Strictly speaking, control chars include the range [0x7f, 0x9f], not just
		// [0x00, 0x1f], but in practice, we ignore the multi-byte control characters
		// as it is simpler to inspect the tag's bytes than the tag's runes.
		i = 0
		for i < len(tag) && tag[i] > ' ' && tag[i] != ':' && tag[i] != '"' && tag[i] != 0x7f {
			i++
		}
		if i == 0 || i+1 >= len(tag) || tag[i] != ':' || tag[i+1] != '"' {
			break
		}
		name := string(tag[:i])
		tag = tag[i+1:]

		// Scan quoted string to find value.
		i = 1
		for i < len(tag) && tag[i] != '"' {
			if tag[i] == '\\' {
				i++
			}
			i++
		}
		if i >= len(tag) {
			break
		}
		qvalue := string(tag[:i+1])
		tag = tag[i+1:]

		value, err := strconv.Unquote(qvalue)
		if err != nil {
			return nil, fmt.Errorf("Cannot unquote tag %s in %T.%s: %s", name, of.obj.iface, of.name, err.Error())
		}
		res[name] = value
		/*
			if key == name {
				value, err := strconv.Unquote(qvalue)
				if err != nil {
					break
				}
				return value
			}
		*/
	}

	return res, nil
}

// TagExpanded returns the tag value "expanded" with commas
func (of *ObjField) TagExpanded(tag string) ([]string, error) {
	if err := of.assertValid(); err != nil {
		return nil, err
	}
	return strings.Split(of.structField.Tag.Get(tag), ","), nil
}

func (of *ObjField) Valid() bool {
	return of.valid
}

func (of *ObjField) Anonymous() bool {
	if err := of.assertValid(); err != nil {
		return false
	}
	field, found := of.obj.underlyingType.FieldByName(of.name)
	if !found {
		return false
	}
	return field.Anonymous
}

func (of *ObjField) Set(value interface{}) error {
	if err := of.assertValid(); err != nil {
		return err
	}

	if !of.valueField.CanSet() {
		return fmt.Errorf("Field %s in %T not settable", of.name, of.obj.iface)
	}

	of.valueField.Set(reflect.ValueOf(value))

	return nil
}

func (of *ObjField) Get() (interface{}, error) {
	if err := of.assertValid(); err != nil {
		return nil, err
	}

	return of.valueField.Interface(), nil
}

type ObjMethod struct {
	obj    *Obj
	name   string
	method reflect.Value
}

func newObjMethod(obj *Obj, name string) *ObjMethod {
	return &ObjMethod{
		obj:    obj,
		name:   name,
		method: reflect.ValueOf(obj.iface).MethodByName(name),
	}
}

func (om *ObjMethod) InTypes() []reflect.Type {
	method := reflect.ValueOf(om.obj.iface).MethodByName(om.name)
	if !method.IsValid() {
		return []reflect.Type{}
	}
	ty := method.Type()
	out := make([]reflect.Type, ty.NumIn())
	for i := 0; i < ty.NumIn(); i++ {
		out[i] = ty.In(i)
	}
	return out
}

func (om *ObjMethod) OutTypes() []reflect.Type {
	method := reflect.ValueOf(om.obj.iface).MethodByName(om.name)
	if !method.IsValid() {
		return []reflect.Type{}
	}
	ty := method.Type()
	out := make([]reflect.Type, ty.NumOut())
	for i := 0; i < ty.NumOut(); i++ {
		out[i] = ty.Out(i)
	}
	return out
}

func (om *ObjMethod) IsValid() bool {
	return om.method.IsValid()
}

// Call calls this method. Note that in the error returning value is not the error from the method call
func (om *ObjMethod) Call(args ...interface{}) (*CallResult, error) {
	if !om.method.IsValid() {
		return nil, fmt.Errorf("Invalid method %s in %T", om.name, om.obj.iface)
	}
	in := make([]reflect.Value, len(args))
	for n := range args {
		in[n] = reflect.ValueOf(args[n])
	}
	out := om.method.Call(in)
	res := make([]interface{}, len(out))
	for n := range out {
		res[n] = out[n].Interface()
	}
	return newCallResult(res), nil
}

type CallResult struct {
	Result []interface{}
	Error  error
}

func newCallResult(res []interface{}) *CallResult {
	cr := &CallResult{Result: res}
	if len(res) == 0 {
		return cr
	}
	errorCandidate := res[len(res)-1]
	if errorCandidate != nil {
		if err, is := errorCandidate.(error); is {
			cr.Error = err
		}
	}
	return cr
}

// IsError checks if the last value is a non-nil error
func (cr *CallResult) IsError() bool {
	return cr.Error != nil
}

func isExportable(field reflect.StructField) bool {
	return field.PkgPath == ""
}
