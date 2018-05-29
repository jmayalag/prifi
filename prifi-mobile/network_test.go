package prifiMobile

import (
	"fmt"
	"testing"
)

func TestMakeHttpRequestThroughPrifi(t *testing.T) {
	r := NewHttpRequestResult()
	fmt.Println(r)

	e := r.RetrieveHttpResponseThroughPrifi("http://128.178.151.111", 5, false)
	if e != nil {
		fmt.Println(e)
	}
	fmt.Println(r)
}
