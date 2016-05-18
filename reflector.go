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

func (o *Object) Fields() []ObjField {
	return o.fields(reflect.TypeOf(o.obj))
}

func (o *Object) fields(ty reflect.Type) []ObjField {
	fields := make([]ObjField, 0)

	if ty.Kind() == reflect.Ptr {
		ty = ty.Elem()
	}

	if ty.Kind() != reflect.Struct {
		return fields // No need to populate nonstructs
	}

	for i := 0; i < ty.NumField(); i++ {
		field := ty.Field(i)

		switch field.Type.Kind() {
		case reflect.Struct:
			if field.Anonymous && string(field.Name[0]) == strings.ToUpper(string(field.Name[0])) {
				fields = append(fields, o.fields(field.Type)...)
			} else {
				fields = append(fields, ObjField{obj: o, name: field.Name})
			}
		default:
			fields = append(fields, ObjField{obj: o, name: field.Name})
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

func (o Object) FieldsDeep() []ObjField {
	return nil
}

func (o Object) Method(name string) *ObjMethod {
	return nil
}

func (o Object) Methods() []ObjMethod {
	return nil
}

type ObjField struct {
	obj  *Object
	name string
}

func (of *ObjField) Name() string {
	return of.name
}

func (of *ObjField) Valid() bool {
	ty := of.obj.Type()
	if ty.Kind() == reflect.Ptr {
		ty = ty.Elem()
	}
	if ty.Kind() != reflect.Struct {
		return false
	}
	_, found := ty.FieldByName(of.name)
	return found
}

func (of *ObjField) Set(value interface{}) error {
	ty := of.obj.Type()
	if ty.Kind() == reflect.Ptr {
		ty = ty.Elem()
	} else {
		return fmt.Errorf("Cannot set %s because obj is not a pointer", of.name)
	}
	if ty.Kind() != reflect.Struct {
		return fmt.Errorf("Cannot set %s because obj is not a struct", of.name)
	}
	v := reflect.ValueOf(value)
	reflect.ValueOf(of.obj.obj).Elem().FieldByName(of.name).Set(v)
	return nil
}

func (of *ObjField) Get() (interface{}, error) {
	field, found := reflect.TypeOf(of.obj).FieldByName(of.name)
	_ = field
	if !found {
		return nil, fmt.Errorf("Cannot find %s", of.name)
	}
	return nil, nil
}

type ObjMethod struct {
}

func (om *ObjMethod) Call(args []interface{}) ([]interface{}, error) {
	return nil, nil
}
