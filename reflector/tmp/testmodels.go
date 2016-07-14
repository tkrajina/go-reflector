package tmp

type TestStruct struct {
	_          int `bu:"ba"`
	Exported   string
	unexported int `aaa:"bbb" ccc:"ddd"`
}
