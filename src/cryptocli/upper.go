package main

import (
	"github.com/spf13/pflag"
	"sync"
)

func init() {
	MODULELIST.Register("upper", "Uppercase all ascii characters", NewUpper)
}

type Upper struct {}

func (m Upper) Init(in, out chan *Message, global *GlobalFlags) (error) {
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
						go startUpper(cb, mc, wg)

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

func NewUpper() (Module) {
	return &Upper{}
}

func startUpper(cb MessageChannelFunc, mc *MessageChannel, wg *sync.WaitGroup) {
	mc.Start(nil)
	_, inc := cb()
	outc := mc.Channel

	for payload := range inc {
		for i, b := range payload {
			if b > 96 && b < 123 {
				payload[i] = b - 32
			}
		}

		outc <- payload
	}

	close(outc)
	wg.Done()
}

func (m *Upper) SetFlagSet(fs *pflag.FlagSet, args []string) {}
