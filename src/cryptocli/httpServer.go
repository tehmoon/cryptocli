package main

import (
	"time"
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
	Waitc chan struct{}
	Relayc chan *Message
	Init bool
	Sync *sync.WaitGroup

	sync.Mutex
}

type HTTPServer struct {
	in chan *Message
	out chan *Message
	req *http.Request
	addr string
	ln net.Listener
	connectTimeout time.Duration
	sync *sync.WaitGroup
}

func (m *HTTPServer) SetFlagSet(fs *pflag.FlagSet) {
	fs.StringVar(&m.addr, "addr", "", "Listen on an address")
	fs.DurationVar(&m.connectTimeout, "connect-timeout", 30 * time.Second, "Max amount of time to wait for a potential connection when pipeline is closing")
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

func httpServerHandleOut(out chan *Message, body io.ReadCloser, wait *sync.WaitGroup) {
	defer close(out)
	defer wait.Done()
	defer body.Close()

	err := ReadBytesSendMessages(body, out)
	if err != nil {
		log.Println(errors.Wrap(err, "Error reading tcp connection in http-server"))
	}
}

func httpServerHandle(in, out chan *Message, options *HTTPServerHandleOptions) (func(w http.ResponseWriter, r *http.Request)) {
	return func(w http.ResponseWriter, r *http.Request) {
		options.Lock()

		if options.Init {
			w.WriteHeader(500)
			options.Unlock()
			return
		}

		defer options.Sync.Done()

		options.Init = true
		options.Unlock()

		options.Waitc <- struct{}{}

		// done synchronises the write part of the 2 go routines
		done := &sync.WaitGroup{}
		done.Add(1)

		// wait tells the write part to start writing when read is done
		// this is because the http specs says we should read the body
		// before writing
		wait := &sync.WaitGroup{}
		wait.Add(1)

		go httpServerHandleIn(in, w, wait, done)
		go httpServerHandleOut(out, r.Body, wait)

		done.Wait()
	}
}

func (m *HTTPServer) Init(global *GlobalFlags) (error) {
	if m.connectTimeout < 1 {
		return errors.Errorf("Flag %q cannot be negative or zero", "--connect-timeout")
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

func (m *HTTPServer) Start() {
	waitc := make(chan struct{})
	relayc := make(chan *Message, 1)

	m.sync.Add(1)

	options := &HTTPServerHandleOptions{
		Init: false,
		Waitc: waitc,
		Sync: m.sync,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", httpServerHandle(relayc, m.out, options))

	server := &http.Server{
		Handler: mux,
	}

	startRelay := sync.WaitGroup{}
	startRelay.Add(1)

	go server.Serve(m.ln)
	go func() {
		select {
			case <- waitc:
				startRelay.Done()
			case <- time.NewTicker(m.connectTimeout).C:
				log.Println("Connect timeout reached, nobody connected and no inputs were sent")
				options.Init = true
				close(m.out)
				m.sync.Done()
				return
			case message, closed := <- m.in:
				select {
					case <- time.NewTicker(m.connectTimeout).C:
						log.Println("Connect timeout reached, let's hope somebody's connected")
					case <- waitc:
				}

				options.Lock()
				if closed && ! options.Init {
					options.Init = true
					options.Unlock()

					log.Println("Pipeline is shutting down and nobody connected")
					close(m.out)
					m.sync.Done()
					return
				}

				options.Unlock()

				relayc <- message
				startRelay.Done()
		}
	}()

	go func() {
		startRelay.Wait()

		for message := range m.in {
			relayc <- message
		}

		close(relayc)
	}()
}

func (m HTTPServer) Wait() {
	m.sync.Wait()

	for range m.in {}
}

func NewHTTPServer() (Module) {
	return &HTTPServer{
		sync: &sync.WaitGroup{},
	}
}
