package prifiMobile

import (
	"testing"
	"fmt"
)

func TestMakeHttpRequestThroughPrifi(t *testing.T) {
	r := NewHttpRequestResult()
	fmt.Println(r)

	e := r.RetrieveHttpResponseThroughPrifi(5)
	if e != nil {
		fmt.Println(e)
	}
	fmt.Println(r)
}
