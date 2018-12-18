package main

import (
	"sync"
	"github.com/spf13/pflag"
	"github.com/tehmoon/errors"
	"log"
	"encoding/hex"
)

func init() {
	MODULELIST.Register("hex", "Hex de-compress", NewHex)
}

type Hex struct {
	in chan *Message
	out chan *Message
	wg *sync.WaitGroup
	encode bool
	decode bool
}

func (m *Hex) Init(global *GlobalFlags) (error) {
	if (m.encode && m.decode) || (! m.encode && ! m.decode) {
		return errors.Errorf("One of %q and %q must be provided", "encode", "decode")
	}

	return nil
}

func (m Hex) Start() {
	m.wg.Add(1)

	if m.encode {
		go startHexEncode(m.in, m.out, m.wg)
		return
	}

	if m.decode {
		go startHexDecode(m.in, m.out, m.wg)
		return
	}

	log.Fatal(errors.New("Code path shouldn't be reached in hex module"))
}

// TODO: limit the buffer size because it allocates * 2 right now.
func startHexEncode(in, out chan *Message, wg *sync.WaitGroup) {
	for message := range in {
		buff := make([]byte, hex.EncodedLen(len(message.Payload)))
		hex.Encode(buff, message.Payload)
		SendMessage(buff, out)
	}

	close(out)
	wg.Done()
}

func hexDecodeMessages(in, out chan *Message) (error) {
	var (
		crumb byte
		set bool
		buff []byte
		payload []byte
	)

	for message := range in {
		l := len(message.Payload)

		// If we have data, there are 4 states:
		//	- we have even number of bytes and no bytes from the previous message
		//	- even number of bytes and one byte left from the previous message
		//	- odd number of bytes and no bytes from the previous message
		//	- odd number of bytes and one byte left from the previous message
		if l != 0 {
			if l % 2 == 0 && ! set {
				payload = message.Payload

			} else if l % 2 == 0 && set {
				payload = append([]byte{crumb,}, message.Payload[:l - 1]...)
				crumb = message.Payload[l - 1]

			} else if l % 2 != 0 && set {
				payload = append([]byte{crumb,}, message.Payload[:]...)
				set = false

			} else {
				crumb = message.Payload[l - 1]
				payload = message.Payload[:l - 1]

				set = true
			}

			buff = make([]byte, hex.DecodedLen(len(payload)))
			_, err := hex.Decode(buff, payload)
			if err != nil {
				return err
			}

			SendMessage(buff, out)
		}
	}

	return nil
}

func startHexDecode(in, out chan *Message, wg *sync.WaitGroup) {
	err := hexDecodeMessages(in, out)
	if err != nil {
		log.Println(errors.Wrap(err, "Error decoding hex in hex module"))
	}

	close(out)
	wg.Done()
}

func (m Hex) Wait() {
	m.wg.Wait()

	for range m.in {}
}

func (m *Hex) In(in chan *Message) (chan *Message) {
	m.in = in

	return in
}

func (m *Hex) Out(out chan *Message) (chan *Message) {
	m.out = out

	return out
}

func NewHex() (Module) {
	return &Hex{
		wg: &sync.WaitGroup{},
	}
}

func (m *Hex) SetFlagSet(fs *pflag.FlagSet) {
	fs.BoolVar(&m.encode, "encode", false, "Hexadecimal encode")
	fs.BoolVar(&m.decode, "decode", false, "Hexadecimal decode")
}
