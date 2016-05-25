package reflector

import (
	"fmt"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// Utility to test performance of some typical operations, run with:
// N=1000000 go test -v ./... -run=TestPerformance
// Iy you change anything here, change TestPerformancePlain too!
func TestPerformance(t *testing.T) {
	n, _ := strconv.ParseInt(os.Getenv("N"), 10, 64)
	if n <= 0 {
		n = 1000
	}
	started := time.Now()
	for i := 0; i < int(n); i++ {
		p := &Person{}
		obj := New(p)

		err := obj.Field("Number").Set(i)
		if err != nil {
			panic("Should not error")
		}
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

	assert.Equal(t, 1, metadataCached, "Only 1 metadata must be cached")
	assert.Equal(t, 1, len(metadataCache), "Only 1 metadata must be cached")

	ended := time.Now()
	fmt.Println("WITH REFLECTION")
	fmt.Println("    n=", n)
	fmt.Println("    started:", started.Format("2006-01-02 15:04:05.123"))
	fmt.Println("    ended:", ended.Format("2006-01-02 15:04:05.123"))
	fmt.Printf("    duration: %fs\n", ended.Sub(started).Seconds())
}

func TestPerformancePlain(t *testing.T) {
	n, _ := strconv.ParseInt(os.Getenv("N"), 10, 64)
	if n <= 0 {
		n = 1000
	}
	started := time.Now()
	for i := 0; i < int(n); i++ {
		p := &Person{}

		p.Number = i
		number := p.Number
		if number != i {
			panic("Should be " + string(i))
		}
		res := p.Add(1, 2, 3)
		if res != 6 {
			panic("result should be 6")
		}
	}

	ended := time.Now()
	fmt.Println("WITHOUT REFLECTION")
	fmt.Println("    n=", n)
	fmt.Println("    started:", started.Format("2006-01-02 15:04:05.123"))
	fmt.Println("    ended:", ended.Format("2006-01-02 15:04:05.123"))
	fmt.Printf("    duration: %fs\n", ended.Sub(started).Seconds())
}
