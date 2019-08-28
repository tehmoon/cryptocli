package main

import (
	"net/http"
	"github.com/tehmoon/errors"
	"github.com/spf13/pflag"
	"sync"
	"log"
	"io"
	"crypto/tls"
	"time"
	"strings"
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
	dataTimeout time.Duration
	insecure bool
	cancel chan struct{}
	rawHeaders []string
	headers http.Header
}

func (m *HTTP) SetFlagSet(fs *pflag.FlagSet) {
	fs.StringVar(&m.url, "url", "", "HTTP url to query")
	fs.StringVar(&m.method, "method", "GET", "Set the method to query")
	fs.DurationVar(&m.dataTimeout, "data-timeout", 5 * time.Second, "Wait before closing the input pipeline")
	fs.BoolVar(&m.insecure, "insecure", false, "Don't valid the TLS certificate chain")
	fs.StringArrayVar(&m.rawHeaders, "header", make([]string, 0), "Send HTTP Headers")
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
	for _, rawHeader := range m.rawHeaders {
		header := strings.Split(rawHeader, ":")

		key := header[0]
		value := ""
		if len(header) > 1 {
			value = strings.Join(header[1:], ":")
		}

		m.headers.Set(key, value)
	}

	return nil
}

func (m HTTP) Start() {
	m.sync.Add(2)

	go func() {
		// This waits until a first message is received from input
		// otherwise for some reason, it makes the http client hang
		reqc := make(chan *HttpRequestC, 0)

		options := &HTTPOptions{
			Insecure: m.insecure,
			Url: m.url,
			Method: m.method,
			DataTimeout: m.dataTimeout,
			Headers: m.headers,
		}

		go httpStartIn(m.in, options, reqc, m.sync)
		go httpStartOut(m.out, httpCreateTransport(options), reqc, m.sync, m.cancel)
	}()
}

func (m HTTP) Wait() {
	m.cancel <- struct{}{}
	m.sync.Wait()

	for range m.in {}
}

type HttpRequestC struct {
	Request *http.Request
	Error error
}

func httpStartIn(in chan *Message, options *HTTPOptions, reqc chan *HttpRequestC, wg *sync.WaitGroup) {
	defer wg.Done()

	reader, writer := io.Pipe()

	started := false
	if options.DataTimeout > 0 {
		select {
			case <- time.NewTicker(options.DataTimeout).C:
				log.Println("No data received from the pipeline, sending the request...")
			case message, closed := <- in:
				if ! closed {
					break
				}

				started = true
				req, err := http.NewRequest(options.Method, options.Url, reader)
				req.Header = options.Headers
				reqc <- &HttpRequestC{Request: req, Error: err,}
				writer.Write(message.Payload)
		}
	}

	if ! started {
		req, err := http.NewRequest(options.Method, options.Url, reader)
		req.Header = options.Headers
		reqc <- &HttpRequestC{Request: req, Error: err,}
		writer.Close()

		for range in {}

		return
	}

	for message := range in {
		writer.Write(message.Payload)
	}

	writer.Close()
}

type HTTPOptions struct {
	Insecure bool
	Url string
	Method string
	DataTimeout time.Duration
	Headers http.Header
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

func httpStartOut(out chan *Message, transport http.RoundTripper, reqc chan *HttpRequestC, wg *sync.WaitGroup, cancel chan struct{}) {
	canceled := false
	defer wg.Done()
	defer func() {
		if ! canceled {
			<- cancel
		}
	}()
	defer close(out)

	r := <- reqc
	if r.Error != nil {
		log.Println(errors.Wrap(r.Error, "Error generating the http request").Error())
		return
	}

	client := &http.Client{
		Transport: transport,
	}

	resp, err := client.Do(r.Request)
	if err != nil {
		log.Println(errors.Wrap(err, "Error in http response").Error())
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
		headers: make(http.Header),
	}
}
