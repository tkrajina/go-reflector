package reflector

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
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

func (o *Obj) Fields() []ObjField {
	return o.fields(reflect.TypeOf(o.iface), false)
}

func (o Obj) FieldsFlattened() []ObjField {
	return o.fields(reflect.TypeOf(o.iface), true)
}

func (o *Obj) fields(ty reflect.Type, flatten bool) []ObjField {
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
		if string(field.Name[0]) == strings.ToUpper(string(field.Name[0])) {
			if flatten && k == reflect.Struct && field.Anonymous {
				fields = append(fields, o.fields(field.Type, flatten)...)
			} else {
				fields = append(fields, *newObjField(o, field.Name))
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
	return &ObjField{
		obj:  o,
		name: name,
	}
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
}

func newObjField(obj *Obj, name string) *ObjField {
	return &ObjField{
		obj:  obj,
		name: name,
	}
}

func (of *ObjField) Name() string {
	return of.name
}

func (of *ObjField) Kind() reflect.Kind {
	ty, err := of.Type()
	if err != nil {
		return reflect.Invalid
	}
	return ty.Kind()
}

func (of *ObjField) Type() (reflect.Type, error) {
	value, err := of.Get()
	if err != nil {
		return nil, fmt.Errorf("Invalid field %s in %T", of.name, of.obj.iface)
	}
	return reflect.TypeOf(value), nil
}

func (of *ObjField) Tag(tag string) (string, error) {
	_, field, err := of.field()
	if err != nil {
		return "", err
	}
	return (*field).Tag.Get(tag), nil
}

func (of *ObjField) Tags() (map[string]string, error) {
	_, field, err := of.field()
	if err != nil {
		return nil, err
	}

	res := map[string]string{}
	tag := (*field).Tag

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
	_, field, err := of.field()
	if err != nil {
		return nil, err
	}
	return strings.Split((*field).Tag.Get(tag), ","), nil
}

func (of *ObjField) Valid() bool {
	if !of.obj.IsStructOrPtrToStruct() {
		return false
	}
	_, found := of.obj.underlyingType.FieldByName(of.name)
	return found
}

func (of *ObjField) Anonymous() bool {
	field, found := of.obj.underlyingType.FieldByName(of.name)
	if !found {
		return false
	}
	return field.Anonymous
}

func (of *ObjField) Set(value interface{}) error {
	if !of.obj.IsStructOrPtrToStruct() {
		return fmt.Errorf("Cannot set %s in %T because obj is not a pointer to struct", of.name, of.obj.iface)
	}

	v := reflect.ValueOf(value)
	var field reflect.Value
	if of.obj.isPtrToStruct {
		field = reflect.ValueOf(of.obj.iface).Elem().FieldByName(of.name)
	} else {
		field = reflect.ValueOf(of.obj.iface).FieldByName(of.name)
	}

	if !field.IsValid() {
		return fmt.Errorf("Field %s in %T not valid", of.name, of.obj.iface)
	}
	if !field.CanSet() {
		return fmt.Errorf("Field %s in %T not settable", of.name, of.obj.iface)
	}

	field.Set(v)

	return nil
}

func (of *ObjField) field() (*reflect.Value, *reflect.StructField, error) {
	if !of.obj.IsStructOrPtrToStruct() {
		return nil, nil, fmt.Errorf("Cannot get %s in %T because underlying obj is not a struct", of.name, of.obj.iface)
	}

	var valueField reflect.Value
	var structField reflect.StructField
	var found bool
	if of.obj.isPtrToStruct {
		valueField = reflect.ValueOf(of.obj.iface).Elem().FieldByName(of.name)
		structField, found = reflect.TypeOf(of.obj.iface).Elem().FieldByName(of.name)
	} else {
		valueField = reflect.ValueOf(of.obj.iface).FieldByName(of.name)
		structField, found = reflect.TypeOf(of.obj.iface).FieldByName(of.name)
	}

	if !found {
		return nil, nil, fmt.Errorf("Field not found %s in %T", of.name, of.obj.iface)
	}

	return &valueField, &structField, nil
}

func (of *ObjField) Get() (interface{}, error) {
	fptr, _, err := of.field()
	if err != nil {
		return nil, err
	}
	field := *fptr

	if !field.IsValid() {
		return nil, fmt.Errorf("Invalid field %s in %T", of.name, of.obj.iface)
	}

	value := field.Interface()
	return value, nil
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
