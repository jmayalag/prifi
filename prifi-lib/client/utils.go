package client

import "time"

// MsTimeStamp returns the current timestamp, in milliseconds.
func MsTimeStamp() int64 {
	//http://stackoverflow.com/questions/24122821/go-golang-time-now-unixnano-convert-to-milliseconds
	return time.Now().UnixNano() / int64(time.Millisecond)
}
