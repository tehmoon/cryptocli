package main

import (
	"time"
	"sync"
	"log"
	"crypto/tls"
	"net"
	"github.com/tehmoon/errors"
	"github.com/spf13/pflag"
)

func init() {
	MODULELIST.Register("tcp-server", "Listens TCP and wait for a single connection to complete", NewTCPServer)
}

type TCPServer struct {
	in chan *Message
	out chan *Message
	sync *sync.WaitGroup
	addr string
	listener net.Listener
	cert string
	key string
	connectTimeout time.Duration
}

func (m *TCPServer) Init(global *GlobalFlags) (error) {
	if m.connectTimeout < 1 {
		return errors.Errorf("Flag %q cannot be negative or zero", "--connect-timeout")
	}

	if m.cert != "" && m.key == "" {
		return errors.Errorf("Flag %q is missing when flag %q is set", "--key", "--certiticate")
	}

	if m.key != "" && m.cert == "" {
		return errors.Errorf("Flag %q is missing when flag %q is set", "--certificate", "--key")
	}

	addr, err := net.ResolveTCPAddr("tcp", m.addr)
	if err != nil {
		return errors.Wrap(err, "Unable to resolve tcp address")
	}

	m.listener, err = net.ListenTCP("tcp", addr)
	if err != nil {
		return errors.Wrap(err, "Unable to listen on tcp address")
	}

	if m.cert != "" && m.key != "" {
		pem, err := tls.LoadX509KeyPair(m.cert, m.key)
		if err != nil {
			return errors.Wrap(err, "Error loading x509 key pair from files")
		}

		config := &tls.Config{
			Certificates: []tls.Certificate{pem,},
		}

		m.listener = tls.NewListener(m.listener, config)
	}

	log.Printf("Tcp-server listening on %s with TLS enabled: %t\n", m.listener.Addr().String(), m.key != "" && m.cert != "")

	return nil
}

func (m TCPServer) Start() {
	m.sync.Add(2)

	go func() {
		var conn net.Conn
		var err error

		wait := make(chan struct{})

		go func() {
			conn, err = m.listener.Accept()
			if err != nil {
				return
			}

			close(wait)
		}()

		// Channel to relay messages. It has a buffer of 1
		// because the first message must not be blocking
		relayc := make(chan *Message, 1)

		select {
			case <- wait:
			case <- time.NewTicker(m.connectTimeout).C:
				log.Println("Connect timeout reached, nobody connected and no inputs were sent")

				close(relayc)
				close(m.out)
				m.sync.Done()
				m.sync.Done()
				return
			case message, closed := <- m.in:
				log.Println("Message receive before connection got accepted")

				select {
					case <- time.NewTicker(m.connectTimeout).C:
						log.Println("Connect timeout reached, let's hope somebody's connected")
					case <- wait:
				}

				if closed && conn == nil {
					log.Println("Pipeline is shutting down and nobody connected")
					close(relayc)
					close(m.out)
					m.sync.Done()
					m.sync.Done()
					return
				}

				relayc <- message
		}

		go func() {
			for message := range m.in {
				relayc <- message
			}

			close(relayc)
		}()

		go tcpServerStartIn(conn, relayc, m.sync)
		go tcpServerStartOut(conn, m.out, m.sync)
	}()
}

func tcpServerStartIn(conn net.Conn, in chan *Message, wg *sync.WaitGroup) {
	defer conn.Close()
	defer wg.Done()

	for message := range in {
		_, err := conn.Write(message.Payload)
		if err != nil {
			log.Println(errors.Wrap(err, "Error writing to tcp connection in tcp-server"))
			return
		}
	}
}

func tcpServerStartOut(conn net.Conn, out chan *Message, wg *sync.WaitGroup) {
	defer close(out)
	defer wg.Done()

	err := ReadBytesSendMessages(conn, out)
	if err != nil {
		log.Println(errors.Wrap(err, "Error reading tcp connection in tcp-server"))
		return
	}
}

func (m TCPServer) Wait() {
	m.sync.Wait()

	for range m.in {}
}

func (m *TCPServer) In(in chan *Message) (chan *Message) {
	m.in = in

	return in
}

func (m *TCPServer) Out(out chan *Message) (chan *Message) {
	m.out = out

	return out
}

func (m *TCPServer) SetFlagSet(fs *pflag.FlagSet) {
	fs.StringVar(&m.addr, "listen", "", "Listen on addr:port. If port is 0, random port will be assigned")
	fs.StringVar(&m.cert, "certificate", "", "Path to certificate in PEM format")
	fs.StringVar(&m.key, "key", "", "Path to private key in PEM format")
	fs.DurationVar(&m.connectTimeout, "connect-timeout", 30 * time.Second, "Max amount of time to wait for a potential connection when pipeline is closing")
}

func NewTCPServer() (Module) {
	return &TCPServer{
		sync: &sync.WaitGroup{},
	}
}
