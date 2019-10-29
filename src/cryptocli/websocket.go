package main

import (
	"time"
	"github.com/gorilla/websocket"
	"github.com/tehmoon/errors"
	"github.com/spf13/pflag"
	"sync"
	"crypto/tls"
	"log"
)

func init() {
	MODULELIST.Register("websocket", "Connect using the websocket protocol", NewWebsocket)
}

type Websocket struct {
	url string
	insecure bool
	readTimeout time.Duration
	headers []string
	closeTimeout time.Duration
	pingInterval time.Duration
	text bool
	mode int
}

func (m *Websocket) SetFlagSet(fs *pflag.FlagSet, args []string) {
	fs.StringVar(&m.url, "url", "", "Websocket server to connect to")
	fs.BoolVar(&m.insecure, "insecure", false, "Don't verify the tls certificate chain")
	fs.DurationVar(&m.readTimeout, "read-timeout", 15 * time.Second, "Read timeout for the websocket connection")
	fs.DurationVar(&m.closeTimeout, "close-timeout", 15 * time.Second, "Timeout to wait for after sending the closure message")
	fs.DurationVar(&m.pingInterval, "ping-interval", 30 * time.Second, "Interval of time between ping websocket messages")
	fs.StringArrayVar(&m.headers, "header", make([]string, 0), "Set header in the form of \"header: value\"")
	fs.BoolVar(&m.text, "text", false, "Set the websocket message's metadata to text")
}

func (m *Websocket) Init(in, out chan *Message, global *GlobalFlags) (error) {
	if m.readTimeout <= 0 {
		return errors.Errorf("Flag %q has to be greater that 0", "--read-timeout")
	}

	if m.closeTimeout <= 0 {
		return errors.Errorf("Flag %q has to be greater that 0", "--close-timeout")
	}

	if m.pingInterval <= 0 {
		return errors.Errorf("Flag %q has to be greater that 0", "--ping-interval")
	}

	if m.text {
		m.mode = websocket.TextMessage
	}

	go func(in, out chan *Message) {
		wg := &sync.WaitGroup{}

		init := false
		outc := make(MessageChannel)

		out <- &Message{
			Type: MessageTypeChannel,
			Interface: outc,
		}

		LOOP: for message := range in {
			switch message.Type {
				case MessageTypeTerminate:
					if ! init {
						close(outc)
					}
					wg.Wait()
					out <- message
					break LOOP
				case MessageTypeChannel:
					inc, ok := message.Interface.(MessageChannel)
					if ok {
						if ! init {
							init = true
						} else {
							outc = make(MessageChannel)

							out <- &Message{
								Type: MessageTypeChannel,
								Interface: outc,
							}
						}
						dialer := &websocket.Dialer{
							TLSClientConfig: &tls.Config{
								InsecureSkipVerify: m.insecure,
							},
						}

						wg.Add(1)
						go websocketStartHandler(m, dialer, inc, outc, wg)

						if ! global.MultiStreams {
							if ! init {
								close(outc)
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

func websocketStartHandler(m *Websocket, dialer *websocket.Dialer, inc, outc MessageChannel, wg *sync.WaitGroup) {
	defer wg.Done()

	conn, _, err := dialer.Dial(m.url, ParseHTTPHeaders(m.headers))
	if err != nil {
		err = errors.Wrap(err, "Error dialing websocket connection")
		log.Println(err.Error())
		close(outc)
		DrainChannel(inc, nil)
		return
	}

	donec := make(chan struct{})

	conn.SetPongHandler(func(message string) error {
		conn.SetReadDeadline(time.Now().Add(m.readTimeout))
		return nil
	})

	wg.Add(1)
	go func(conn *websocket.Conn, inc MessageChannel, timeout, pingInterval time.Duration, wg *sync.WaitGroup, donec chan struct{}) {
		defer wg.Done()
		defer DrainChannel(inc, nil)
		defer func() {
			closer := websocket.FormatCloseMessage(websocket.CloseNormalClosure, "")
			conn.WriteMessage(websocket.CloseMessage, closer)

			select {
				case <- time.After(timeout):
				case <- donec:
			}

			conn.Close()
		}()

		ticker := time.NewTicker(pingInterval)

		LOOP: for {
			select {
				case payload, opened := <- inc:
					if ! opened {
						ticker.Stop()
						break LOOP
					}

					err := conn.WriteMessage(m.mode, payload)
					if err != nil {
						err = errors.Wrap(err, "Error writing message to websocket")
						log.Println(err.Error())

						ticker.Stop()
						break LOOP
					}
				case <- ticker.C:
					err := conn.WriteMessage(websocket.PingMessage, []byte(`test`))
					if err != nil {
						err = errors.Wrap(err, "err sending ping message")
						log.Println(err.Error())
						break LOOP
					}
			}
		}

	}(conn, inc, m.closeTimeout, m.pingInterval, wg, donec)

	wg.Add(1)
	go func(conn *websocket.Conn, outc MessageChannel, timeout time.Duration, wg *sync.WaitGroup, donec chan struct{}) {
		defer wg.Done()
		defer close(outc)
		defer close(donec)

		for {
			conn.SetReadDeadline(time.Now().Add(timeout))
			t, payload, err := conn.ReadMessage()
			if err != nil {
				err = errors.Wrap(err, "Error reading message from websocket")
				log.Println(err.Error())
				return
			}

			switch t {
				case websocket.PingMessage:
					err = conn.WriteMessage(websocket.PongMessage, make([]byte, 0))
					if err != nil {
						err = errors.Wrap(err, "Error sending pong message to websocket")
						log.Println(err.Error())
						return
					}
				case websocket.CloseMessage:
					return
			}

			outc <- payload
		}
	}(conn, outc, m.readTimeout, wg, donec)
}

func NewWebsocket() (Module) {
	return &Websocket{
		mode: websocket.BinaryMessage,
	}
}
