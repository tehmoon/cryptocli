package main

import (
	"sync"
	"github.com/spf13/pflag"
	"compress/gzip"
	"bytes"
)

func init() {
	MODULELIST.Register("gzip", "Gzip compress", NewGzip)
}

type Gzip struct {
	in chan *Message
	out chan *Message
	wg *sync.WaitGroup
}

func (m Gzip) Init(global *GlobalFlags) (error) {
	return nil
}

func (m Gzip) Start() {
	m.wg.Add(1)

	go startGzip(m.in, m.out, m.wg)
}

func (m Gzip) Wait() {
	m.wg.Wait()

	for range m.in {}
}

func (m *Gzip) In(in chan *Message) (chan *Message) {
	m.in = in

	return in
}

func (m *Gzip) Out(out chan *Message) (chan *Message) {
	m.out = out

	return out
}

func NewGzip() (Module) {
	return &Gzip{
		wg: &sync.WaitGroup{},
	}
}

func startGzipIn(in chan *Message, writer *gzip.Writer, buff *bytes.Buffer, c chan []byte, wg *sync.WaitGroup) {
	for message := range in {
		_, err := writer.Write(message.Payload)
		if err != nil {
			break
		}

		writer.Flush()

		c <- CopyResetBuffer(buff)
	}

	writer.Close()
	c <- CopyResetBuffer(buff)
	close(c)
	wg.Done()
}

func startGzipOut(out chan *Message, c chan []byte, wg *sync.WaitGroup) {
	for payload := range c {
		SendMessage(payload, out)
	}

	close(out)
	wg.Done()
}

func startGzip(in, out chan *Message, wg *sync.WaitGroup) {
	buff := bytes.NewBuffer(nil)
	gzipWriter := gzip.NewWriter(buff)
	c := make(chan []byte, 0)

	wg.Add(2)

	go startGzipIn(in, gzipWriter, buff, c, wg)
	go startGzipOut(out, c, wg)

	wg.Done()
}

func (m *Gzip) SetFlagSet(fs *pflag.FlagSet) {}
