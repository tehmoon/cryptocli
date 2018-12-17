package main

import (
	"bytes"
)

// Copy the content of a *io.Buffer and return it
func CopyResetBuffer(buff *bytes.Buffer) ([]byte) {
	data := buff.Bytes()
	payload := make([]byte, len(data))

	copy(payload, data)
	buff.Reset()

	return payload
}
