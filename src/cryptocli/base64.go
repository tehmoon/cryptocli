package main

import (
	"bytes"
	"sync"
	"github.com/spf13/pflag"
	"io"
	"encoding/base64"
	"log"
	"github.com/tehmoon/errors"
)

func init() {
	MODULELIST.Register("base64", "Base64 decode or encode", NewBase64)
}

type Base64 struct {
	in chan *Message
	out chan *Message
	decode bool
	encode bool
	wg *sync.WaitGroup
}

func (m Base64) Init(global *GlobalFlags) (error) {
	if (m.decode && m.encode) || (! m.decode && ! m.encode) {
		return errors.Errorf("One of %q and %q is required", "encode", "decode")
	}

	return nil
}

func (m Base64) Start() {
	m.wg.Add(1)

	if m.decode && ! m.encode {
		go startBase64Decode(m.in, m.out, m.wg)
		return
	}

	if ! m.decode && m.encode {
		go startBase64Encode(m.in, m.out, m.wg)
		return
	}

	log.Fatal("Should not be reached in base64 module")
}

func startBase64Decode(in, out chan *Message, wg *sync.WaitGroup) {
	reader, writer := io.Pipe()
	b64 := base64.NewDecoder(base64.StdEncoding, reader)

	wg.Add(1)

	go func() {
		for message := range in {
			_, err := writer.Write(message.Payload)
			if err != nil {
				log.Println(errors.Wrap(err, "Error writing data to pipe in base64"))
				break
			}
		}

		writer.Close()
	}()

	go func() {
		err := ReadBytesSendMessages(b64, out)
		if err != nil {
			log.Println(errors.Wrap(err, "Error reading base64 reader in base64"))
		}

		close(out)
		wg.Done()
	}()

	wg.Done()
}

func startBase64Encode(in, out chan *Message, wg *sync.WaitGroup) {
	c := make(chan []byte, 0)

	wg.Add(2)

	go startBase64EncodeIn(in, c, wg)
	go startBase64EncodeOut(out, c, wg)

	wg.Done()
}

func startBase64EncodeOut(out chan *Message, c chan []byte, wg *sync.WaitGroup) {
	for payload := range c {
		SendMessage(payload, out)
	}

	close(out)

	wg.Done()
}

func startBase64EncodeIn(in chan *Message, c chan []byte, wg *sync.WaitGroup) {
	buff := bytes.NewBuffer(nil)
	writer := base64.NewEncoder(base64.StdEncoding, buff)

	for message := range in {
		_, err := writer.Write(message.Payload)
		if err != nil {
			break
		}

		c <- CopyResetBuffer(buff)
	}

	writer.Close()
	c <- CopyResetBuffer(buff)
	close(c)

	wg.Done()
}

func (m Base64) Wait() {
	m.wg.Wait()

	for range m.in {}
}

func (m *Base64) In(in chan *Message) (chan *Message) {
	m.in = in

	return in
}

func (m *Base64) Out(out chan *Message) (chan *Message) {
	m.out = out

	return out
}

func NewBase64() (Module) {
	return &Base64{
		wg: &sync.WaitGroup{},
	}
}

func (m *Base64) SetFlagSet(fs *pflag.FlagSet) {
	fs.BoolVar(&m.decode, "decode", false, "Base64 decode")
	fs.BoolVar(&m.encode, "encode", false, "Base64 encode")
}
