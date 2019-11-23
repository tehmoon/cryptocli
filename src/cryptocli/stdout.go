package main

import (
	"sync"
	"os"
	"github.com/tehmoon/errors"
	"github.com/spf13/pflag"
)

var stdoutMutex = struct{sync.Mutex; Init bool}{Init: false,}

func init() {
	MODULELIST.Register("stdout", "Writes to stdout", NewStdout)
}

type Stdout struct {}

func (m Stdout) Init(in, out chan *Message, global *GlobalFlags) (error) {
	stdoutMutex.Lock()
	defer stdoutMutex.Unlock()
	defer func() {
		stdoutMutex.Init = false
	}()

	if stdoutMutex.Init {
		return errors.New("Module \"stdout\" cannot be added more than once")
	}

	stdoutMutex.Init = true

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
						go func(cb MessageChannelFunc, mc *MessageChannel, wg *sync.WaitGroup) {
							mc.Start(nil)
							_, inc := cb()

							for payload := range inc {
								os.Stdout.Write(payload)
								os.Stdout.Sync()
							}

							close(mc.Channel)
							wg.Done()
						}(cb, mc, wg)

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

func NewStdout() (Module) {
	return &Stdout{}
}

func (m Stdout) SetFlagSet(fs *pflag.FlagSet, args []string) {}
