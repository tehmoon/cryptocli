package main

import (
	"net/http"
	"github.com/tehmoon/errors"
	"github.com/spf13/pflag"
	"sync"
	"log"
	"io"
	"crypto/tls"
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
	insecure bool
	cancel chan struct{}
}

func (m *HTTP) SetFlagSet(fs *pflag.FlagSet) {
	fs.StringVar(&m.url, "url", "", "HTTP url to query")
	fs.StringVar(&m.method, "method", "GET", "Set the method to query")
	fs.BoolVar(&m.data, "data", false, "Send data from the pipeline to the server")
	fs.BoolVar(&m.insecure, "insecure", false, "Don't valid the TLS certificate chain")
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
			Insecure: m.insecure,
		}

		go httpStartIn(m.in, m.writer, wait, m.sync)
		go httpStartOut(m.out, m.req, options, wait, m.sync, m.cancel)
	}()
}

func (m HTTP) Wait() {
	m.cancel <- struct{}{}
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
			close(wait)
			started = true
		}

		writer.Write(message.Payload)
	}

	if ! started {
		close(wait)
	}

	writer.Close()
}

type HTTPOptions struct {
	Insecure bool
}

// Copy http.DefaultTransport in order to change it and keep the defaults
// Returns http.DefaultTransport if cannot assert the http.RoundTripper interface
func httpCreateTransport(options *HTTPOptions) (http.RoundTripper) {
	var (
		tr = &http.Transport{}
		rt = http.DefaultTransport
	)

	transport, ok := http.DefaultTransport.(*http.Transport)
	if ok {
		*tr = *transport

		tr.TLSClientConfig = httpCreateTLSConfig(options)

		rt = tr
	}

	return rt
}

func httpCreateTLSConfig(options *HTTPOptions) (config *tls.Config) {
	return &tls.Config{
		InsecureSkipVerify: options.Insecure,
	}
}

func httpStartOut(out chan *Message, req *http.Request, options *HTTPOptions, wait chan struct{}, wg *sync.WaitGroup, cancel chan struct{}) {
	canceled := false
	defer wg.Done()
	defer func() {
		if ! canceled {
			<- cancel
		}
	}()
	defer close(out)

	client := &http.Client{
		Transport: httpCreateTransport(options),
	}

	<- wait

	resp, err := client.Do(req)
	if err != nil {
		log.Println(errors.Wrap(err, "Error in http response"))
		return
	}
	defer resp.Body.Close()

	err = ReadBytesStep(resp.Body, func(payload []byte) (bool) {
		select {
			case <- cancel:
				canceled = true
				return false
			default:
				SendMessage(payload, out)
		}

		return true
	})
	if err != nil {
		log.Println(errors.Wrap(err, "Error reading body of http response"))
		return
	}
}

func NewHTTP() (Module) {
	return &HTTP{
		sync: &sync.WaitGroup{},
		cancel: make(chan struct{}),
	}
}
