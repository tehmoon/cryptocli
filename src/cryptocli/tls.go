package main

import (
	"sync"
	"time"
	"log"
	"crypto/tls"
	"net"
	"github.com/tehmoon/errors"
	"github.com/spf13/pflag"
	"io"
	"bytes"
	"crypto/x509"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto"
	"crypto/x509/pkix"
	"math/big"
	"encoding/pem"
	"text/template"
	"strconv"
	mathRand "math/rand"
)

func init() {
	MODULELIST.Register("tls", "TLS Server", NewTLS)
}

type TLS struct {
	addr string
	connectTimeout time.Duration
	readTimeout time.Duration
	decrypt string
	decryptTmpl *template.Template
	port int
	caCert *x509.Certificate
	caKey crypto.PrivateKey
	caFileCert string
	caFileKey string
}

type TLSRelayer struct {
	Callback MessageChannelFunc
	MessageChannel *MessageChannel
	Wg *sync.WaitGroup
}

func tlsHandler(conn net.Conn, m *TLS, relay *TLSRelayer) {
	mc, cb, wg := relay.MessageChannel, relay.Callback, relay.Wg
	defer wg.Done()

	log.Printf("Client %q is connected\n", conn.LocalAddr().String())

	wrapper := NewTLSWrapper(conn)

	log.Printf("New connection accepted from %s\n", wrapper.LocalAddr())

	var inc chan []byte
	var decrypt bool

	config := &tls.Config{
		GetConfigForClient: func(hello *tls.ClientHelloInfo) (config *tls.Config, err error) {
			metadata := map[string]interface{}{
				"local-addr": conn.RemoteAddr().String(),
				"remote-addr": conn.RemoteAddr().String(),
				"addr": m.addr,
				"servername": hello.ServerName,
				"port": m.port,
			}

			buff := bytes.NewBuffer(make([]byte, 0))
			err = m.decryptTmpl.Execute(buff, metadata)
			if err != nil {
				err = errors.Wrap(err, "Error executing template addr")
				log.Println(err.Error())
				mc.Start(nil)
				_, inc = cb()
				buff.Reset()
				return
			}

			decrypt, err = strconv.ParseBool(string(buff.Bytes()[:]))
			if err != nil {
				err = errors.Wrap(err, "Error parsing redirect flag to boolean")
				log.Println(err.Error())
				buff.Reset()
				mc.Start(nil)
				_, inc = cb()
				return
			}
			buff.Reset()

			metadata["decrypt"] = decrypt

			mc.Start(metadata)

			log.Printf("Servername: %s\n", hello.ServerName)

			_, inc = cb()

			firstPacket, err := wrapper.ReadBuffer()
			if err != nil {
				err = errors.Wrap(err, "Error reading buffer for new config")
				log.Println(err.Error())
				return nil, err
			}

			if decrypt {
				cert, key, err := TLSCreateServerCert(hello.ServerName, m.caCert, m.caKey)
				if err != nil {
					err = errors.Wrap(err, "Error creating server certificate")
					log.Println(err.Error())
					return nil, err
				}

				config = &tls.Config{
					Certificates: []tls.Certificate{tls.Certificate{
						Certificate: [][]byte{
							cert.Raw,
							m.caCert.Raw,
						},
						PrivateKey: key,
					},},
				}

				wrapper.Pivot()

				return config, nil
			}

			mc.Channel <- firstPacket
			wrapper.Pivot()

			syn := &sync.WaitGroup{}
			syn.Add(2)
			go TLSStartInc(wrapper, inc, syn)
			go TLSStartOutc(m, wrapper, mc.Channel, syn)
			syn.Wait()

			return nil, ErrTLSAbortHandshake
		},
	}

	sconn := tls.Server(wrapper, config)

	err := sconn.Handshake()
	if err != nil && err != ErrTLSAbortHandshake {
		err = errors.Wrap(err, "Error with TLS handshake")
		log.Println(err.Error())
		close(mc.Channel)
		wrapper.Close()
		DrainChannel(inc, nil)
		return
	}

	if decrypt {
		log.Println("TLS Decrypted OK")

		syn := &sync.WaitGroup{}
		syn.Add(2)
		go TLSStartInc(sconn, inc, syn)
		go TLSStartOutc(m, sconn, mc.Channel, syn)
		syn.Wait()
	}

	wrapper.Close()
}

