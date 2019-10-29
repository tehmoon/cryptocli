package main

import (
	"bytes"
	"sync"
	"regexp"
	"strings"
	"net/http"
)

// Copy the content of a *io.Buffer and return it
func CopyResetBuffer(buff *bytes.Buffer) ([]byte) {
	data := buff.Bytes()
	payload := make([]byte, len(data))

	copy(payload, data)
	buff.Reset()

	return payload
}

func DrainChannel(inc MessageChannel, wg *sync.WaitGroup) {
	for range inc {}
	if wg != nil {
		wg.Done()
	}
}

var ParseHTTPHeadersRE, _ = regexp.Compile(`^\ +`)

func ParseHTTPHeaders(rawHeaders []string) (headers http.Header) {
	headers = make(http.Header)

	for _, rawHeader := range rawHeaders {
		header := strings.Split(rawHeader, ":")

		key := header[0]
		value := ""
		if len(header) > 1 {
			value = strings.Join(header[1:], ":")
		}

		value = ParseHTTPHeadersRE.ReplaceAllString(value, "")

		headers.Set(key, value)
	}

	return headers
}
