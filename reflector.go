package reflector

import (
	"fmt"
	"reflect"
	"strings"
)

type Object struct {
	obj interface{}
}

func panicIfErr(err error) {
	if err != nil {
		panic(err.Error())
	}
}

func New(obj interface{}) *Object {
	return &Object{obj: obj}
}

func (o *Object) IsStructOrPtrToStructUnderlyingType() (bool, bool, reflect.Type) {
	var isStruct, isPtrToStruct bool
	ty := o.Type()
	if ty.Kind() == reflect.Struct {
		isStruct = true
	}
	if ty.Kind() == reflect.Ptr && ty.Elem().Kind() == reflect.Struct {
		ty = ty.Elem()
		isPtrToStruct = true
	}
	return isStruct, isPtrToStruct, ty
}

func (o *Object) Fields() []ObjField {
	return o.fields(reflect.TypeOf(o.obj), false)
}

func (o Object) FieldFlattened() []ObjField {
	return o.fields(reflect.TypeOf(o.obj), true)
}

func (o *Object) fields(ty reflect.Type, flatten bool) []ObjField {
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

func (o Object) IsPtr() bool {
	return o.Kind() == reflect.Ptr
}

func (o *Object) Field(name string) *ObjField {
	return &ObjField{
		obj:  o,
		name: name,
	}
}

func (o Object) Type() reflect.Type {
	return reflect.TypeOf(o.obj)
}

func (o Object) Kind() reflect.Kind {
	return o.Type().Kind()
}

func (o *Object) Method(name string) *ObjMethod {
	return newObjMethod(o, name)
}

func (o Object) Methods() []ObjMethod {
	return nil
}

type ObjField struct {
	obj  *Object
	name string
}

func newObjField(obj *Object, name string) *ObjField {
	return &ObjField{
		obj:  obj,
		name: name,
	}
}

func (of *ObjField) Name() string {
	return of.name
}

func (of *ObjField) Kind() reflect.Kind {
	return reflect.Invalid
}

func (of *ObjField) Type() reflect.Type {
	return nil
}

func (of *ObjField) Valid() bool {
	strct, ptrStrct, ty := of.obj.IsStructOrPtrToStructUnderlyingType()
	if !strct && !ptrStrct {
		return false
	}
	_, found := ty.FieldByName(of.name)
	return found
}

func (of *ObjField) Set(value interface{}) error {
	strct, ptrStrct, ty := of.obj.IsStructOrPtrToStructUnderlyingType()
	fmt.Print(strct, ptrStrct, ty)
	if !strct && !ptrStrct {
		return fmt.Errorf("Cannot set %s because obj is not a pointer to struct", of.name)
	}

	v := reflect.ValueOf(value)
	var field reflect.Value
	if ptrStrct {
		field = reflect.ValueOf(of.obj.obj).Elem().FieldByName(of.name)
	} else {
		field = reflect.ValueOf(of.obj.obj).FieldByName(of.name)
	}

	if !field.IsValid() {
		return fmt.Errorf("Field %s not valid", of.name)
	}
	if !field.CanSet() {
		return fmt.Errorf("Field %s not settable", of.name)
	}

	fmt.Println(ty)
	field.Set(v)
	return nil
}

func (of *ObjField) Get() (interface{}, error) {
	strct, ptrStrct, _ := of.obj.IsStructOrPtrToStructUnderlyingType()
	if !strct && !ptrStrct {
		return nil, fmt.Errorf("Cannot get %s because underlying obj is not a struct", of.name)
	}

	value := reflect.ValueOf(of.obj.obj).FieldByName(of.name).Interface()
	return value, nil
}

type ObjMethod struct {
	obj  *Object
	name string
}

func newObjMethod(obj *Object, name string) *ObjMethod {
	return &ObjMethod{
		obj:  obj,
		name: name,
	}
}

func (om *ObjMethod) IsValid() bool {
	return false
}

func (om *ObjMethod) Call(args []interface{}) ([]interface{}, error) {
	return nil, nil
}