var ErrTLSAbortHandshake = errors.New("Aborting handshake")

func TLSStartInc(writer io.WriteCloser, inc chan []byte, wg *sync.WaitGroup) {
	if wg != nil {
		defer wg.Done()
	}

	defer writer.Close()

	for payload := range inc {
		_, err := writer.Write(payload)
		if err != nil {
			err = errors.Wrap(err, "Error writing to tls connection")
			log.Println(err.Error())
			break
		}
	}

	DrainChannel(inc, nil)
}

func TLSStartOutc(m *TLS, conn net.Conn, outc chan []byte, wg *sync.WaitGroup) {
	if wg != nil {
		defer wg.Done()
	}
	defer conn.Close()
	defer close(outc)

	conn.SetReadDeadline(time.Now().Add(m.readTimeout))

	err := ReadBytesStep(conn, func(payload []byte) bool {
		outc <- payload
		conn.SetReadDeadline(time.Now().Add(m.readTimeout))

		return true
	})
	if err != nil {
		err = errors.Wrap(err, "Error reading from tcp socket")
		log.Println(err.Error())
		return
	}
}

func tlsServe(conn net.Conn, m *TLS, relayer chan *TLSRelayer, connc, donec, cancel chan struct{}) {
	donec <- struct{}{}
	defer func(donec chan struct{}) {
		<- donec
	}(donec)

	select {
		case relay, opened := <- relayer:
			if ! opened {
				return
			}

			tlsHandler(conn, m, relay)
			return
		case <- cancel:
			return
		default:
	}

	select {
		case connc <- struct{}{}:
		case <- cancel:
			return
	}

	select {
		case relay, opened := <- relayer:
			if ! opened {
				return
			}

			tlsHandler(conn, m, relay)
			return
		case <- cancel:
			return
	}
}

func (m *TLS) SetFlagSet(fs *pflag.FlagSet, args []string) {
	fs.StringVar(&m.addr, "listen", "", "Listen on addr:port. If port is 0, random port will be assigned")
	fs.DurationVar(&m.connectTimeout, "connect-timeout", 30 * time.Second, "Max amount of time to wait for a potential connection when pipeline is closing")
	fs.DurationVar(&m.readTimeout, "read-timeout", 15 * time.Second, "Amout of time to wait reading from the connection")
	fs.StringVar(&m.decrypt, "decrypt", "", "TLS intercept the handshake and replace with own CA to decrypt the traffic. Use template, return boolean false or true.")
	fs.StringVar(&m.caFileCert, "ca-cert", "", "Specify the certificate file for the CA")
	fs.StringVar(&m.caFileKey, "ca-key", "", "Specify the key file for the CA")
}

func NewTLS() (Module) {
	return &TLS{}
}

