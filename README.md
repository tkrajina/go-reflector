# Golang reflector

First of all, don't use reflection if you don't have to.

But if you really have to... This library offers a simplified Golang reflection abstraction.

**This lib is still a work in progress**

## Getting and setting fields

Let's suppose we have structs like this one:

    type Address struct {
        Street string `tag:"be" tag2:"1,2,3"`
        Number int    `tag:"bi"`
    }

    type Person struct {
        Name string `tag:"bu"`
        Address
    }

    func (p Person) Hi(name string) string {
        return fmt.Sprintf("Hi %s my name is %s", name, p.Name)
    }

First, initialize the reflector's object wrapper:

    import "github.com/tkrajina/go-reflector/reflector"

	p := Person{}
	obj := reflector.New(p)

Check if field is valid:

    obj.Field("Name").IsValid()

Get field value:

    val, err := obj.Field("Name").Get()

Set field value:

	p := Person{}
	obj := reflector.New(&p)
    err := obj.Field("Name").Set("Something")

Don't forget to use a pointer in `New()`, otherwise setters won't work (they will work but on a copy of your data).

## Listing fields

There are three ways to list fields:

 * List all fields: This will include Anonymous structs **and** fields declared in those anonymous structs (`Name`, `Address`, `Street`, `Number`).
 * List flattened fields: Includes fields declared in anonymous structs **without** those anonymous structs (`Name`, `Street`, `Number`).
 * List nonflattened fields: Includes fields anonymous structs **without** theis fields (`Name`, `Address`).

Depending on which listing you want, you can use:

    fields := obj.fieldsAll()
    fields := obj.FieldsFlattened()
    fields := obj.Fields()

Note that because of those anonymous structs, some fields can be returned twice.
In most cases this is not a desired situation, if you want to use reflector to detect such situations, you can use...

    doubleDeclaredFields := obj.FindDoubleFields()
    if len(doubleDeclaredFields) > 0 {
        fmt.Println("Detected multiple fields with same name:", doubleDeclaredFields)
    }

## Calling methods

	obj := reflector.New(&Person{})
    resp, err := obj.Method("Hi").Call("John", "Smith")

The `err` is not nil only if something was wrong with the method (for example invalid method name, or wrong argument number/types), not with the actual method call.
If the call finished, `err` will be `nil`.
If the method call returned an err, you can check it with:

    if resp.IsError() {
        fmt.Println("Got an error:", resp.Error.Error())
    } else {
        fmt.Println("Method call response:", resp.Result)
    }

## Listing methods

    for _, method := range obj.Methods() {
        fmt.Println("Method", method.Name(), "with input types", method.InTypes(), "and output types", method.OutTypes())
    }

License
-------

Reflector is licensed under the [Apache License, Version 2.0](http://www.apache.org/licenses/LICENSE-2.0)
