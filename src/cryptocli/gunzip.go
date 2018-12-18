package main

import (
	"sync"
	"github.com/spf13/pflag"
	"compress/gzip"
	"io"
	"github.com/tehmoon/errors"
	"log"
)

func init() {
	MODULELIST.Register("gunzip", "Gunzip de-compress", NewGunzip)
}

type Gunzip struct {
	in chan *Message
	out chan *Message
	wg *sync.WaitGroup
	reader *io.PipeReader
	writer *io.PipeWriter
	line bool
}

func (m *Gunzip) Init(global *GlobalFlags) (error) {
	if global.Line {
		m.line = global.Line
	}

	m.reader, m.writer = io.Pipe()

	return nil
}

func (m Gunzip) Start() {
	m.wg.Add(1)

	go startGunzipIn(m.in, m.writer)
	go startGunzipOut(m.out, m.reader, m.line, m.wg)
}

func (m Gunzip) Wait() {
	m.wg.Wait()

	for range m.in {}
}

func (m *Gunzip) In(in chan *Message) (chan *Message) {
	m.in = in

	return in
}

func (m *Gunzip) Out(out chan *Message) (chan *Message) {
	m.out = out

	return out
}

func NewGunzip() (Module) {
	return &Gunzip{
		wg: &sync.WaitGroup{},
	}
}

func startGunzipIn(in chan *Message, writer io.WriteCloser) {
	for message := range in {
		_, err := writer.Write(message.Payload)
		if err != nil {
			log.Println(errors.Wrap(err, "Error writing data to pipe in gunzip"))
			break
		}
	}

	writer.Close()
}

func startGunzipOut(out chan *Message, reader io.Reader, line bool, wg *sync.WaitGroup) {
	defer wg.Done()
	defer close(out)

	gzipReader, err := gzip.NewReader(reader)
	if err != nil {
		log.Println(errors.Wrap(err, "Error initializing gunzip reader"))
		return
	}

	cb := func(payload []byte) {
		SendMessage(payload, out)
	}

	if line {
		err = ReadDelimStep(gzipReader, '\n', cb)
	} else {
		err = ReadBytesStep(gzipReader, cb)
	}

	if err != nil {
		log.Println(errors.Wrap(err, "Error reading gzip reader in gunzip"))
		return
	}
}

func (m *Gunzip) SetFlagSet(fs *pflag.FlagSet) {
	fs.BoolVar(&m.line, "line", false, "Read lines from the de-compressed data")
}
