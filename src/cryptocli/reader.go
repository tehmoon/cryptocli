package main

import (
	"io"
	"github.com/tehmoon/errors"
)

const (
	ReaderMaxPowerSize uint = 24
	ReaderMinPowerSize uint = 8
)

func ReadBytesSendMessages(r io.Reader, c chan []byte) (error) {
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

type ChannelReader struct {
	crumb []byte
	c chan []byte
}


// Not thread safe
func NewChannelReader(c chan []byte) (cr *ChannelReader) {
	cr = &ChannelReader{
		crumb: make([]byte, 0),
		c: c,
	}

	return cr
}

func (cr *ChannelReader) Read(p []byte) (n int, err error) {
	if cr.crumb == nil {
		return -1, ErrChannelReaderClosed
	}

	if len(cr.crumb) >= len(p) {
		copy(p, cr.crumb[:len(p)])
		cr.crumb = cr.crumb[len(p):]
		return len(p), nil
	}

	for {
		payload, opened := <- cr.c
		if ! opened {
			copy(p, cr.crumb[:len(p)])
			cr.crumb = cr.crumb[len(p):]
			return len(p), io.EOF
		}

		cr.crumb = append(cr.crumb, payload...)

		if len(cr.crumb) >= len(p) {
			copy(p, cr.crumb[:len(p)])
			cr.crumb = cr.crumb[len(p):]
			return len(p), nil
		}
	}

	return -1, errors.New("Un-handled error")
}

func (cr *ChannelReader) Crumbs() (payload []byte, err error) {
	if cr.crumb == nil {
		return nil, ErrChannelReaderClosed
	}

	crumb := cr.crumb
	cr.crumb = make([]byte, 0)
	return crumb, nil
}

func (cr *ChannelReader) ReadMessage() (payload []byte, err error) {
	if len(cr.crumb) != 0 {
		crumb := cr.crumb
		cr.crumb = make([]byte, 0)
		return crumb, nil
	}

	payload, opened := <- cr.c
	if ! opened {
		return nil, io.EOF
	}

	return payload, nil
}

var ErrChannelReaderClosed = errors.New("Channel reader already closed")

func (cr *ChannelReader) ReadLine() (payload []byte, err error) {
	if cr.crumb == nil {
		return nil, ErrChannelReaderClosed
	}

	if len(cr.crumb) != 0 {
		for i, c := range cr.crumb {
			if c == '\n' {
				payload = cr.crumb[:i]
				cr.crumb = cr.crumb[i:]
				return payload, nil
			}
		}
	}

	for {
		payload, opened := <- cr.c
		if ! opened {
			crumb := cr.crumb
			cr.crumb = make([]byte, 0)
			return crumb, io.EOF
		}

		for i, c := range payload {
			if c == '\n' {
				payload = append(cr.crumb, payload[:i]...)
				cr.crumb = payload[i:]
				return payload, nil
			}
		}

		cr.crumb = append(cr.crumb, payload...)
	}
}

func (cr *ChannelReader) Close() error {
	if cr.crumb == nil {
		return ErrChannelReaderClosed
	}

	DrainChannel(cr.c, nil)
	cr.crumb = nil

	return nil
}
