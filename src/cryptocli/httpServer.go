package main

import (
	"io"
	"net/http"
	"github.com/tehmoon/errors"
	"github.com/spf13/pflag"
	"sync"
	"log"
	"net"
)

func init() {
	MODULELIST.Register("http-server", "Create an http web webserver", NewHTTPServer)
}

type HTTPServerHandleOptions struct {
	Sync *sync.WaitGroup
	Line bool
	Mutex *sync.Mutex
	Init bool
}

type HTTPServer struct {
	in chan *Message
	out chan *Message
	sync *sync.WaitGroup
	req *http.Request
	addr string
	ln net.Listener
	line bool
	mutex *sync.Mutex
	init bool
}

func (m *HTTPServer) SetFlagSet(fs *pflag.FlagSet) {
	fs.StringVar(&m.addr, "addr", "", "Listen on an address")
	fs.BoolVar(&m.line, "line", false, "Read lines from the connection")
}

func (m *HTTPServer) In(in chan *Message) (chan *Message) {
	m.in = in

	return in
}

func (m *HTTPServer) Out(out chan *Message) (chan *Message) {
	m.out = out

	return out
}

func httpServerHandleIn(in chan *Message, w http.ResponseWriter, wait, done *sync.WaitGroup) {
	defer done.Done()
	wait.Wait()

	w.WriteHeader(200)

	for message := range in {
		_, err := w.Write(message.Payload)
		if err != nil {
			log.Println(errors.Wrap(err, "Error writing tcp connection in http-server"))
			return
		}

		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
	}
}

func httpServerHandleOut(out chan *Message, body io.ReadCloser, line bool, wait *sync.WaitGroup) {
	defer close(out)
	defer wait.Done()
	defer body.Close()

	var err error

	cb := func(payload []byte) {
		SendMessage(payload, out)
	}

	if line {
		err = ReadDelimStep(body, '\n', cb)
	} else {
		err = ReadBytesStep(body, cb)
	}

	if err != nil {
		log.Println(errors.Wrap(err, "Error reading tcp connection in http-server"))
	}
}

func httpServerHandle(in, out chan *Message, options *HTTPServerHandleOptions) (func(w http.ResponseWriter, r *http.Request)) {
	return func(w http.ResponseWriter, r *http.Request) {
		options.Mutex.Lock()
		defer options.Mutex.Unlock()

		if ! options.Init {
			defer options.Sync.Done()
			options.Init = true

			// done synchronises the write part of the 2 go routines
			done := &sync.WaitGroup{}
			done.Add(1)

			// wait tells the write part to start writing when read is done
			// this is because the http specs says we should read the body
			// before writing
			wait := &sync.WaitGroup{}
			wait.Add(1)

			go httpServerHandleIn(in, w, wait, done)
			go httpServerHandleOut(out, r.Body, options.Line, wait)

			done.Wait()

			return
		}
	}
}

func (m *HTTPServer) Init(global *GlobalFlags) (error) {
	if global.Line {
		m.line = true
	}

	addr, err := net.ResolveTCPAddr("tcp", m.addr)
	if err != nil {
		return errors.Wrap(err, "Unable to resolve tcp address")
	}

	ln, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return errors.Wrap(err, "Unable to listen on tcp address")
	}

	m.ln = ln

	return nil
}

func (m HTTPServer) Start() {
	mux := http.NewServeMux()
	mux.HandleFunc("/", httpServerHandle(m.in, m.out, &HTTPServerHandleOptions{
		Line: m.line,
		Sync: m.sync,
		Mutex: m.mutex,
		Init: m.init,
	}))

	server := &http.Server{
		Handler: mux,
	}

	m.sync.Add(1)

	go server.Serve(m.ln)
}

func (m HTTPServer) Wait() {
	m.sync.Wait()

	for range m.in {}
}

func NewHTTPServer() (Module) {
	return &HTTPServer{
		sync: &sync.WaitGroup{},
		mutex: &sync.Mutex{},
		init: false,
	}
}