func (m *TLS) Init(in, out chan *Message, global *GlobalFlags) (err error) {
	if m.readTimeout < 1 {
		return errors.Errorf("Flag %q cannot be negative or zero", "--read-timeout")
	}

	if m.connectTimeout < 1 {
		return errors.Errorf("Flag %q cannot be negative or zero", "--connect-timeout")
	}

	rander := mathRand.New(mathRand.NewSource(time.Now().UnixNano()))

	m.decryptTmpl, err = template.New("root").Funcs(template.FuncMap{
		"rand": func(n int64) int64 {
			return rander.Int63n(n + 1)
		},
	}).Parse(m.decrypt)
	if err != nil {
		return errors.Wrap(err, "Error parsing template for \"--decrypt\" flag")
	}

	if m.caFileCert != "" && m.caFileKey == "" {
		return errors.Errorf("Flag %q is missing when flag %q is set", "--ca-key", "--ca-cert")
	}

	if m.caFileCert == "" && m.caFileKey != "" {
		return errors.Errorf("Flag %q is missing when flag %q is set", "--ca-cert", "--ca-key")
	}

	addr, err := net.ResolveTCPAddr("tcp", m.addr)
	if err != nil {
		return errors.Wrap(err, "Unable to resolve tcp address")
	}

	m.port = addr.Port

	var listener net.Listener
	listener, err = net.ListenTCP("tcp", addr)
	if err != nil {
		return errors.Wrap(err, "Unable to listen on tcp address")
	}

	if m.decrypt != "" {
		if m.caFileCert != "" && m.caFileKey != "" {
			ca, err := tls.LoadX509KeyPair(m.caFileCert, m.caFileKey)
			if err != nil {
				return errors.Wrap(err, "Unable to load CA certificate and key")
			}

			caCert, err := x509.ParseCertificate(ca.Certificate[0])
			if err != nil {
				return errors.Wrap(err, "Error parsing ca certificate from DER")
			}

			m.caCert = caCert
			m.caKey = ca.PrivateKey
			log.Println("CA certificates and key loaded")
		} else {
			m.caCert, m.caKey, err = TLSCreateCA()
			if err != nil {
				return errors.Wrap(err, "Error creating CA")
			}

			cacertPEM, cakeyPEM, err := TLSEncodeCertificateKey(m.caCert, m.caKey)
			if err != nil {
				return errors.Wrap(err, "Error displaying CA certificate and key")
			}

			log.Printf("\nCA certificate: \n%s\nCA key: \n%s\n", string(cacertPEM[:]), string(cakeyPEM[:]))
		}
	}

	log.Printf("Tcp-server listening on %s\n", listener.Addr().String())

	go func() {
		wg := &sync.WaitGroup{}
		relayer := make(chan *TLSRelayer)
		connc := make(chan struct{})
		cancel := make(chan struct{})

		donec := make(chan struct{}, global.MaxConcurrentStreams)

		go func(m *TLS, l net.Listener, relayer chan *TLSRelayer, connc, done, cancel chan struct{}) {
			for {
				conn, err := l.Accept()
				if err != nil {
					err = errors.Wrap(err, "Error accepting tcp connection")
					log.Println(err.Error())
					return
				}

				go tlsServe(conn, m, relayer, connc, donec, cancel)
			}
		}(m, listener, relayer, connc, donec, cancel)

		ticker := time.NewTicker(m.connectTimeout)

		cbs := make([]MessageChannelFunc, 0)
		mcs := make([]*MessageChannel, 0)

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

					mc := NewMessageChannel()

					out <- &Message{
						Type: MessageTypeChannel,
						Interface: mc.Callback,
					}

					if len(cbs) == 0 {
						mcs = append(mcs, mc)
						continue
					}

					wg.Add(1)

					cb := cbs[0]
					cbs = cbs[1:]

					relayer <- &TLSRelayer{
						Callback: cb,
						MessageChannel: mc,
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
							cb, ok := message.Interface.(MessageChannelFunc)
							if ok {
								if len(mcs) == 0 {
									cbs = append(cbs, cb)
									continue
								}

								wg.Add(1)
								mc := mcs[0]
								mcs = mcs[1:]

								relayer <- &TLSRelayer{
									Callback: cb,
									MessageChannel: mc,
									Wg: wg,
								}

								if ! global.MultiStreams {
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

		for _, mc := range mcs {
			close(mc.Channel)
		}

		for _, cb := range cbs {
			_, inc := cb()
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

type TLSWrapper struct {
	conn net.Conn
	buffer *bytes.Buffer
	reader io.Reader
	sync *sync.Mutex
	pivoted bool
}

func NewTLSWrapper(conn net.Conn) *TLSWrapper {
	buffer := bytes.NewBuffer([]byte{})
	reader := io.TeeReader(conn, buffer)

	wrapper := &TLSWrapper{
		conn: conn,
		buffer: buffer,
		reader: reader,
		sync: &sync.Mutex{},
		pivoted: false,
	}

	return wrapper
}

func (w *TLSWrapper) Pivot() {
	if w.pivoted {
		return
	}

	w.reader = w.conn
	w.pivoted = true
}

func (w TLSWrapper) ReadBuffer() (data []byte, err error) {
	w.sync.Lock()
	defer w.sync.Unlock()
	defer w.buffer.Reset()

	return w.buffer.Bytes(), nil
}

func (w TLSWrapper) Read(b []byte) (n int, err error) {
	w.sync.Lock()
	defer w.sync.Unlock()

	return w.reader.Read(b)
}

func (w TLSWrapper) Write(b []byte) (n int, err error) {
	return w.conn.Write(b)
}

func (w TLSWrapper) Close() error {
	return w.conn.Close()
}

func (w TLSWrapper) LocalAddr() net.Addr {
	return w.conn.LocalAddr()
}

func (w TLSWrapper) RemoteAddr() net.Addr {
	return w.conn.RemoteAddr()
}

func (w TLSWrapper) SetDeadline(t time.Time) error {
	return w.conn.SetDeadline(t)
}

func (w TLSWrapper) SetReadDeadline(t time.Time) error {
	return w.conn.SetReadDeadline(t)
}

func (w TLSWrapper) SetWriteDeadline(t time.Time) error {
	return w.conn.SetWriteDeadline(t)
}

func TLSCreateServerCert(name string, cacert *x509.Certificate, cakey crypto.PrivateKey) (cert *x509.Certificate, key *ecdsa.PrivateKey, err error) {
	key, err = ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, nil, errors.Wrap(err, "Failed to create the private key")
	}

	req, err := TLSCreateReqCert(false, name)
	if err != nil {
		return nil, nil, errors.Wrap(err, "Error create CA req certificate")
	}

	cert, err = TLSSignCertificate(req, cacert, cakey, key.Public())
	if err != nil {
		return nil, nil, errors.Wrap(err, "Error self signing CA certificate")
	}

	return cert, key, nil
}

func TLSCreateCA() (cert *x509.Certificate, key *ecdsa.PrivateKey, err error) {
	key, err = ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, nil, errors.Wrap(err, "Failed to create the private key")
	}

	req, err := TLSCreateReqCert(true, "Cryptocli trusted CA")
	if err != nil {
		return nil, nil, errors.Wrap(err, "Error create CA req certificate")
	}

	cert, err = TLSSignCertificate(req, req, key, key.Public())
	if err != nil {
		return nil, nil, errors.Wrap(err, "Error self signing CA certificate")
	}

	return cert, key, nil
}

func TLSCreateReqCert(isCA bool, name string) (req *x509.Certificate, err error) {
	now := time.Now()

	req = &x509.Certificate{
		Subject: pkix.Name{
			CommonName: name,
		},
		IsCA: isCA,
		NotBefore: now,
		NotAfter: now.Add(90 * 24 * time.Hour),
		BasicConstraintsValid: true,
	}

	if isCA {
		req.KeyUsage = x509.KeyUsageCRLSign | x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature
		req.MaxPathLen = 0
	} else {
		req.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth,}
	}

	req.SubjectKeyId = make([]byte, 20)

	_, err = io.ReadFull(rand.Reader, req.SubjectKeyId)
	if err != nil {
		return nil, errors.Wrap(err, "Error generating subject key id")
	}

	sn := make([]byte, 20)

	_, err = io.ReadFull(rand.Reader, sn)
	if err != nil {
		return nil, errors.Wrap(err, "Error generating serial number")
	}

	req.SerialNumber = new(big.Int).SetBytes(sn)

	return req, nil
}

func TLSSignCertificate(template, req *x509.Certificate, priv crypto.PrivateKey, pub crypto.PublicKey) (*x509.Certificate, error) {
	raw, err := x509.CreateCertificate(rand.Reader, template, req, pub, priv)
	if err != nil {
		return nil, errors.Wrap(err, "Error signing the certificate's template")
	}

	cert, err := x509.ParseCertificate(raw)
	if err != nil {
		return nil, errors.Wrap(err, "Error parsing the newly created certificate")
	}

	return cert, nil
}

func TLSEncodeCertificate(cert *x509.Certificate) ([]byte) {
	b := &pem.Block{
		Type: "CERTIFICATE",
		Bytes: cert.Raw,
	}

	return pem.EncodeToMemory(b)
}

func TLSEncodePrivateKey(priv crypto.PrivateKey) ([]byte, error) {
	var (
		data []byte
		err error
		t string
	)

	switch p := priv.(type) {
		case *ecdsa.PrivateKey:
			data, err = x509.MarshalECPrivateKey(p)
			if err != nil {
				return nil, err
			}

			t = "EC PRIVATE KEY"

		default:
			return nil, errors.Errorf("Unsupported private key of type %T", p)
	}

	b := &pem.Block{
		Type: t,
		Bytes: data,
	}

	return pem.EncodeToMemory(b), nil
}

func TLSEncodeCertificateKey(cert *x509.Certificate, key crypto.PrivateKey) (certPEM, keyPEM []byte, err error) {
	certPEM = TLSEncodeCertificate(cert)

	keyPEM, err = TLSEncodePrivateKey(key)
	if err != nil {
		return nil, nil, errors.Wrap(err, "Error encoding private key in PEM format")
	}

	return certPEM, keyPEM, nil
}
