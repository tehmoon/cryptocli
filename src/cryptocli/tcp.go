package main

import (
	"crypto/tls"
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
	conn net.Conn
	tls string
	insecure bool
}

func (m *TCP) SetFlagSet(fs *pflag.FlagSet) {
	fs.StringVar(&m.addr, "addr", "", "Tcp address to connect to")
	fs.StringVar(&m.tls, "tls", "", "Use TLS with servername in client hello")
	fs.BoolVar(&m.insecure, "insecure", false, "Don't verify certificate chain when \"--tls\" is set")
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

	if m.tls != "" || m.insecure {
		config := &tls.Config{
			InsecureSkipVerify: m.insecure,
			ServerName: m.tls,
		}

		m.conn = tls.Client(m.conn, config)
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

func tcpStartIn(conn net.Conn, in chan *Message, wg *sync.WaitGroup) {
	defer conn.Close()
	defer wg.Done()

	for message := range in {
		_, err := conn.Write(message.Payload)
		if err != nil {
			log.Println(errors.Wrap(err, "Error writing to tcp connection in tcp"))
			return
		}
	}
}

func tcpStartOut(conn net.Conn, out chan *Message, wg *sync.WaitGroup) {
	defer close(out)
	defer wg.Done()

	err := ReadBytesSendMessages(conn, out)
	if err != nil {
		log.Println(errors.Wrap(err, "Error reading tcp connection in tcp"))
		return
	}
}

func NewTCP() (Module) {
	return &TCP{
		sync: &sync.WaitGroup{},
	}
}
