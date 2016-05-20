# Golang reflector

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

First, initialize the reflector's object wrapper:

    import "github.com/tkrajina/go-reflector/reflector"

	p := Person{}
	obj := reflector.New(p)

Check if field is valid:

    obj.Field("Name").IsValid()

Get field value:

    val, err := obj.Field("Name").Get()

Set field value:

    err := obj.Field("Name").Set("Something")

## Listing fields

There are three ways to list fields:

 * List all fields: This will include Anonymous structs **and** fields declared in those anonymous structs (`Name`, `Address`, `Street`, `Number`).
 * List flattened fields: Includes fields declared in anonymous structs **without** those anonymous structs (`Name`, `Street`, `Address`).
 * List nonflattened fields: Includes fields anonymous structs **without** theis fields (`Name`, `Address`).

Depending on which listing you want, you can use:

    fields := obj.fieldsAll()
    fields := obj.FieldsFlattened()
    fields := obj.Fields()

Note that because of those anonymous structs, some fields can be returned twice.
In most cases this is not a desired situation, if you want to use reflector to detect such situations, you can use...

    doubleDeclaredFields := obj.FindDoubleFields()

## Calling methods

## Listing methods

# License
