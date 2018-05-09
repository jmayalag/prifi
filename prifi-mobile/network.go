package prifiMobile

import (
	"github.com/parnurzeal/gorequest"
	"time"
	"strconv"
)

// Used for latency test
type HttpRequestResult struct {
	Latency int64
	StatusCode int
	BodySize int
}

func NewHttpRequestResult() *HttpRequestResult {
	return &HttpRequestResult{0,0,0}
}

/*
 * Request google home page through PriFi
 *
 * It is a method instead of a function due to the type restriction of gomobile.
 */
func (result *HttpRequestResult) RetrieveHttpResponseThroughPrifi(timeout int) error {
	// Get the localhost PriFi server port
	prifiPort, err := GetPrifiPort()
	if err != nil {
		return err
	}

	// Construct the proxy host address
	proxyUrl := "socks5://127.0.0.1:" + strconv.Itoa(prifiPort)

	// Construct a request object with proxy and timeout value
	request := gorequest.New().Proxy(proxyUrl).Timeout(time.Duration(timeout)*time.Second)

	// Used for latency test
	start := time.Now()
	resp, bodyBytes, errs := request.Get("https://www.google.com").EndBytes()
	elapsed := time.Since(start)

	if len(errs) > 0 {
		return errs[0]
	}

	result.Latency = elapsed.Nanoseconds()
	result.StatusCode = resp.StatusCode
	result.BodySize = len(bodyBytes)

	return nil
}
