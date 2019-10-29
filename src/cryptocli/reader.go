package main

import (
	"io"
)

const (
	ReaderMaxPowerSize uint = 24
	ReaderMinPowerSize uint = 8
)

func ReadBytesSendMessages(r io.Reader, c MessageChannel) (error) {
	return ReadBytesStep(r, func(payload []byte) (bool) {
		c <- payload

		return true
	})
}

// Allocate a buffer and read from the reader.
// If the buffer is full, next read will allocate more.
// If the buffer is less than full for 2 times in a row,
// it will allocate less on the next run.
// The callback will pass that buffer. If the callback return false,
// it stops reading and returns EOF
func ReadBytesStep(r io.Reader, cb func([]byte) (bool)) (error) {
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
			if i != 0 {
				cb(buff[:i])
			}

			break
		}

		cont := cb(buff[:i])
		if ! cont {
			err = io.EOF
			break
		}

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

type MessageReader struct {
	reader *io.PipeReader
	writer *io.PipeWriter
	sync chan struct{}
	err error
}

// Wrap chan *Message into a io.Pipe to make a io.ReadCloser()
func NewMessageReader(c chan []byte) (*MessageReader) {
	mr := &MessageReader{
		sync: make(chan struct{}, 0),
	}

	mr.reader, mr.writer = io.Pipe()

	go func() {
		opened := false

		LOOP: for {
			select {
				case payload, ok := <- c:
					if ! ok {
						opened = true
						break LOOP
					}

					_, mr.err = mr.writer.Write(payload)
					if mr.err != nil {
						opened = true
						break LOOP
					}
				case <- mr.sync:
					break LOOP
			}
		}

		if opened {
			mr.writer.CloseWithError(mr.err)
			<- mr.sync
		}
	}()

	return mr
}

func (mr MessageReader) Read(p []byte) (int, error) {
	return mr.reader.Read(p)
}

func (mr MessageReader) Close() (error) {
	mr.writer.CloseWithError(mr.err)
	mr.sync <- struct{}{}

	return mr.err
}
