package main

import (
	"time"
	"crypto/tls"
	"net"
	"github.com/tehmoon/errors"
	"github.com/spf13/pflag"
	"sync"
	"log"
	"text/template"
	"bytes"
)

func init() {
	MODULELIST.Register("tcp", "Connects to TCP", NewTCP)
}

type TCP struct {
	addr string
	servername string
	insecure bool
	readTimeout time.Duration
	tplAddr *template.Template
	tplTLS *template.Template
}

func (m *TCP) SetFlagSet(fs *pflag.FlagSet, args []string) {
	fs.StringVar(&m.addr, "addr", "", "Tcp address to connect to")
	fs.StringVar(&m.servername, "tls", "", "Use TLS with servername in client hello")
	fs.BoolVar(&m.insecure, "insecure", false, "Don't verify certificate chain when \"--servername\" is set")
	fs.DurationVar(&m.readTimeout, "read-timeout", 3 * time.Second, "Read timeout for the tcp connection")
}

func (m *TCP) Init(in, out chan *Message, global *GlobalFlags) (err error) {
	if m.readTimeout <= 0 {
		return errors.Errorf("Flag %q has to be greater that 0", "--read-timeout")
	}

	m.tplAddr, err = template.New("root").Parse(m.addr)
	if err != nil {
		return errors.Wrap(err, "Error parsing template for \"--addr\" flag")
	}

	m.tplTLS, err = template.New("root").Parse(m.servername)
	if err != nil {
		return errors.Wrap(err, "Error parsing tepmlate for \"--tls\" flag")
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
						go tcpStartHandler(m, cb, mc, wg)

						if ! global.MultiStreams {
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

func tcpStartHandler(m *TCP, cb MessageChannelFunc, mc *MessageChannel, wg *sync.WaitGroup) {
	defer wg.Done()

	mc.Start(nil)
	metadata, inc := cb()

	buff := bytes.NewBuffer(make([]byte, 0))
	err := m.tplAddr.Execute(buff, metadata)
	if err != nil {
		err = errors.Wrap(err, "Error executing template addr")
		log.Println(err.Error())
		close(mc.Channel)
		DrainChannel(inc, nil)
		return
	}
	addr := string(buff.Bytes()[:])
	buff.Reset()

	err = m.tplTLS.Execute(buff, metadata)
	if err != nil {
		err = errors.Wrap(err, "Error executing template tls")
		log.Println(err.Error())
		close(mc.Channel)
		DrainChannel(inc, nil)
		return
	}
	servername := string(buff.Bytes()[:])
	buff.Reset()

	outc := mc.Channel

	a, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		err = errors.Wrap(err, "Unable to resolve tcp address")
		log.Println(err.Error())
		close(outc)
		DrainChannel(inc, nil)
		return
	}

	var conn net.Conn
	conn, err = net.DialTCP("tcp", nil, a)
	if err != nil {
		err = errors.Wrap(err, "Fail to dial tcp")
		log.Println(err.Error())
		close(outc)
		DrainChannel(inc, nil)
		return
	}

	if servername != "" || m.insecure {
		config := &tls.Config{
			InsecureSkipVerify: m.insecure,
			ServerName: servername,
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

func tcpStartIn(conn net.Conn, inc chan []byte, wg *sync.WaitGroup) {
	defer wg.Done()
	defer conn.Close()

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

func tcpStartOut(conn net.Conn, outc chan []byte, timeout time.Duration, wg *sync.WaitGroup) {
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
