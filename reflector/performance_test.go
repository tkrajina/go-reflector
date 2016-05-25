package reflector

import (
	"fmt"
	"os"
	"strconv"
	"testing"
	"time"
)

// Utility to test performance of some typical operations, run with:
// N=1000000 go test -v ./... -run=TestPerformance
func TestPerformance(t *testing.T) {
	n, _ := strconv.ParseInt(os.Getenv("N"), 10, 64)
	if n <= 0 {
		n = 1000
	}
	p := &Person{}
	obj := New(p)
	started := time.Now()
	for i := 0; i < int(n); i++ {
		obj.Field("Number").Set(i)
		if p.Number != i {
			panic("Should be " + string(i))
		}
		number, err := obj.Field("Number").Get()
		if err != nil {
			panic("Number is valid")
		}
		if number.(int) != i {
			panic("Should be " + string(i))
		}
		res, err := obj.Method("Add").Call(1, 2, 3)
		if err != nil {
			panic("shouldn't be an error")
		}
		if res.IsError() {
			panic("method shouldn't return an error")
		}
		if len(res.Result) != 1 && res.Result[0].(int) != 6 {
			panic("result should be 6")
		}
	}
	ended := time.Now()
	fmt.Println("n=", n)
	fmt.Println("started:", started.Format("2006-01-02 15:04:05.123"))
	fmt.Println("ended:", ended.Format("2006-01-02 15:04:05.123"))
	fmt.Printf("duration: %fs\n", ended.Sub(started).Seconds())
}
