package main

import (
	"sync"
	"github.com/spf13/pflag"
	"github.com/tehmoon/errors"
	"log"
	"encoding/hex"
)

func init() {
	MODULELIST.Register("hex", "Hex encoding/decoding", NewHex)
}

type Hex struct {
	encode bool
	decode bool
}

func (m *Hex) Init(in, out chan *Message, global *GlobalFlags) (error) {
	if (m.encode && m.decode) || (! m.encode && ! m.decode) {
		return errors.Errorf("One of %q and %q must be provided", "encode", "decode")
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
						if m.encode {
							go startHexEncode(cb, mc, wg)
						} else {
							go startHexDecode(cb, mc, wg)
						}

						if ! global.MultiStreams {
							if ! init {
								close(mc.Channel)
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

// TODO: limit the buffer size because it allocates * 2 right now.
func startHexEncode(cb MessageChannelFunc, mc *MessageChannel, wg *sync.WaitGroup) {
	mc.Start(nil)
	_, inc := cb()
	outc := mc.Channel

	for payload := range inc {
		buff := make([]byte, hex.EncodedLen(len(payload)))
		hex.Encode(buff, payload)

		outc <- buff
	}

	close(outc)
	wg.Done()
}

func startHexDecode(cb MessageChannelFunc, mc *MessageChannel, wg *sync.WaitGroup) {
	mc.Start(nil)
	_, inc := cb()
	outc := mc.Channel

	var (
		crumb byte
		set bool
		buff []byte
	)

	for payload := range inc {
		l := len(payload)

		// If we have data, there are 4 states:
		//	- we have even number of bytes and no bytes from the previous message
		//	- even number of bytes and one byte left from the previous message
		//	- odd number of bytes and no bytes from the previous message
		//	- odd number of bytes and one byte left from the previous message
		if l != 0 {
			if l % 2 == 0 && ! set {
			} else if l % 2 == 0 && set {
				payload = append([]byte{crumb,}, payload[:l - 1]...)
				crumb = payload[l - 1]

			} else if l % 2 != 0 && set {
				payload = append([]byte{crumb,}, payload[:]...)
				set = false

			} else {
				crumb = payload[l - 1]
				payload = payload[:l - 1]

				set = true
			}

			buff = make([]byte, hex.DecodedLen(len(payload)))
			_, err := hex.Decode(buff, payload)
			if err != nil {
				err = errors.Wrap(err, "Error decoding hex")
				log.Println(err.Error())
				break
			}

			outc <- buff
		}
	}

	close(outc)
	DrainChannel(inc, nil)
	wg.Done()
}

func NewHex() (Module) {
	return &Hex{}
}

func (m *Hex) SetFlagSet(fs *pflag.FlagSet, args []string) {
	fs.BoolVar(&m.encode, "encode", false, "Hexadecimal encode")
	fs.BoolVar(&m.decode, "decode", false, "Hexadecimal decode")
}
