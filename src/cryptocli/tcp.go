package main

import (
	"net"
	"github.com/tehmoon/errors"
	"github.com/spf13/pflag"
	"sync"
	"log"
)

func init() {
	MODULELIST.Register("tcp", "Connects to TCP", NewTCP)
}

type TCP struct {
	in chan *Message
	out chan *Message
	sync *sync.WaitGroup
	addr string
	conn *net.TCPConn
}

func (m *TCP) SetFlagSet(fs *pflag.FlagSet) {
	fs.StringVar(&m.addr, "addr", "", "Tcp address to connect to")
}

func (m *TCP) In(in chan *Message) (chan *Message) {
	m.in = in

	return in
}

func (m *TCP) Out(out chan *Message) (chan *Message) {
	m.out = out

	return out
}

func (m *TCP) Init(global *GlobalFlags) (error) {
	addr, err := net.ResolveTCPAddr("tcp", m.addr)
	if err != nil {
		return errors.Wrap(err, "Unable to resolve tcp address")
	}

	m.conn, err = net.DialTCP("tcp", nil, addr)
	if err != nil {
		return errors.Wrap(err, "Fail to dial tcp")
	}

	log.Printf("Tcp connection established to %s\n", addr.String())

	return nil
}

func (m TCP) Start() {
	m.sync.Add(2)

	go func() {
		go tcpStartIn(m.conn, m.in, m.sync)
		go tcpStartOut(m.conn, m.out, m.sync)
	}()
}

func (m TCP) Wait() {
	m.sync.Wait()

	for range m.in {}
}

func tcpStartIn(conn *net.TCPConn, in chan *Message, wg *sync.WaitGroup) {
	for message := range in {
		_, err := conn.Write(message.Payload)
		if err != nil {
			log.Println(errors.Wrap(err, "Error writing to tcp connection in tcp"))
			break
		}
	}

	conn.CloseRead()
	wg.Done()
}

func tcpStartOut(conn *net.TCPConn, out chan *Message, wg *sync.WaitGroup) {
	err := ReadBytesSendMessages(conn, out)
	if err != nil {
		log.Println(errors.Wrap(err, "Error reading tcp connection in tcp"))
	}

	conn.CloseWrite()
	close(out)
	wg.Done()
}

func NewTCP() (Module) {
	return &TCP{
		sync: &sync.WaitGroup{},
	}
}
