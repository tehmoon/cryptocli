package main

import (
	"io"
)

const (
	ReaderMaxPowerSize uint = 24
	ReaderMinPowerSize uint = 8
)

func ReadBytesSendMessages(r io.Reader, c chan *Message) (error) {
	return ReadBytesStep(r, func(payload []byte) {
		SendMessage(payload, c)
	})
}

// Allocate a buffer and read from the reader.
// If the buffer is full, next read will allocate more.
// If the buffer is less than full for 2 times in a row,
// it will allocate less on the next run.
func ReadBytesStep(r io.Reader, cb func([]byte)) (error) {
	var (
		err error
		i int
		power = ReaderMinPowerSize
		down = 0
	)

	for {
		l := 2 << power
		buff := make([]byte, l)

		i, err = r.Read(buff)
		if err != nil {
			cb(buff[:i])
			break
		}

		cb(buff[:i])

		if i >= l && power != ReaderMaxPowerSize {
			power++
			continue
		}

		down++

		if down == 3 && power != ReaderMaxPowerSize {
			power--
			down = 0
		}
	}

	return err
}
