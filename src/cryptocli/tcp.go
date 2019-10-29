package main

import (
	"time"
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
	addr string
	servername string
	insecure bool
	readTimeout time.Duration
}

func (m *TCP) SetFlagSet(fs *pflag.FlagSet, args []string) {
	fs.StringVar(&m.addr, "addr", "", "Tcp address to connect to")
	fs.StringVar(&m.servername, "tls", "", "Use TLS with servername in client hello")
	fs.BoolVar(&m.insecure, "insecure", false, "Don't verify certificate chain when \"--servername\" is set")
	fs.DurationVar(&m.readTimeout, "read-timeout", 3 * time.Second, "Read timeout for the tcp connection")
}

func (m *TCP) Init(in, out chan *Message, global *GlobalFlags) (error) {
	if m.readTimeout <= 0 {
		return errors.Errorf("Flag %q has to be greater that 0", "--read-timeout")
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

						wg.Add(1)
						go tcpStartHandler(m, inc, outc, wg)

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

func tcpStartHandler(m *TCP, inc, outc MessageChannel, wg *sync.WaitGroup) {
	defer wg.Done()

	addr, err := net.ResolveTCPAddr("tcp", m.addr)
	if err != nil {
		err = errors.Wrap(err, "Unable to resolve tcp address")
		log.Println(err.Error())
		close(outc)
		DrainChannel(inc, nil)
		return
	}

	var conn net.Conn
	conn, err = net.DialTCP("tcp", nil, addr)
	if err != nil {
		err = errors.Wrap(err, "Fail to dial tcp")
		log.Println(err.Error())
		close(outc)
		DrainChannel(inc, nil)
		return
	}

	if m.servername != "" || m.insecure {
		config := &tls.Config{
			InsecureSkipVerify: m.insecure,
			ServerName: m.servername,
		}

		conn = tls.Client(conn, config)
	}

	syn := &sync.WaitGroup{}
	syn.Add(2)

	go tcpStartIn(conn, inc, syn)
	go tcpStartOut(conn, outc, m.readTimeout, syn)

	syn.Wait()
	conn.Close()
}

func tcpStartIn(conn net.Conn, inc MessageChannel, wg *sync.WaitGroup) {
	defer wg.Done()

	for payload := range inc {
		_, err := conn.Write(payload)
		if err != nil {
			err = errors.Wrap(err, "Error writing to tcp connection in tcp")
			log.Println(err.Error())
			break
		}
	}

	DrainChannel(inc, nil)
}

func tcpStartOut(conn net.Conn, outc MessageChannel, timeout time.Duration, wg *sync.WaitGroup) {
	defer wg.Done()
	defer close(outc)
	defer conn.Close()

	conn.SetReadDeadline(time.Now().Add(timeout))

	err := ReadBytesStep(conn, func(payload []byte) (bool) {
		outc <- payload
		conn.SetReadDeadline(time.Now().Add(timeout))

		return true
	})
	if err != nil {
		err = errors.Wrap(err, "Error reading tcp connection in tcp")
		log.Println(err.Error())
		return
	}
}

func NewTCP() (Module) {
	return &TCP{}
}
