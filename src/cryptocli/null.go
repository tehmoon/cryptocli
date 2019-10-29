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

		LOOP: for message := range in {
			switch message.Type {
				case MessageTypeTerminate:
					wg.Wait()
					out <- message
					break LOOP
				case MessageTypeChannel:
					inc, ok := message.Interface.(MessageChannel)
					if ok {
						outc := make(MessageChannel)

						out <- &Message{
							Type: MessageTypeChannel,
							Interface: outc,
						}
						wg.Add(1)
						go DrainChannel(inc, wg)
						close(outc)
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
