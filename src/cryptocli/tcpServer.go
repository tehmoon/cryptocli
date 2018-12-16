package main

import (
	"sync"
	"log"
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
	listener *net.TCPListener
	line bool
}

func (m *TCPServer) Init(global *GlobalFlags) (error) {
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

	log.Printf("Tcp-server listening on %s\n", ln.Addr().String())

	m.listener = ln

	return nil
}

func (m TCPServer) Start() {
	m.sync.Add(2)

	go func() {
		conn, err := m.listener.AcceptTCP()
		if err != nil {
			close(m.out)
			return
		}

		go tcpServerStartIn(conn, m.in, m.sync)
		go tcpServerStartOut(conn, m.out, m.line, m.sync)
	}()
}

func tcpServerStartIn(conn *net.TCPConn, in chan *Message, wg *sync.WaitGroup) {
	for message := range in {
		_, err := conn.Write(message.Payload)
		if err != nil {
			log.Println(errors.Wrap(err, "Error writing to tcp connection in tcp-server"))
			break
		}
	}

	conn.CloseRead()
	wg.Done()
}

func tcpServerStartOut(conn *net.TCPConn, out chan *Message, line bool, wg *sync.WaitGroup) {
	var err error

	cb := func(payload []byte) {
		SendMessage(payload, out)
	}

	if line {
		err = ReadDelimStep(conn, '\n', cb)
	} else {
		err = ReadBytesStep(conn, cb)
	}

	if err != nil {
		log.Println(errors.Wrap(err, "Error reading tcp connection in tcp-server"))
	}

	conn.CloseWrite()
	close(out)
	wg.Done()
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
	fs.BoolVar(&m.line, "line", false, "Read lines from the socket")
}

func NewTCPServer() (Module) {
	return &TCPServer{
		sync: &sync.WaitGroup{},
	}
}
