package main

import (
	"io"
	"time"
	"net/http"
	"github.com/tehmoon/errors"
	"github.com/spf13/pflag"
	"sync"
	"log"
	"crypto/tls"
	"text/template"
	"bytes"
)

func init() {
	MODULELIST.Register("http", "Makes HTTP requests", NewHTTP)
}

type HTTP struct {
	url string
	insecure bool
	readTimeout time.Duration
	headers []string
	method string
	data bool
	user string
	password string
	showClientHeaders bool
	showServerHeaders bool
	tplUrl *template.Template
}

func (m *HTTP) SetFlagSet(fs *pflag.FlagSet, args []string) {
	fs.BoolVar(&m.showClientHeaders, "show-client-headers", false, "Show client headers in the logs")
	fs.BoolVar(&m.showServerHeaders, "show-server-headers", false, "Show server headers in the logs")
	fs.StringVar(&m.url, "url", "", "HTTP server to connect to")
	fs.BoolVar(&m.insecure, "insecure", false, "Don't verify the tls certificate chain")
	fs.DurationVar(&m.readTimeout, "read-timeout", 15 * time.Second, "Read timeout for the tcp connection")
	fs.StringVar(&m.method, "method", "GET", "HTTP Verb")
	fs.StringArrayVar(&m.headers, "header", make([]string, 0), "Set header in the form of \"header: value\"")
	fs.BoolVar(&m.data, "data", false, "Read data from the stream and send it before reading the response")
	fs.StringVar(&m.user, "user", "", "Specify the required user for basic auth")
	fs.StringVar(&m.password, "password", "", "Specify the required password for basic auth")
}

func (m *HTTP) Init(in, out chan *Message, global *GlobalFlags) (err error) {
	if m.readTimeout <= 0 {
		return errors.Errorf("Flag %q has to be greater that 0", "--read-timeout")
	}

	if m.user != "" && m.password == "" {
		return errors.Errorf("Flag %q is required when %q is set", "--password", "--user")
	}

	if m.user == "" && m.password != "" {
		return errors.Errorf("Flag %q is required when %q is set", "--user", "--password")
	}

	m.tplUrl, err = template.New("root").Parse(m.url)
	if err != nil {
		return errors.Wrap(err, "Error parsing template for \"--url\" flag")
	}

	go func(in, out chan *Message) {
		wg := &sync.WaitGroup{}

		init := false
		mc := NewMessageChannel()

		out <- &Message{
			Type: MessageTypeChannel,
			Interface: mc.Callback,
		}

		LOOP: for message := range in {
			switch message.Type {
				case MessageTypeTerminate:
					if ! init {
						close(mc.Channel)
					}
					wg.Wait()
					out <- message
					break LOOP
				case MessageTypeChannel:
					cb, ok := message.Interface.(MessageChannelFunc)
					if ok {
						if ! init {
							init = true
						} else {
							mc = NewMessageChannel()

							out <- &Message{
								Type: MessageTypeChannel,
								Interface: mc.Callback,
							}
						}

						wg.Add(1)
						// TODO: remove m.data
						go httpStartHandler(m, cb, mc, m.data, wg)

						if ! global.MultiStreams {
							if ! init {
								close(mc.Channel)
							}
							wg.Wait()
							out <- &Message{Type: MessageTypeTerminate,}
							break LOOP
						}
					}

			}
		}

		wg.Wait()
		// Last message will signal the closing of the channel
		<- in
		close(out)
	}(in, out)

	return nil
}

func httpStartHandler(m *HTTP, cb MessageChannelFunc, mc *MessageChannel, data bool, wg *sync.WaitGroup) {
	defer wg.Done()

	reader, writer := io.Pipe()
	cancel := make(chan error)
	goahead := &sync.WaitGroup{}

	mc.Start(nil)
	metadata, inc := cb()

	buff := bytes.NewBuffer(make([]byte, 0))
	err := m.tplUrl.Execute(buff, metadata)
	if err != nil {
		err = errors.Wrap(err, "Error executing template url")
		log.Println(err.Error())
		close(mc.Channel)
		DrainChannel(inc, nil)
		return
	}

	url := string(buff.Bytes()[:])
	buff.Reset()

	outc := mc.Channel
	defer close(outc)

	wg.Add(1)
	goahead.Add(1)
	go func(inc chan []byte, writer *io.PipeWriter, wg *sync.WaitGroup, goahead *sync.WaitGroup, cancel chan error) {
		defer wg.Done()
		defer DrainChannel(inc, nil)
		defer func(cancel chan error) {
			for err := range cancel {
				log.Println(err.Error())
			}
		}(cancel)

		if ! data {
			goahead.Done()
			writer.Close()
			return
		}

		signaled := false

		LOOP: for {
			select {
				case err, opened := <- cancel:
					if ! opened {
						break LOOP
					}

					writer.CloseWithError(err)
					return
				case payload, opened := <- inc:
					if ! signaled {
						go goahead.Done()
						signaled = true
					}

					if ! opened {
						break LOOP
					}

					_, err := writer.Write(payload)
					if err != nil {
						err = errors.Wrap(err, "Errror writing to pipe")
						writer.CloseWithError(err)
						return
					}
			}
		}

		writer.Close()
	}(inc, writer, wg, goahead, cancel)

	goahead.Wait()

	client := &http.Client{
		Timeout: m.readTimeout,
		Transport: httpCreateTransport(m.insecure),
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) > 0 {
				return errors.New("Unsupported redirect")
			}

			return nil
		},
	}

	log.Printf("Opening connection to %q\n", url)
	req, err := http.NewRequest(m.method, url, reader)
	if err != nil {
		err = errors.Wrap(err, "Error creating new request")
		cancel <- err
		close(cancel)
		return
	}

	headers := ParseHTTPHeaders(m.headers)
	req.Host = headers.Get("host")
	req.Header = headers

	if m.user != "" && m.password != "" {
		req.SetBasicAuth(m.user, m.password)
	}

	if m.showClientHeaders {
		ShowHTTPClientHeaders(req.Header)
	}

	resp, err := client.Do(req)
	if m.showServerHeaders {
		ShowHTTPServerHeaders(resp.Header)
	}
	if err != nil {
		err = errors.Wrap(err, "Error sending request")
		cancel <- err
		close(cancel)
		return
	}

	log.Printf("Response status is %q\n", resp.Status)

	close(cancel)

	if resp.Body == nil {
		return
	}

	err = ReadBytesSendMessages(resp.Body, outc)
	if err != nil {
		err = errors.Wrap(err, "Error reading http body")
		log.Println(err.Error())
		return
	}

	resp.Body.Close()
}

func NewHTTP() (Module) {
	return &HTTP{}
}

func httpCreateTransport(insecure bool) (http.RoundTripper) {
	var (
		tr = &http.Transport{}
		rt = http.DefaultTransport
	)

	transport, ok := http.DefaultTransport.(*http.Transport)
	if ok {
		*tr = *transport

		tr.TLSClientConfig = httpCreateTLSConfig(insecure)

		rt = tr
	}

	return rt
}

func httpCreateTLSConfig(insecure bool) (config *tls.Config) {
	return &tls.Config{
		InsecureSkipVerify: insecure,
	}
}
