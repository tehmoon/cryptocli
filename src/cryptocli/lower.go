package main

import (
	"github.com/spf13/pflag"
	"sync"
)

func init() {
	MODULELIST.Register("lower", "Lowercase all ascii characters", NewLower)
}

type Lower struct {}

func (m Lower) Init(in, out chan *Message, global *GlobalFlags) (error) {
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
						go startLower(cb, mc, wg)
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

func NewLower() (Module) {
	return &Lower{}
}

func startLower(cb MessageChannelFunc, mc *MessageChannel, wg *sync.WaitGroup) {
	mc.Start(nil)
	_, inc := cb()
	outc := mc.Channel

	for payload := range inc {
		for i, b := range payload {
			if b > 64 && b < 91 {
				payload[i] = b + 32
			}
		}
		outc <- payload
	}

	close(outc)
	wg.Done()
}

func (m *Lower) SetFlagSet(fs *pflag.FlagSet, args []string) {}
