package main

import (
	"time"
	"net/http"
	"github.com/tehmoon/errors"
	"github.com/gorilla/websocket"
	"github.com/spf13/pflag"
	"sync"
	"log"
	"net"
)

func init() {
	MODULELIST.Register("websocket-server", "Create an http websocket server", NewWebsocketServer)
}

type WebsocketServer struct {
	addr string
	connectTimeout time.Duration
	readHeaderTimeout time.Duration
	readTimeout time.Duration
	closeTimeout time.Duration
	mode int
	text bool
	headers []string
}

func (m *WebsocketServer) SetFlagSet(fs *pflag.FlagSet, args []string) {
	fs.StringVar(&m.addr, "addr", "", "Listen on an address")
	fs.StringArrayVar(&m.headers, "header", make([]string, 0), "Set header in the form of \"header: value\"")
	fs.DurationVar(&m.connectTimeout, "connect-timeout", 30 * time.Second, "Max amount of time to wait for a potential connection when pipeline is closing")
	fs.DurationVar(&m.closeTimeout, "close-timeout", 15 * time.Second, "Timeout to wait for after sending the closure message")
	fs.DurationVar(&m.readTimeout, "read-timeout", 15 * time.Second, "Read timeout for the websocket connection")
	fs.DurationVar(&m.readHeaderTimeout, "read-headers-timeout", 15 * time.Second, "Set the amount of time allowed to read request headers.")
	fs.BoolVar(&m.text, "text", false, "Set the websocket message's metadata to text")
}

func websocketServerUpgrade(m *WebsocketServer, w http.ResponseWriter, req *http.Request, relay *WebsocketServerRelayer) {
	outc, inc, wg := relay.Outc, relay.Inc, relay.Wg
	defer wg.Done()

	upgrader := &websocket.Upgrader{}
	upgrader.CheckOrigin = func(r *http.Request) bool {
		return true
	}

	conn, err := upgrader.Upgrade(w, req, ParseHTTPHeaders(m.headers))
	if err != nil {
		err = errors.Wrap(err, "Fail to upgrade to websocket")
		log.Println(err.Error())
		close(outc)
		DrainChannel(relay.Inc, nil)
		return
	}

	conn.SetPingHandler(func(message string) error {
		conn.SetReadDeadline(time.Now().Add(m.readTimeout))
		return conn.WriteMessage(websocket.PongMessage, []byte(`hello`))
	})

	doneReadC := make(chan struct{})
	syn := &sync.WaitGroup{}
	syn.Add(2)

	go func(conn *websocket.Conn, inc MessageChannel, mode int, timeout time.Duration, doneReadC chan struct{}, wg *sync.WaitGroup) {
		defer wg.Done()
		defer conn.Close()

		for payload := range inc {
			err := conn.WriteMessage(mode, payload)
			if err != nil {
				err = errors.Wrap(err, "Error writing to websocket connection")
				log.Println(err.Error())
				break
			}
		}

		DrainChannel(inc, nil)

		closer := websocket.FormatCloseMessage(websocket.CloseNormalClosure, "")
		conn.WriteMessage(websocket.CloseMessage, closer)

		select {
			case <- time.After(timeout):
			case <- doneReadC:
		}

	}(conn, inc, m.mode, m.closeTimeout, doneReadC, syn)

	go func(conn *websocket.Conn, outc MessageChannel, timeout time.Duration, doneReadC chan struct{}, wg *sync.WaitGroup) {
		defer wg.Done()
		defer close(outc)
		defer close(doneReadC)

		conn.SetReadDeadline(time.Now().Add(timeout))

		for {
			t, payload, err := conn.ReadMessage()
			if err != nil {
				err = errors.Wrap(err, "Error reading message from websocket")
				log.Println(err.Error())
				return
			}

			switch t {
				case websocket.CloseMessage:
					return
			}

			outc <- payload
			conn.SetReadDeadline(time.Now().Add(timeout))
		}
	}(conn, outc, m.readTimeout, doneReadC, syn)

	syn.Wait()
}

func websocketServerHandle(m *WebsocketServer, relayer chan *WebsocketServerRelayer, connc, donec, cancel chan struct{}) (func(w http.ResponseWriter, r *http.Request)) {
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

				websocketServerUpgrade(m, w, req, relay)
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

				websocketServerUpgrade(m, w, req, relay)
				return
			case <- cancel:
				w.WriteHeader(500)
				return
		}
	}
}

func (m *WebsocketServer) Init(in, out chan *Message, global *GlobalFlags) (error) {
	if m.connectTimeout < 1 {
		return errors.Errorf("Flag %q cannot be negative or zero", "--connect-timeout")
	}

	if m.closeTimeout < 1 {
		return errors.Errorf("Flag %q cannot be negative or zero", "--close-timeout")
	}

	if m.readHeaderTimeout < 1 {
		return errors.Errorf("Flag %q cannot be negative or zero", "--read-header-timeout")
	}

	if m.readTimeout < 1 {
		return errors.Errorf("Flag %q cannot be negative or zero", "--read-timeout")
	}

	if m.text {
		m.mode = websocket.TextMessage
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
		relayer := make(chan *WebsocketServerRelayer)

		connc := make(chan struct{})
		cancel := make(chan struct{})

		donec := make(chan struct{}, global.MaxConcurrentStreams)

		mux := http.NewServeMux()
		mux.HandleFunc("/", websocketServerHandle(m, relayer, connc, donec, cancel))

		server := &http.Server{
			Handler: mux,
			ReadHeaderTimeout: m.readHeaderTimeout,
		}
		go server.Serve(listener)

		incs := make([]MessageChannel, 0)
		outcs := make([]MessageChannel, 0)

		ticker := time.NewTicker(m.connectTimeout)

		LOOP: for {
			select {
				case <- ticker.C:
					ticker.Stop()
					close(cancel)
					wg.Wait()
					log.Println("Connect timeout reached, nobody connected and no messages from inputs were received")
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

					relayer <- &WebsocketServerRelayer{
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

								relayer <- &WebsocketServerRelayer{
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

type WebsocketServerRelayer struct {
	Inc MessageChannel
	Outc MessageChannel
	Wg *sync.WaitGroup
}

func NewWebsocketServer() (Module) {
	return &WebsocketServer{
		mode: websocket.BinaryMessage,
	}
}
