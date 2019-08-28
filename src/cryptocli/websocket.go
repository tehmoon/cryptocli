package main

import (
	"net/http"
	"github.com/gorilla/websocket"
	"github.com/tehmoon/errors"
	"github.com/spf13/pflag"
	"sync"
	"log"
	"io"
	"time"
	"crypto/tls"
	"strings"
)

func init() {
	MODULELIST.Register("websocket", "Connects to a websocket webserver", NewWebsocket)
}

type Websocket struct {
	in chan *Message
	out chan *Message
	sync *sync.WaitGroup
	url string
	reader io.Reader
	writer io.WriteCloser
	req *http.Request
	insecure bool
	conn *websocket.Conn
	closeTimeout time.Duration
	binary bool
	mode int
	rawHeaders []string
	headers http.Header
}

func (m *Websocket) SetFlagSet(fs *pflag.FlagSet) {
	fs.StringVar(&m.url, "url", "", "HTTP url to query")
	fs.BoolVar(&m.insecure, "insecure", false, "Don't valid the TLS certificate chain")
	fs.DurationVar(&m.closeTimeout, "close-timeout", 5 * time.Second, "Duration to wait to read the close message")
	fs.BoolVar(&m.binary, "binary", false, "Set the websocket message's metadata to binary")
	fs.StringArrayVar(&m.rawHeaders, "header", make([]string, 0), "Send HTTP Headers")
}

func (m *Websocket) In(in chan *Message) (chan *Message) {
	m.in = in

	return in
}

func (m *Websocket) Out(out chan *Message) (chan *Message) {
	m.out = out

	return out
}

func (m *Websocket) Init(global *GlobalFlags) (err error) {
	if m.closeTimeout < 0 {
		return errors.Errorf("Duration for flag %q cannot be negative", "--close-timeout")
	}

	dialer := &websocket.Dialer{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: m.insecure,
		},
	}

	if m.binary {
		m.mode = websocket.BinaryMessage
	}

	for _, rawHeader := range m.rawHeaders {
		header := strings.Split(rawHeader, ":")

		key := header[0]
		value := ""
		if len(header) > 1 {
			value = strings.Join(header[1:], ":")
		}

		m.headers.Set(key, value)
	}

	m.conn, _, err = dialer.Dial(m.url, m.headers)
	if err != nil {
		err = errors.Wrap(err, "Error dialing the websocket connection")

		return err
	}

	return nil
}

func (m Websocket) Start() {
	m.sync.Add(2)

	donec := make(chan struct{})

	go func() {
		defer func() {
			closer := websocket.FormatCloseMessage(websocket.CloseNormalClosure, "")
			m.conn.WriteMessage(websocket.CloseMessage, closer)

			select {
				case <- time.After(m.closeTimeout):
				case <- donec:
			}
		}()

		defer m.sync.Done()

		for message := range m.in {
			err := m.conn.WriteMessage(m.mode, message.Payload)
			if err != nil {
				err = errors.Wrap(err, "Error writing message to websocket")
				log.Println(err.Error())

				return
			}
		}
	}()

	go func() {
		defer close(donec)
		defer close(m.out)
		defer m.sync.Done()

		for {
			_, payload, err := m.conn.ReadMessage()
			if err != nil {
				err = errors.Wrap(err, "Error reading message from websocket")
				log.Println(err.Error())

				return
			}

			SendMessage(payload, m.out)
		}
	}()
}

func (m Websocket) Wait() {
	m.sync.Wait()

	for range m.in {}
}


type WebsocketOptions struct {
	Insecure bool
}

func NewWebsocket() (Module) {
	return &Websocket{
		sync: &sync.WaitGroup{},
		mode: websocket.TextMessage,
		headers: make(http.Header),
	}
}
