package main

import (
	"net/http"
	"github.com/tehmoon/errors"
	"github.com/spf13/pflag"
	"sync"
	"log"
	"io"
)

func init() {
	MODULELIST.Register("http", "Connects to an HTTP webserver", NewHTTP)
}

type HTTP struct {
	in chan *Message
	out chan *Message
	sync *sync.WaitGroup
	url string
	method string
	data bool
	reader io.Reader
	writer io.WriteCloser
	req *http.Request
	line bool
}

func (m *HTTP) SetFlagSet(fs *pflag.FlagSet) {
	fs.StringVar(&m.url, "url", "", "HTTP url to query")
	fs.StringVar(&m.method, "method", "GET", "Set the method to query")
	fs.BoolVar(&m.data, "data", false, "Send data from the pipeline to the server")
	fs.BoolVar(&m.line, "line", false, "Read lines from the connection")
}

func (m *HTTP) In(in chan *Message) (chan *Message) {
	m.in = in

	return in
}

func (m *HTTP) Out(out chan *Message) (chan *Message) {
	m.out = out

	return out
}

func (m *HTTP) Init(global *GlobalFlags) (error) {
	if m.data {
		m.reader, m.writer = io.Pipe()
	}

	if global.Line {
		m.line = global.Line
	}

	req, err := http.NewRequest(m.method, m.url, m.reader)
	if err != nil {
		return errors.Wrap(err, "Error generating the http request")
	}

	m.req = req

	return nil
}

func (m HTTP) Start() {
	m.sync.Add(2)

	go func() {
		// This waits until a first message is received from input
		// otherwise for some reason, it makes the http client hang
		wait := make(chan struct{}, 0)

		options := &HTTPOptions{
			Line: m.line,
		}

		go httpStartIn(m.in, m.writer, wait, m.sync)
		go httpStartOut(m.out, m.req, options, wait, m.sync)
	}()
}

func (m HTTP) Wait() {
	m.sync.Wait()

	for range m.in {}
}

func httpStartIn(in chan *Message, writer io.WriteCloser, wait chan struct{}, wg *sync.WaitGroup) {
	defer wg.Done()

	if writer == nil {
		wait <- struct{}{}
		close(wait)

		for range in {}

		return
	}

	started := false
	for message := range in {
		if ! started {
			wait <- struct{}{}
			close(wait)
			started = true
		}

		writer.Write(message.Payload)
	}

	writer.Close()
}

type HTTPOptions struct {
	Line bool
}

func httpStartOut(out chan *Message, req *http.Request, options *HTTPOptions, wait chan struct{}, wg *sync.WaitGroup) {
	defer wg.Done()
	defer close(out)

	client := &http.Client{}

	<- wait

	resp, err := client.Do(req)
	if err != nil {
		log.Println(errors.Wrap(err, "Error in http response"))
		return
	}
	defer resp.Body.Close()

	cb := func(payload []byte) {
		SendMessage(payload, out)
	}

	if options.Line {
		err = ReadDelimStep(resp.Body, '\n', cb)
	} else {
		err = ReadBytesStep(resp.Body, cb)
	}

	if err != nil {
		log.Println(errors.Wrap(err, "Error reading body of http response"))
		return
	}
}

func NewHTTP() (Module) {
	return &HTTP{
		sync: &sync.WaitGroup{},
	}
}
