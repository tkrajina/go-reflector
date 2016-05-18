package reflector

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

type Address struct {
	Street string `tag:"be"`
	Number int    `tag:"bi"`
}

type Person struct {
	Name string `tag:"bu"`
	Address
}

func (p Person) Hi(name string) string {
	return fmt.Sprintf("Hi %s my name is %s", name, p.Name)
}

func TestListFields(t *testing.T) {
	p := Person{}
	obj := New(p)

	assert.Equal(t, len(obj.Fields()), 3)

	/*
		for _, field := range obj.Fields() {
		}

		err := obj.Field("Address").Set("bu")
		panicIfErr(err)

		value, err := obj.Field("Address").Get()
		panicIfErr(err)
		fmt.Println("value %s", value)

		res, err := obj.Method("Hi").Call([]interface{}{"John"})
		panicIfErr(err)
		fmt.Println("res=", res)

		for _, field := range obj.Fields() {
			fmt.Println("Field:", field)
		}

		for _, method := range obj.Methods() {
			fmt.Println("Method:", method)
		}

		fmt.Println("%#v", obj)
	*/
}
