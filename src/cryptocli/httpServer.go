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
	"path"
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
	formUpload bool
	redirect string
	user string
	password string
	headers []string
}

var HTTPServerFormUploadPage = []byte(`
 <html>
 <title>Cryptocli http-server </title>
 <body>

 <form action="/" method="post" enctype="multipart/form-data">
 <label for="file">Filenames:</label>
 <input type="file" name="file" id="file">
 <input type="submit" name="submit" value="Submit">
 </form>

 </body>
 </html>
`)

func (m *HTTPServer) SetFlagSet(fs *pflag.FlagSet, args []string) {
	fs.StringVar(&m.addr, "addr", "", "Listen on an address")
	fs.BoolVar(&m.formUpload, "file-upload", false, "Serve a HTML page on GET / and a file upload endpoint on POST /")
	fs.DurationVar(&m.connectTimeout, "connect-timeout", 30 * time.Second, "Max amount of time to wait for a potential connection when pipeline is closing")
	fs.StringArrayVar(&m.headers, "header", make([]string, 0), "Set header in the form of \"header: value\"")
	fs.DurationVar(&m.writeTimeout, "write-timeout", 15 * time.Second, "Set maximum duration before timing out writes of the response")
	fs.DurationVar(&m.readTimeout, "read-timeout", 15 * time.Second, "Set the maximum duration for reading the entire request, including the body.")
	fs.DurationVar(&m.readHeaderTimeout, "read-headers-timeout", 15 * time.Second, "Set the amount of time allowed to read request headers.")
	fs.DurationVar(&m.idleTimeout, "iddle-timeout", 5 * time.Second, "IdleTimeout is the maximum amount of time to wait for the next request when keep-alives are enabled")
	fs.StringVar(&m.user, "user", "", "Specify the required user for basic auth")
	fs.StringVar(&m.password, "password", "", "Specify the required password for basic auth")
	fs.StringVar(&m.redirect, "redirect-to", "", "Redirect the request to where the download can begin")
}

func HTTPServerHandleResponse(m *HTTPServer, w http.ResponseWriter, req *http.Request, relay *HTTPServerRelayer) {
	outc, inc, wg := relay.Outc, relay.Inc, relay.Wg
	defer wg.Done()
	defer DrainChannel(inc, nil)
	defer close(outc)

	if m.formUpload {
		file, _, err := req.FormFile("file")
		if err != nil {
			err = errors.Wrap(err, "Error reading from form")
			log.Println(err.Error())
			w.WriteHeader(500)
			return
		}

		err = ReadBytesSendMessages(file, outc)
		if err != nil && err != io.EOF {
			err = errors.Wrap(err, "Error reading form file")
			log.Println(err.Error())
			w.WriteHeader(500)
			return
		}

		w.WriteHeader(200)
		w.Write([]byte(`uploaded`))
		return
	}

	w.Header().Add("Content-Type", "application/octet-stream")
	w.Header().Add("Content-Disposition", "attachment;")

	if req.Body != nil {
		err := ReadBytesSendMessages(req.Body, outc)
		if err != nil && err != io.EOF {
			err = errors.Wrap(err, "Error reading from http request")
			log.Println(err.Error())
			return
		}
	}

	w.WriteHeader(200)

	for {
		select {
			case payload, opened := <- inc:
				if ! opened {
					return
				}

				_, err := w.Write(payload)
				if err != nil {
					err = errors.Wrap(err, "Error writing to http connect")
					log.Println(err.Error())
					return
				}

				if f, ok := w.(http.Flusher); ok {
					f.Flush()
				}
			case <- req.Context().Done():
				log.Println("Connection got closed")
				return
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

type HTTPServerHandle struct {
	cb http.HandlerFunc
	m *HTTPServer
	headers http.Header
}

func (h *HTTPServerHandle) ServeHTTP(w http.ResponseWriter, r *http.Request) {
		headers := w.Header()
		for k, v := range h.headers {
			headers[k] = v
		}

		if h.m.user != "" && h.m.password != "" {
			user, password, found := r.BasicAuth()
			if ! found {
				w.Header().Add("WWW-Authenticate", "Basic realm=cryptocli")
				w.WriteHeader(401)
				return
			}

			if h.m.user != user || h.m.password != password {
				w.Header().Add("WWW-Authenticate", "Basic realm=cryptocli")
				w.WriteHeader(401)
				w.Write([]byte(`Unauthorized access`))
				return
			}
		}

		switch true {
			case h.m.redirect != "":
				switch r.Method {
					case "GET":
						endpoint := path.Join("/", h.m.redirect)
						if r.URL.Path == endpoint {
							h.cb(w, r)
							return
						}

						w.Header().Add("Location", endpoint)
						w.WriteHeader(302)
						return
					default:
						w.WriteHeader(404)
						return
				}
			case h.m.formUpload:
				switch r.Method {
					case "GET":
						if r.URL.Path == "/" {
							w.Header().Add("Content-Type", "text/html")
							w.Write(HTTPServerFormUploadPage)
							return
						}

						w.WriteHeader(404)
						return
					case "POST":
						if r.URL.Path == "/" {
							h.cb(w, r)
							return
						}

						w.WriteHeader(404)
						return
					default:
						w.WriteHeader(404)
						return
				}
			default:
				switch r.Method {
					case "GET":
						h.cb(w, r)
						return
					default:
						w.WriteHeader(404)
						return
				}
		}
}

func NewHTTPServerHandle(m *HTTPServer, cb http.HandlerFunc) (http.Handler) {
	return &HTTPServerHandle{
		cb: cb,
		m: m,
		headers: ParseHTTPHeaders(m.headers),
	}
}

func (m *HTTPServer) Init(in, out chan *Message, global *GlobalFlags) (error) {
	if m.user != "" && m.password == "" {
		return errors.Errorf("Flag %q is required when %q is set", "--password", "--user")
	}

	if m.user == "" && m.password != "" {
		return errors.Errorf("Flag %q is required when %q is set", "--user", "--password")
	}

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

	if m.redirect != "" && m.formUpload {
		return errors.Errorf("Flag %q and flag %q are mutually exclusive", "--redirect-to", "--file-upload")
	}

	if m.addr == "" {
		return errors.Errorf("Missing flag %q", "--addr")
	}

	addr, err := net.ResolveTCPAddr("tcp", m.addr)
	if err != nil {
		return errors.Wrap(err, "Unable to resolve tcp address")
	}

	listener, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return errors.Wrap(err, "Unable to listen on tcp address")
	}

	log.Printf("HTTP Server is listening on address: %q\n", addr.String())

	go func() {
		wg := &sync.WaitGroup{}
		relayer := make(chan *HTTPServerRelayer)

		cancel := make(chan struct{})
		incs := make([]MessageChannel, 0)
		outcs := make([]MessageChannel, 0)
		donec := make(chan struct{}, global.MaxConcurrentStreams)
		connc := make(chan struct{})

		mux := http.NewServeMux()
		mux.Handle("/", NewHTTPServerHandle(m, HTTPServerHandler(m, relayer, connc, donec, cancel)))

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
