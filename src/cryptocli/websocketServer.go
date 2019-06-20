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
	MODULELIST.Register("websocket-server", "Create an websocket webserver", NewWebsocketServer)
}

type WebsocketServerHandleOptions struct {
	Sync *sync.WaitGroup
	Mutex *sync.Mutex
	Init bool
	Upgrader websocket.Upgrader
	Waitc chan struct{}
	CloseTimeout time.Duration
}

type WebsocketServer struct {
	in chan *Message
	out chan *Message
	sync *sync.WaitGroup
	req *http.Request
	addr string
	ln net.Listener
	mutex *sync.Mutex
	init bool
	closeTimeout time.Duration
	connectTimeout time.Duration
}

func (m *WebsocketServer) SetFlagSet(fs *pflag.FlagSet) {
	fs.StringVar(&m.addr, "addr", "", "Listen on an address")
	fs.DurationVar(&m.closeTimeout, "close-timeout", 5 * time.Second, "Duration to wait to read the close message")
	fs.DurationVar(&m.connectTimeout, "connect-timeout", 15 * time.Second, "Duration to wait for a websocket connection")
}

func (m *WebsocketServer) In(in chan *Message) (chan *Message) {
	m.in = in

	return in
}

func (m *WebsocketServer) Out(out chan *Message) (chan *Message) {
	m.out = out

	return out
}

func websocketServerHandle(in, out chan *Message, options *WebsocketServerHandleOptions) (func(w http.ResponseWriter, r *http.Request)) {
	return func(w http.ResponseWriter, r *http.Request) {
		options.Mutex.Lock()

		if ! options.Init {
			c, err := options.Upgrader.Upgrade(w, r, nil)
			if err != nil {
				err = errors.Wrap(err, "Fail to upgrade to websocket")
				log.Println(err.Error())
				options.Mutex.Unlock()
				return
			}

			options.Sync.Add(2)
			donec := make(chan struct{})

			go func() {
				defer func() {
					c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))

					select {
						case <- time.After(options.CloseTimeout):
						case <- donec:
					}
				}()
				defer options.Sync.Done()

				for message := range in {
					err := c.WriteMessage(websocket.BinaryMessage, message.Payload)
					if err != nil {
						err = errors.Wrap(err, "Error writing message to websocket")
						log.Println(err.Error())
						return
					}
				}
			}()

			go func() {
				defer close(donec)
				defer close(out)
				defer options.Sync.Done()

				for {
					_, payload, err := c.ReadMessage()
					if err != nil {
						err = errors.Wrap(err, "Error reading meessage from websocket")
						log.Println(err.Error())
						return
					}

					SendMessage(payload, out)
				}
			}()

			options.Init = true
			options.Sync.Done()
			options.Mutex.Unlock()
			close(options.Waitc)
		}
	}
}

func (m *WebsocketServer) Init(global *GlobalFlags) (error) {
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

func (m WebsocketServer) Start() {
	waitc := make(chan struct{})
	relayc := make(chan *Message, 1)

	options := &WebsocketServerHandleOptions{
		Sync: m.sync,
		Mutex: m.mutex,
		Init: m.init,
		Waitc: waitc,
		CloseTimeout: m.closeTimeout,
		Upgrader: websocket.Upgrader{},
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", websocketServerHandle(relayc, m.out, options))

	server := &http.Server{
		Handler: mux,
	}

	m.sync.Add(1)

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
				options.Sync.Done()
				return
			case message, closed := <- m.in:
				select {
					case <- time.NewTicker(m.connectTimeout).C:
						log.Println("Connect timeout reached, let's hope somebody's connected")
					case <- waitc:
				}

				options.Mutex.Lock()
				if closed && ! options.Init {
					options.Init = true
					options.Mutex.Unlock()

					log.Println("Pipeline is shutting down and nobody connected")
					close(m.out)
					options.Sync.Done()
					return
				}

				options.Mutex.Unlock()

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

func (m WebsocketServer) Wait() {
	m.sync.Wait()

	for range m.in {}
}

func NewWebsocketServer() (Module) {
	return &WebsocketServer{
		sync: &sync.WaitGroup{},
		mutex: &sync.Mutex{},
		init: false,
	}
}
