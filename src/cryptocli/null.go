package main

import (
	"github.com/spf13/pflag"
	"sync"
)

func init() {
	MODULELIST.Register("null", "Discard all incoming data", NewNull)
}

type Null struct {}

func (m Null) Init(in, out chan *Message, global *GlobalFlags) (error) {
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
						go func() {
							defer wg.Done()

							mc.Start(nil)
							_, inc := cb()

							close(mc.Channel)
							DrainChannel(inc, nil)
						}()

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

func NewNull() (Module) {
	return &Null{}
}

func (m *Null) SetFlagSet(fs *pflag.FlagSet, args []string) {}
