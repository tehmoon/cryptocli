package main

import (
	"time"
	"net/http"
	"github.com/tehmoon/errors"
	"github.com/spf13/pflag"
	"sync"
	"log"
	"net"
	"io"
)

func init() {
	MODULELIST.Register("http-server", "Create an http web webserver", NewHTTPServer)
}

type HTTPServer struct {
	addr string
	connectTimeout time.Duration
	writeTimeout time.Duration
	readHeaderTimeout time.Duration
	idleTimeout time.Duration
	readTimeout time.Duration
}

func (m *HTTPServer) SetFlagSet(fs *pflag.FlagSet, args []string) {
	fs.StringVar(&m.addr, "addr", "", "Listen on an address")
	fs.DurationVar(&m.connectTimeout, "connect-timeout", 30 * time.Second, "Max amount of time to wait for a potential connection when pipeline is closing")
	fs.DurationVar(&m.writeTimeout, "write-timeout", 15 * time.Second, "Set maximum duration before timing out writes of the response")
	fs.DurationVar(&m.readTimeout, "read-timeout", 15 * time.Second, "Set the maximum duration for reading the entire request, including the body.")
	fs.DurationVar(&m.readHeaderTimeout, "read-headers-timeout", 15 * time.Second, "Set the amount of time allowed to read request headers.")
	fs.DurationVar(&m.idleTimeout, "iddle-timeout", 5 * time.Second, "IdleTimeout is the maximum amount of time to wait for the next request when keep-alives are enabled")
}

func HTTPServerHandleResponse(m *HTTPServer, w http.ResponseWriter, req *http.Request, relay *HTTPServerRelayer) {
	outc, inc, wg := relay.Outc, relay.Inc, relay.Wg
	defer wg.Done()
	defer DrainChannel(inc, nil)
	defer close(outc)

	if req.Body != nil {
		err := ReadBytesSendMessages(req.Body, outc)
		if err != nil && err != io.EOF {
			err = errors.Wrap(err, "Error reading from http request")
			log.Println(err.Error())
			return
		}
	}

	w.WriteHeader(200)

	for payload := range inc {
		_, err := w.Write(payload)
		if err != nil {
			err = errors.Wrap(err, "Error writing to http connect")
			log.Println(err.Error())
			return
		}

		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
	}
}

func HTTPServerHandler(m *HTTPServer, relayer chan *HTTPServerRelayer, connc, donec, cancel chan struct{}) (func(w http.ResponseWriter, r *http.Request)) {
	return func(w http.ResponseWriter, req *http.Request) {
		donec <- struct{}{}

		defer func(donec chan struct{}) {
			<- donec
		}(donec)

		select {
			case <- cancel:
				w.WriteHeader(500)
				return
			case relay, opened := <- relayer:
				if ! opened {
					w.WriteHeader(500)
					return
				}

				HTTPServerHandleResponse(m, w, req, relay)
				return
			default:
		}

		select {
			case connc <- struct{}{}:
			case <- cancel:
				w.WriteHeader(500)
				return
		}

		select {
			case relay, opened := <- relayer:
				if ! opened {
					w.WriteHeader(500)
					return
				}

				HTTPServerHandleResponse(m, w, req, relay)
				return
			case <- cancel:
				w.WriteHeader(500)
				return
		}
	}
}

func (m *HTTPServer) Init(in, out chan *Message, global *GlobalFlags) (error) {
	if m.connectTimeout < 1 {
		return errors.Errorf("Flag %q cannot be negative or zero", "--connect-timeout")
	}

	if m.connectTimeout < 1 {
		return errors.Errorf("Flag %q cannot be negative or zero", "--connect-timeout")
	}

	if m.readTimeout < 1 {
		return errors.Errorf("Flag %q cannot be negative or zero", "--read-timeout")
	}

	if m.idleTimeout < 1 {
		return errors.Errorf("Flag %q cannot be negative or zero", "--idle-timeout")
	}

	if m.writeTimeout < 1 {
		return errors.Errorf("Flag %q cannot be negative or zero", "--write-timeout")
	}

	addr, err := net.ResolveTCPAddr("tcp", m.addr)
	if err != nil {
		return errors.Wrap(err, "Unable to resolve tcp address")
	}

	listener, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return errors.Wrap(err, "Unable to listen on tcp address")
	}

	go func() {
		wg := &sync.WaitGroup{}
		relayer := make(chan *HTTPServerRelayer)

		cancel := make(chan struct{})
		incs := make([]MessageChannel, 0)
		outcs := make([]MessageChannel, 0)
		donec := make(chan struct{}, global.MaxConcurrentStreams)
		connc := make(chan struct{})

		mux := http.NewServeMux()
		mux.HandleFunc("/", HTTPServerHandler(m, relayer, connc, donec, cancel))

		server := &http.Server{
			Handler: mux,
			ReadTimeout: m.readTimeout,
			IdleTimeout: m.idleTimeout,
			WriteTimeout: m.writeTimeout,
			ReadHeaderTimeout: m.readHeaderTimeout,
		}
		go server.Serve(listener)

		ticker := time.NewTicker(m.connectTimeout)

		LOOP: for {
			select {
				case <- ticker.C:
					ticker.Stop()
					close(cancel)
					wg.Wait()
					out <- &Message{
						Type: MessageTypeTerminate,
					}

					break LOOP

				case _, opened := <- connc:
					if ! opened {
						break LOOP
					}

					outc := make(MessageChannel)

					out <- &Message{
						Type: MessageTypeChannel,
						Interface: outc,
					}

					if len(incs) == 0 {
						outcs = append(outcs, outc)
						continue
					}

					wg.Add(1)

					inc := incs[0]
					incs = incs[1:]

					relayer <- &HTTPServerRelayer{
						Inc: inc,
						Outc: outc,
						Wg: wg,
					}

					if ! global.MultiStreams {
						close(cancel)
						wg.Wait()
						out <- &Message{Type: MessageTypeTerminate,}
						break LOOP
					}

				case message, opened := <- in:
					ticker.Stop()
					if ! opened {
						close(cancel)
						wg.Wait()
						out <- &Message{
							Type: MessageTypeTerminate,
						}
						break LOOP
					}

					switch message.Type {
						case MessageTypeTerminate:
							close(cancel)
							wg.Wait()
							out <- message
							break LOOP
						case MessageTypeChannel:
							inc, ok := message.Interface.(MessageChannel)
							if ok {
								if len(outcs) == 0 {
									incs = append(incs, inc)
									continue
								}

								wg.Add(1)
								outc := outcs[0]
								outcs = outcs[1:]

								relayer <- &HTTPServerRelayer{
									Inc: inc,
									Outc: outc,
									Wg: wg,
								}

								if ! global.MultiStreams {
									incs = append(incs, inc)
									close(cancel)
									wg.Wait()
									out <- &Message{Type: MessageTypeTerminate,}
									break LOOP
								}
							}
					}
			}
		}

		listener.Close()
		close(connc)

		for _, outc := range outcs {
			close(outc)
		}

		for _, inc := range incs {
			DrainChannel(inc, nil)
		}

		wg.Wait()
		close(relayer)
		close(donec)

		<- in
		close(out)
	}()

	return nil
}

type HTTPServerRelayer struct {
	Inc MessageChannel
	Outc MessageChannel
	Wg *sync.WaitGroup
}

func NewHTTPServer() (Module) {
	return &HTTPServer{}
}
