package reflector

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

// TODOs
// - Most of the data needed for reflection is retrieved in constructors, but most of them can be cached for future use.

type fieldListingType int

const (
	fieldsAll                fieldListingType = iota
	fieldsFlattenAnonymous                    = iota
	fieldsNoFlattenAnonymous                  = iota
)

// Obj is a wrapper for golang values which needed to be reflected. The value can be of any kind and any type.
type Obj struct {
	iface interface{}

	isStruct      bool
	isPtrToStruct bool

	// If ptr to struct, this field will contain the type of that struct
	underlyingType reflect.Type

	objType reflect.Type
	objKind reflect.Kind
}

// NewFromType creates a new Obj but using reflect.Type
func NewFromType(ty reflect.Type) *Obj {
	if ty == nil {
		return New(nil)
	}
	return New(reflect.New(ty).Interface())
}

// New initializes a new Obj wrapper
func New(obj interface{}) *Obj {
	o := &Obj{iface: obj}

	if obj == nil {
		o.objKind = reflect.Invalid
		return o
	}
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

// IsValid checks if the underlying objects is valid. Nil is an invalid value, for example.
func (o *Obj) IsValid() bool {
	return o.objKind != reflect.Invalid
}

// Fields returns fields. Don't list fields inside Anonymous fields as distinct fields
func (o *Obj) Fields() []ObjField {
	return o.fields(reflect.TypeOf(o.iface), fieldsNoFlattenAnonymous)
}

// FieldsFlattened returns fields. Will not list Anonymous fields but it will list fields declared in those anonymous fields
func (o Obj) FieldsFlattened() []ObjField {
	return o.fields(reflect.TypeOf(o.iface), fieldsFlattenAnonymous)
}

// FieldsAll returns fields. List both anonymous fields and fields declared inside anonymous fields.
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
	var fields []ObjField

	if !o.IsValid() {
		return fields
	}

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

// IsPtr checks if the value is a pointer
func (o Obj) IsPtr() bool {
	return o.objKind == reflect.Ptr
}

// IsStructOrPtrToStruct checks if the value is a struct or a pointer to a struct
func (o Obj) IsStructOrPtrToStruct() bool {
	return o.isStruct || o.isPtrToStruct
}

// Field get a field wrapper. Note that the field name can be invalid. You can check the field validity using ObjField.IsValid()
func (o *Obj) Field(name string) *ObjField {
	return newObjField(o, name)
}

// Type returns the value type. If kind is invalid, this will return a zero filled reflect.Type
func (o Obj) Type() reflect.Type {
	return o.objType
}

// Kind returns the value's kind
func (o Obj) Kind() reflect.Kind {
	return o.objKind
}

// Method returns a new method wrapper. The method name can be invalid, check the method validity with ObjMethod.IsValid()
func (o *Obj) Method(name string) *ObjMethod {
	return newObjMethod(o, name)
}

// Methods returns the list of all methods
func (o *Obj) Methods() []ObjMethod {
	res := []ObjMethod{}
	if !o.IsValid() {
		return res
	}
	ty := o.Type()
	for i := 0; i < ty.NumMethod(); i++ {
		method := ty.Method(i)
		res = append(res, *newObjMethod(o, method.Name))
	}
	return res
}

// ObjField is a wrapper for the object's field.
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

// IsValid checks if the fiels is valid.
func (of *ObjField) IsValid() bool {
	return of.valid
}

// Name returns the field's name
func (of *ObjField) Name() string {
	return of.name
}

// Kind returns the field's kind
func (of *ObjField) Kind() reflect.Kind {
	return of.fieldKind
}

// Type returns the field's type
func (of *ObjField) Type() reflect.Type {
	return of.fieldType
}

// Tag returns the value of this specific tag or error if the field is invalid
func (of *ObjField) Tag(tag string) (string, error) {
	if err := of.assertValid(); err != nil {
		return "", err
	}
	return of.structField.Tag.Get(tag), nil
}

// Tags returns the map of all fields or error (if the field is invalid)
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

// IsAnonymous checks if this is an anonymous (embedded) field
func (of *ObjField) IsAnonymous() bool {
	if err := of.assertValid(); err != nil {
		return false
	}
	field, found := of.obj.underlyingType.FieldByName(of.name)
	if !found {
		return false
	}
	return field.Anonymous
}

// IsSettable checks if this field is settable
func (of *ObjField) IsSettable() bool {
	return of.valueField.CanSet()
}

// Set sets a value for this field or error if field is invalid (or not settable)
func (of *ObjField) Set(value interface{}) error {
	if err := of.assertValid(); err != nil {
		return err
	}

	if !of.IsSettable() {
		return fmt.Errorf("Field %s in %T not settable", of.name, of.obj.iface)
	}

	of.valueField.Set(reflect.ValueOf(value))

	return nil
}

// Get gets the field value (of error if field is invalid)
func (of *ObjField) Get() (interface{}, error) {
	if err := of.assertValid(); err != nil {
		return nil, err
	}

	return of.valueField.Interface(), nil
}

// ObjMethod is a wrapper for an object method. The name of the method can be invalid.
type ObjMethod struct {
	obj    *Obj
	name   string
	method reflect.Value
	valid  bool
}

func newObjMethod(obj *Obj, name string) *ObjMethod {
	res := &ObjMethod{
		obj:  obj,
		name: name,
	}
	if !res.obj.IsValid() {
		res.valid = false
	} else {
		res.method = reflect.ValueOf(obj.iface).MethodByName(name)
		res.valid = res.method.IsValid()
	}
	return res
}

// Name returns the method's name
func (om *ObjMethod) Name() string {
	return om.name
}

// InTypes returns an slice with this method's input types.
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

// OutTypes returns an slice with this method's output types.
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

// IsValid returns this method's validity
func (om *ObjMethod) IsValid() bool {
	return om.valid
}

// Call calls this method. Note that in the error returning value is not the error from the method call
func (om *ObjMethod) Call(args ...interface{}) (*CallResult, error) {
	if !om.obj.IsValid() {
		return nil, fmt.Errorf("Invalid object type %T for method %s", om.obj.iface, om.name)
	}
	if !om.IsValid() {
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

// CallResult is a wrapper of a method call result
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
