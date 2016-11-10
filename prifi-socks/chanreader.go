package prifisocks

import "io"

type chanreader struct {
	b   []byte
	c   <-chan []byte
	eof bool
}

// Read reads bytes into an array and returns the number of read bytes.
func (cr *chanreader) Read(p []byte) (n int, err error) {
	if cr.eof {
		return 0, io.EOF
	}
	blen := len(cr.b)
	if blen == 0 {
		cr.b = <-cr.c // read next block from channel
		blen = len(cr.b)
		if blen == 0 { // channel sender signaled EOF
			cr.eof = true
			return 0, io.EOF
		}
	}

	act := min(blen, len(p))
	copy(p, cr.b[:act])
	cr.b = cr.b[act:]
	return act, nil
}

func newChanReader(c <-chan []byte) *chanreader {
	return &chanreader{[]byte{}, c, false}
}

func min(x, y int) int {
	if x < y {
		return x
	}
	return y
}
